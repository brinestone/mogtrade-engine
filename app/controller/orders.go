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
		rate, err := o.cc.GetDefaultExchangeRates(ctx.Request.Context(), payload.Currency)
		if err != nil {
			return err
		}
		exchangerate := decimal.NewFromFloat32(1 / rate[0])
		logger.Debug("creating order record")
		err = orders.PlaceOrder(ctx.Request.Context(), q, orders.PlaceOrderParams{
			TracingId:            tracingId,
			OrderId:              orderId,
			WalletId:             wallet.WalletID,
			WalletTransactionId:  o.idGenerator(),
			SystemWalletId:       systemWallet,
			PlacedBy:             helpers.GetCurrentUserId(ctx),
			Currency:             payload.Currency,
			ExchangeRateSnapshot: exchangerate,
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

		err = billing.RecordWalletTransaction(ctx, q, billing.RecordWalletTransactionParams{
			Id:               o.idGenerator(),
			Src:              &wallet.WalletID,
			Dest:             &systemWallet,
			Intent:           "order placed",
			IdempotencyToken: payload.IdempotencyToken,
			DoneBy:           helpers.GetCurrentUserId(ctx),
			TracingId:        tracingId,
			Currency:         payload.Currency,
			ExchangeRate:     exchangerate,
			Value:            decimal.NewFromFloat32(payload.Quantity).Mul(payload.LimitPrice.Decimal),
		})
		if err != nil {
			return err
		}
		return nil
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

func (c *Orders) MountV1(r *gin.RouterGroup) {
	jwtMiddleware := helpers.ProvideJWTMiddleware()
	kycMiddleware := helpers.ProvideKYCMiddleware()

	router := r.Group("/orders")
	public := router.Group("")
	public.GET("", c.handleFindOrders)

	secured := router.Group("", jwtMiddleware())
	secured.POST("", kycMiddleware(contract.KYCPending), c.handlePlaceOrder)

	mutations := secured.Group("/:order")
	mutations.DELETE("", c.handleCancelOrder)
	c.subscribeToEventsV1()
	go c.bootstrapMatchingEngine(context.TODO())
}

func (c *Orders) handleOnMatchFoundEvent(ctx context.Context, ch <-chan contract.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				continue
			}
			event, ok := ev.Data.(matching.OrderMatched)
			if !ok {
				c.logger.Warn("invalid event data", "event", matching.EventKeyOrderMatched)
				continue
			}

			c.logger.Info("order match found, creating execution", "buy", event.BuyOrder, "sell", event.SellOrder)
			tx, err := c.pool.Begin(ctx)
			if err != nil {
				c.logger.Error("could not open database transaction", "err", err.Error())
				return
			}

			buyOrder, err := c.repo.FindOrderById(ctx, event.BuyOrder)
			if err != nil {
				c.logger.Error("could not find buy order", "err", err.Error(), "buy-order", event.BuyOrder, "sell-order", event.SellOrder)
				tx.Rollback(ctx)
				continue
			}
			sellOrder, err := c.repo.FindOrderById(ctx, event.SellOrder)
			if err != nil {
				c.logger.Error("could not find sell order", "err", err.Error(), "sell-order", event.SellOrder, "buy-order", event.BuyOrder)
				tx.Rollback(ctx)
				continue
			}

			tracingId := c.idGenerator()
			err = orders.CreateExecution(ctx, c.repo.WithTx(tx), orders.CreateExecutionParams{
				OrderId:     event.BuyOrder,
				Id:          c.idGenerator(),
				TracingId:   tracingId,
				Quantity:    buyOrder.Quantity,
				FeeCurrency: buyOrder.Currency,
				FeeRate:     buyOrder.ExchangeRate,
				FeeAmount:   buyOrder.Fee,
				Price:       buyOrder.LimitPrice.Decimal,
			})
			if err != nil {
				c.logger.Error("could not create buy execution", "err", err.Error())
				tx.Rollback(ctx)
				continue
			}

			err = orders.CreateExecution(ctx, c.repo.WithTx(tx), orders.CreateExecutionParams{
				OrderId:     event.SellOrder,
				Id:          c.idGenerator(),
				FeeCurrency: sellOrder.Currency,
				TracingId:   tracingId,
				FeeRate:     sellOrder.ExchangeRate,
				FeeAmount:   sellOrder.Fee,
				Price:       sellOrder.LimitPrice.Decimal,
				Quantity:    sellOrder.Quantity,
			})
			if err != nil {
				c.logger.Error("could not create sell execution", "order", sellOrder.ID, "err", err.Error())
				tx.Rollback(ctx)
				continue
			}
			tx.Commit(ctx)
		}
	}
}

func (c *Orders) subscribeToEventsV1() {
	ctx := context.TODO()
	matchFound := c.eb.Subscribe(ctx, matching.EventKeyOrderMatched)
	go c.handleOnMatchFoundEvent(ctx, matchFound)
}

func NewOrdersController(l *slog.Logger, q *db.Queries, re *orders.RiskEngine, p *pgxpool.Pool, idg contract.IdFactory, e *matching.Engine, eb contract.EventBus, cc contract.CurrencyConverter) *Orders {
	return &Orders{
		repo:           q,
		logger:         l.With("controller", "orders"),
		riskEngine:     re,
		pool:           p,
		idGenerator:    idg,
		matchingEngine: e,
		eb:             eb,
		cc:             cc,
	}
}
