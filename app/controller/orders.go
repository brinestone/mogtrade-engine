package controller

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/infra/events"
	"github.com/brinestone/mogtrade/services/billing"
	"github.com/brinestone/mogtrade/services/matching"
	"github.com/brinestone/mogtrade/services/orders"
	"github.com/brinestone/mogtrade/web/helpers"
	eventpayloads "github.com/brinestone/mogtrade/web/payloads/events"
	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go-slim.dev/ioc"
)

const (
	EventKeyOrderCreatedV1 string = "order.created.v1"
)

type Orders struct {
	repo           *db.Queries
	logger         *slog.Logger
	riskEngine     *orders.RiskEngine
	pool           *pgxpool.Pool
	matchingEngine *matching.MatchingEngine
	idGenerator    contract.IdGeneratorFunc
	eb             events.EventBus
}

func (o *Orders) handlePlaceOrder(ctx *gin.Context) {
	var payload httppayloads.PlaceOrderPayload

	o.logger.Info("handling request to place orders, validating request")
	if err := ctx.ShouldBind(&payload); err != nil {
		o.logger.Warn("form binding validation failed, aborting")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctx.ShouldBindHeader(&payload); err != nil {
		o.logger.Warn("header binding validation failed, aborting")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	errMsgs := payload.Validate()
	if len(errMsgs) > 0 {
		o.logger.Warn("order request failed validation. Aborting request")
		ctx.JSON(http.StatusBadRequest, gin.H{"errors": errMsgs})
		return
	}
	o.logger.Info("request validated successfully, opening database transaction")

	tx, err := o.pool.Begin(ctx.Request.Context())
	if err != nil {
		o.logger.Error("could not begin database transaction", "err", err.Error())
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}
	defer tx.Rollback(ctx.Request.Context())

	o.logger.Info("transaction opened, getting user wallet info", "wallet-type", payload.WalletType)
	q := o.repo.WithTx(tx)
	wallet, err := billing.GetCurrentWalletSnapshot(ctx.Request.Context(), q, billing.GetWalletSnapshotParams{
		Type:    payload.WalletType,
		OwnerId: helpers.GetCurrentUserId(ctx),
	})
	if err != nil {
		if errors.Is(err, billing.ErrWalletNotFound) {
			o.logger.Warn("wallet not found", "uid", helpers.GetCurrentUserId(ctx), "wallet-type", payload.WalletType)
			ctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		o.logger.Error("could not retrieve wallet info", "wallet-type", payload.WalletType, "uid", helpers.GetCurrentUserId(ctx), "err", err.Error())
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}
	o.logger.Info("wallet info retrieval successful, performing risk checks")
	result := o.riskEngine.ValidateOrder(ctx.Request.Context(), orders.OrderContext{
		Symbol:         payload.Symbol,
		Side:           payload.Side,
		OrderType:      payload.Type,
		Quantity:       decimal.NewFromFloat32(payload.Quantity),
		LimitPrice:     payload.LimitPrice,
		StopPrice:      payload.StopPrice,
		AccountBalance: wallet.CurrentBalance,
	})
	if !result.Approved {
		o.logger.Warn("risk check failed, aborting", "reason", result.Reason, "wallet-id", wallet.WalletID, "wallet-type", payload.WalletType, "uid", helpers.GetCurrentUserId(ctx))
		ctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": result.Reason})
		return
	}

	o.logger.Info("risk checks passed, creating order")
	orderId := o.idGenerator()
	tracingId := o.idGenerator()
	systemWalletId := helpers.GetSystemWalletByType(wallet.WalletType)
	_, err = ioc.Invoke(ctx.Request.Context(), func(cc contract.CurrencyConverter, e contract.TickerInfoProvider) error {
		rate, err := cc.GetDefaultExchangeRates(ctx, payload.Currency)
		if err != nil {
			return err
		}
		err = orders.PlaceOrder(ctx.Request.Context(), q, orders.PlaceOrderParams{
			TracingId:            tracingId,
			OrderId:              orderId,
			WalletId:             wallet.WalletID,
			WalletTransactionId:  o.idGenerator(),
			SystemWalletId:       systemWalletId,
			IdempotencyToken:     payload.IdempotencyToken,
			PlacedBy:             helpers.GetCurrentUserId(ctx),
			Currency:             payload.Currency,
			ExchangeRateSnapshot: decimal.NewFromFloat32(1 / rate[0]),
			Symbol:               payload.Symbol,
			Side:                 db.OrderSide(payload.Side),
			Type:                 db.OrderType(payload.Type),
			Quantity:             decimal.NewFromFloat32(payload.Quantity),
			LimitPrice:           payload.LimitPrice,
			StopPrice:            payload.StopPrice,
		})
		return err
	})

	if err != nil {
		if errors.Is(err, orders.ErrDuplicateOrder) {
			ctx.AbortWithStatus(http.StatusConflict)
			return
		}
		o.logger.Error("could not create order", "err", err.Error())
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}
	tx.Commit(ctx.Request.Context())

	ctx.Status(http.StatusCreated)
	o.eb.Publish(EventKeyOrderCreatedV1, eventpayloads.OrderCreated{
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
func (c *Orders) MountV1(ctx context.Context, r *gin.RouterGroup) {
	authMiddleware := helpers.ProvideAuthMiddleware()
	router := r.Group("/orders")
	router.GET("", c.handleFindOrders)

	secured := router.Group("", authMiddleware)
	secured.POST("", c.handlePlaceOrder)
	c.subscribeToEventsV1(ctx)
}

func (c *Orders) handleOnMatchFoundEvent(ctx context.Context, ch events.DataChannel) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				continue
			}
			_, ok = ev.Data.(matching.OrderMatched)
			if !ok {
				continue
			}
			// TODO: use the event to create executions. Maybe enrich the event args with more information.
		}
	}
}

func (c *Orders) subscribeToEventsV1(ctx context.Context) {
	matchFound := c.eb.Subscribe(matching.EventKeyOrderMatched)
	go c.handleOnMatchFoundEvent(ctx, matchFound)
}

func NewOrdersController(l *slog.Logger, q *db.Queries, re *orders.RiskEngine, p *pgxpool.Pool, idg contract.IdGeneratorFunc, e *matching.MatchingEngine, eb events.EventBus) *Orders {
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
