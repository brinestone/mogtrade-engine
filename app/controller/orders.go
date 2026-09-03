package controller

import (
	"log/slog"
	"net/http"

	"github.com/brinestone/mogtrade/infra"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/services/orders"
	"github.com/brinestone/mogtrade/web/payloads"
	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
	"github.com/shopspring/decimal"
)

type Orders struct {
	repo       *db.Queries
	logger     *slog.Logger
	riskEngine *orders.RiskEngine
	connGetter infra.ConnProviderFunc
}

func (o *Orders) HandlePlaceOrder(ctx *gin.Context) {
	var payload payloads.PlaceOrderPayload

	o.logger.Info("handling request to place orders, validating request")
	if err := ctx.ShouldBind(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctx.ShouldBindHeader(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	errors := payload.Validate()
	if len(errors) > 0 {
		o.logger.Warn("order request failed validation. Aborting request")
		ctx.JSON(http.StatusBadRequest, gin.H{"errors": errors})
		return
	}
	o.logger.Info("request validated successfully, creating order")
	id := ulid.Make()
	o.repo.PlaceOrder(ctx.Request.Context(), db.PlaceOrderParams{
		ID:               id.String(),
		Symbol:           payload.Symbol,
		UserID:           nil, // TODO: Use User from session store or jwt
		Side:             db.OrderSide(payload.Side),
		OrderType:        db.OrderType(payload.Type),
		Quantity:         decimal.NewFromFloat32(payload.Quantity),
		LimitPrice:       payload.LimitPrice,
		StopPrice:        payload.StopPrice,
		ClientOrderID:    &payload.IdempotencyToken,
		AverageFillPrice: decimal.NewNullDecimal(decimal.NewFromInt(50)),
	})

	o.riskEngine.ValidateOrder(ctx.Request.Context(), orders.OrderContext{
		Symbol:     payload.Symbol,
		Side:       payload.Side,
		OrderType:  payload.Type,
		Quantity:   decimal.NewFromFloat32(payload.Quantity),
		LimitPrice: payload.LimitPrice,
		StopPrice:  payload.StopPrice,
		// AccountBalance: ,
	}, orders.WithMarginChecking())
}

func (c *Orders) MountV1(r *gin.RouterGroup) {
	router := r.Group("/orders")
	router.POST("", c.HandlePlaceOrder)
}

func NewOrdersController(l *slog.Logger, q *db.Queries, re *orders.RiskEngine, cg infra.ConnProviderFunc) *Orders {
	return &Orders{
		repo:       q,
		logger:     l,
		riskEngine: re,
		connGetter: cg,
	}
}
