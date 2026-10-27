package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/services/billing"
	"github.com/brinestone/mogtrade/services/matching"
	"github.com/brinestone/mogtrade/services/orders"
	"github.com/brinestone/mogtrade/web/helpers"
	eventpayloads "github.com/brinestone/mogtrade/web/payloads/events"
	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

const (
	EventKeyOrderCreatedV1 string = "order.created.v1"
)

var (
	ErrOrderRejected error = errors.New("order has been rejected")
)

type Orders struct {
	repo           *db.Queries
	logger         *slog.Logger
	riskEngine     *orders.RiskEngine
	pool           *pgxpool.Pool
	matchingEngine *matching.Engine
	idGenerator    contract.IdFactory
	eb             contract.EventBus
	cc             contract.CurrencyConverter
}

func (o *Orders) bootstrapMatchingEngine(ctx context.Context) {
	o.logger.Info("loading open orders into matching engine")

	var cursor string = "00000000000000000000000000"
	const batchSize = 1000
	var batches = 0
	var added = 0
	for {
		batch, err := o.repo.FindOpenOrders(ctx, db.FindOpenOrdersParams{
			ID:    cursor,
			Limit: batchSize,
		})
		if err != nil {
			o.logger.Error("could not fetchc batch of orders", "cursor", cursor, "batchSize", batchSize)
			return
		}

		if len(batch) == 0 {
			break
		}

		cursor = batch[len(batch)-1].ID
		for _, order := range batch {
			o.matchingEngine.PlaceMatchOrder(matching.PlaceMatchOrderParams{
				OrderId:  order.ID,
				Symbol:   order.Symbol,
				Type:     order.Side,
				Quantity: order.Quantity,
				Price:    order.LimitPrice.Decimal,
			})
			added++
		}
		batches++
	}
	o.logger.Info("orders added to matching engine", "added", added, "batches", batches, "batchSize", batchSize)
}

func (o *Orders) handlePlaceOrder(ctx *gin.Context) {
	orderId, tracingId := o.idGenerator(), o.idGenerator()
	logger := o.logger.With("user", helpers.GetCurrentUserId(ctx), "orderId", orderId, "tracingId", tracingId)
	var payload httppayloads.PlaceOrderPayload

	o.logger.Info("handling request to place orders, validating request")
	if err := ctx.ShouldBind(&payload); err != nil {
		logger.Warn("form binding validation failed, aborting")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctx.ShouldBindHeader(&payload); err != nil {
		logger.Warn("header binding validation failed, aborting")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	errMsgs := payload.Validate()
	if len(errMsgs) > 0 {
		logger.Warn("order request failed validation. Aborting request")
		ctx.JSON(http.StatusBadRequest, gin.H{"errors": errMsgs})
		return
	}

	err := helpers.WithTransactionRepo(ctx.Request.Context(), func(q *db.Queries) error {
		logger.Debug("getting wallet snapshot", "type", payload.WalletType)
		wallet, err := billing.GetCurrentWalletSnapshot(ctx.Request.Context(), q, billing.GetWalletSnapshotParams{
			Type:    payload.WalletType,
			OwnerId: helpers.GetCurrentUserId(ctx),
		})
		if err != nil {
			return err
		}
		systemWallet := helpers.GetSystemWalletByType(wallet.WalletType)
		logger.Debug("getting exchange rates", "currency", payload.Currency)
		exchangeRates, err := helpers.GetDefaultExchangeRateDecimalFor(ctx, payload.Currency)
		if err != nil {
			return err
		}
		logger.Debug("creating order record")
		err = orders.PlaceOrder(ctx.Request.Context(), q, orders.PlaceOrderParams{
			TracingId:            tracingId,
			OrderId:              orderId,
			WalletId:             wallet.WalletID,
			WalletTransactionId:  o.idGenerator(),
			SystemWalletId:       systemWallet,
			PlacedBy:             helpers.GetCurrentUserId(ctx),
			Currency:             payload.Currency,
			ExchangeRateSnapshot: exchangeRates[0],
			Symbol:               payload.Symbol,
			IdempotencyToken:     payload.IdempotencyToken,
			Side:                 db.OrderSide(payload.Side),
			Type:                 db.OrderType(payload.Type),
			Quantity:             decimal.NewFromFloat32(payload.Quantity),
			LimitPrice:           payload.LimitPrice,
			StopPrice:            payload.StopPrice,
		})
		if err != nil {
			return err
		}

		result := o.riskEngine.ValidateOrder(ctx.Copy().Request.Context(), orders.OrderContext{
			Symbol:         payload.Symbol,
			Side:           payload.Side,
			OrderType:      payload.Type,
			Quantity:       decimal.NewFromFloat32(payload.Quantity),
			LimitPrice:     payload.LimitPrice,
			StopPrice:      payload.StopPrice,
			AccountBalance: wallet.CurrentBalance,
		})
		if !result.Approved {
			err = orders.RejectOrder(ctx.Request.Context(), q, orders.RejectOrderParams{
				OrderId: orderId,
				OwnerId: helpers.GetCurrentUserId(ctx),
			})
			if err != nil {
				return err
			}
			return fmt.Errorf("%w: risk checks failed due to reason: %s", ErrOrderRejected, result.Reason)
		} else {
			err = orders.SubmitOrder(ctx.Request.Context(), q, orders.SubmitOrderParams{
				OrderId: orderId,
				OwnerId: helpers.GetCurrentUserId(ctx),
			})
			if err != nil {
				return err
			}
		}

		txId := o.idGenerator()
		err = billing.RecordWalletTransaction(ctx, q, billing.RecordWalletTransactionParams{
			Id:               txId,
			Src:              &wallet.WalletID,
			Dest:             &systemWallet,
			Intent:           "order placed",
			IdempotencyToken: payload.IdempotencyToken,
			DoneBy:           helpers.GetCurrentUserId(ctx),
			TracingId:        tracingId,
			Currency:         payload.Currency,
			ExchangeRate:     exchangeRates[0],
			Value:            decimal.NewFromFloat32(payload.Quantity).Mul(payload.LimitPrice.Decimal),
		})
		if err != nil {
			return err
		}

		err = billing.UpdateWalletTransactionStatus(ctx, q, billing.UpdateTransactionStatusParams{
			TransactionId: txId,
			Status:        db.TransactionStatusCompleted,
		})
		return err
	})

	if err != nil {
		unprocessableErrors := []error{
			billing.ErrWalletNotFound,
			orders.ErrUnsupportedOrderType,
			orders.ErrOrderNotFound,
			orders.ErrInvalidOrderTransition,
		}
		passableErrors := []error{
			billing.ErrDuplicateIdempotencyKey,
			orders.ErrDuplicateOrder,
		}
		if slices.ContainsFunc(passableErrors, func(e error) bool {
			return errors.Is(err, e)
		}) {
			ctx.Status(http.StatusAccepted)
			return
		}

		if slices.ContainsFunc(unprocessableErrors, func(e error) bool { return errors.Is(err, e) }) {
			o.logger.Warn("could not process placing order", "err", err.Error())
			ctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		o.logger.Error("could not place order", "err", err.Error())
		helpers.InternalServerError(ctx)
		return
	}

	go o.matchingEngine.PlaceMatchOrder(matching.PlaceMatchOrderParams{
		OrderId:  orderId,
		Symbol:   payload.Symbol,
		Type:     db.OrderSide(payload.Side),
		Quantity: decimal.NewFromFloat32(payload.Quantity),
		Price:    payload.LimitPrice.Decimal,
	})
	ctx.Status(http.StatusAccepted)
	o.eb.Publish(EventKeyOrderCreatedV1, eventpayloads.OrderPlaced{
		OrderId:   orderId,
		Timestamp: time.Now(),
	})
}
func (o *Orders) handleFindOrders(ctx *gin.Context) {
	cursor := ctx.Query("cursor")
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "100"))
	if limit > 100 {
		limit = 100
	}

	orders, err := o.repo.GetOrders(ctx.Request.Context(), db.GetOrdersParams{
		ID:    cursor,
		Limit: int32(limit),
	})
	if err != nil {
		o.logger.Error("could not read orders", "err", err.Error())
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}

	var nextCursor string
	if len(orders) > 0 {
		nextCursor = orders[len(orders)-1].ID
	}
	ctx.JSON(http.StatusOK, gin.H{
		"data":       httppayloads.MapOrdersToDto(orders),
		"nextCursor": nextCursor,
	})
}

func (o *Orders) handleCancelOrder(c *gin.Context) {
	userId := helpers.GetCurrentUserId(c)
	orderId := c.Param("order")

	order, err := helpers.WithValueTransaction(c.Request.Context(), func(tx pgx.Tx) (db.Order, error) {
		repo := o.repo.WithTx(tx)
		return orders.CancelOrder(c.Request.Context(), repo, orders.CancelOrderParams{
			OwnerId: userId,
			OrderId: orderId,
		})
	})
	if err != nil {
		if errors.Is(err, orders.ErrOrderNotFound) {
			o.logger.Warn("order not found", "orderId", orderId, "uid", userId)
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		} else if errors.Is(err, orders.ErrInvalidOrderTransition) {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		o.logger.Error("could not cancel order", "err", err.Error(), "uid", userId, "orderId", orderId)
		helpers.InternalServerError(c)
		return
	}

	o.matchingEngine.CancelMatchOrder(orderId, order.Symbol, order.Side, order.LimitPrice.Decimal)
	c.Status(http.StatusAccepted)
}

func (c *Orders) MountV1(ctx context.Context, r *gin.RouterGroup) {
	jwtMiddleware := helpers.ProvideJWTMiddleware()
	kycMiddleware := helpers.ProvideKYCMiddleware()

	router := r.Group("/orders")
	public := router.Group("")
	public.GET("", c.handleFindOrders)

	secured := router.Group("", jwtMiddleware())
	secured.POST("", kycMiddleware(contract.KYCPending), c.handlePlaceOrder)

	mutations := secured.Group("/:order")
	mutations.DELETE("", c.handleCancelOrder)
	c.subscribeToEventsV1(ctx)
	go c.bootstrapMatchingEngine(ctx)
	go c.applyPendingFills(ctx)
}

func (c *Orders) applyPendingFills(ctx context.Context) {
	batches := 0
	added := 0
	filled := make([]string, 0)
	err := helpers.WithTransactionRepo(ctx, func(q *db.Queries) error {
		const batchSize = 1000
		cursor := "00000000000000000000000000"
		for {
			batch, err := q.LookupPendingExecutions(ctx, db.LookupPendingExecutionsParams{ID: cursor, Limit: batchSize})
			if err != nil {
				return fmt.Errorf("could not fetch pending executions: %w", err)
			}

			if len(batch) == 0 {
				break
			}

			cursor = batch[len(batch)-1]
			for _, exec := range batch {
				err := orders.ApplyExecutionFill(ctx, q, exec)
				if err != nil {
					return err
				}

				filled = append(filled, exec)
				added++
			}
			batches++
		}

		c.logger.Info("executions filled", "filled", added, "batches", batches, "batchSize", batchSize)
		return nil
	})

	if err != nil {
		c.logger.Error("filling pending executions failed", "err", err.Error())
	} else {
		c.logger.Info("filled pending executions", "count", added*batches)
	}

	for _, id := range filled {
		helpers.PublishEvent(ctx, matching.EventKeyExecutionFilled, matching.ExecutionFilledEventArgs{
			ExecutionId: id,
		})
	}
}
func (c *Orders) handleOnMatchFoundEvent(ctx context.Context, ch <-chan contract.BusEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				continue
			}
			event, ok := ev.Data.(matching.OrderMatchedEventArgs)
			if !ok {
				c.logger.Warn("invalid event data", "event", matching.EventKeyOrderMatched)
				continue
			}

			c.logger.Info("order match found, creating execution", "buy", event.Bid, "sell", event.Ask)
			execId, err := helpers.WithValueTransactionRepo(ctx, func(q *db.Queries) (string, error) {
				var execId = c.idGenerator()
				bid, err := q.FindOrderById(ctx, event.Bid)
				if err != nil {
					return "", err
				}

				ask, err := q.FindOrderById(ctx, event.Ask)
				if err != nil {
					return "", err
				}

				rates, err := helpers.GetDefaultExchangeRateDecimalFor(ctx, bid.Currency, ask.Currency)
				if err != nil {
					return "", err
				}
				tracingId := c.idGenerator()
				_, err = orders.CreateExecution(ctx, q, orders.CreateExecutionParams{
					Id:              execId,
					Bid:             event.Bid,
					Ask:             event.Ask,
					User:            helpers.GetSystemUserId(),
					TracingId:       tracingId,
					MatchedQuantity: event.MatchQuantity,
					BidExchangeRate: rates[0],
					AskExchangeRate: rates[1],
				})

				err = orders.ApplyExecutionFill(ctx, q, execId)
				if err != nil {
					return "", err
				}

				return execId, nil
			})
			if err != nil {
				if errors.Is(err, orders.ErrDuplicateExecutionFound) {
					c.logger.Warn("duplicate order execution found", "sell", event.Ask, "buy", event.Bid, "qty", event.MatchQuantity)
					return
				}
				c.logger.Error("could not create execution", "err", err.Error(), "sell", event.Ask, "buy", event.Bid, "qty", event.MatchQuantity)
				return
			}

			c.eb.Publish(matching.EventKeyExecutionFilled, matching.ExecutionFilledEventArgs{
				ExecutionId: execId,
			})
		}
	}
}

func (c *Orders) subscribeToEventsV1(ctx context.Context) {
	matchFound := c.eb.Subscribe(ctx, matching.EventKeyOrderMatched)
	go c.handleOnMatchFoundEvent(ctx, matchFound)
}

func NewOrdersController(l *slog.Logger, q *db.Queries, re *orders.RiskEngine, p *pgxpool.Pool, idg contract.IdFactory, e *matching.Engine, eb contract.EventBus) *Orders {
	return &Orders{
		repo:           q,
		logger:         l.With("controller", "orders"),
		riskEngine:     re,
		pool:           p,
		idGenerator:    idg,
		matchingEngine: e,
		eb:             eb,
	}
}
