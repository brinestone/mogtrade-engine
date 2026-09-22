package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/infra/events"
	"github.com/brinestone/mogtrade/services/billing"
	"github.com/brinestone/mogtrade/services/orders"
	"github.com/brinestone/mogtrade/web/helpers"
	eventpayloads "github.com/brinestone/mogtrade/web/payloads/events"
	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

const (
	EventKeyOrderCreatedV1 string = "order.created.v1"
)

type Orders struct {
	repo        *db.Queries
	logger      *slog.Logger
	riskEngine  *orders.RiskEngine
	pool        *pgxpool.Pool
	idGenerator contract.IdGeneratorFunc
	eb          events.EventBus
}

func (o *Orders) handlePlaceOrder(ctx *gin.Context) {
	var payload httppayloads.PlaceOrderPayload

	o.logger.Info("handling request to place orders, validating request")
	if err := ctx.ShouldBind(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctx.ShouldBindHeader(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	errMsgs := payload.Validate()
	if len(errMsgs) > 0 {
		o.logger.Warn("order request failed validation. Aborting request")
		ctx.JSON(http.StatusBadRequest, gin.H{"errors": errMsgs})
		return
	}
	o.logger.Info("request validated successfully, creating order")

	tx, err := o.pool.Begin(ctx.Request.Context())
	if err != nil {
		o.logger.Error("could not begin database transaction", "err", err.Error())
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}
	defer tx.Rollback(ctx.Request.Context())

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
		o.logger.Error("could not retrieve wallet snapshot", "wallet-type", payload.WalletType, "uid", helpers.GetCurrentUserId(ctx), "err", err.Error())
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}
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
		ctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": result.Reason})
		return
	}

	orderId := o.idGenerator()
	tracingId := o.idGenerator()
	err = orders.CreateOrder(ctx.Request.Context(), q, orders.PlaceOrderParams{
		TracingId:        tracingId,
		OrderId:          orderId,
		IdempotencyToken: payload.IdempotencyToken,
		PlacedBy:         helpers.GetCurrentUserId(ctx),
		Symbol:           payload.Symbol,
		Side:             db.OrderSide(payload.Side),
		Type:             db.OrderType(payload.Type),
		Quantity:         decimal.NewFromFloat32(payload.Quantity),
		LimitPrice:       payload.LimitPrice,
		StopPrice:        payload.StopPrice,
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
func (c *Orders) MountV1(r *gin.RouterGroup) {
	authMiddleware := helpers.ProvideAuthMiddleware()
	router := r.Group("/orders")
	router.GET("", c.handleFindOrders)

	secured := router.Group("", authMiddleware)
	secured.POST("", c.handlePlaceOrder)
}

func NewOrdersController(l *slog.Logger, q *db.Queries, re *orders.RiskEngine, p *pgxpool.Pool, idg contract.IdGeneratorFunc) *Orders {
	return &Orders{
		repo:        q,
		logger:      l.With("controller", "orders"),
		riskEngine:  re,
		pool:        p,
		idGenerator: idg,
	}
}
