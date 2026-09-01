package controller

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/brinestone/mogtrade/core/payloads"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/gin-gonic/gin"
	"go-slim.dev/ioc"
)

type Orders struct {
	repo   *db.Queries
	logger *slog.Logger
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
	o.logger.Info("request validated successfully, evaluating risks")
}

func (c *Orders) MountV1(r *gin.RouterGroup) {
	router := r.Group("/orders")
	router.POST("", c.HandlePlaceOrder)
}

func NewOrdersController() (*Orders, error) {
	o := Orders{}
	logger, err := ioc.Get[*slog.Logger](context.TODO())
	if err != nil {
		return nil, err
	}

	o.logger = (*logger).With("controller", "orders")

	repo, err := ioc.Get[*db.Queries](context.TODO())
	if err != nil {
		return nil, err
	}
	o.repo = (*repo)
	return &o, nil
}
