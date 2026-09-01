package controller

import (
	"log/slog"
	"net/http"

	db "github.com/brinestone/mogtrade/internal/models"
	"github.com/brinestone/mogtrade/libs/payloads"
	"github.com/gin-gonic/gin"
	"github.com/golobby/container/v3"
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

func NewOrdersController(ioc *container.Container) (*Orders, error) {
	o := Orders{}
	logger := slog.Logger{}
	if err := ioc.Resolve(&logger); err != nil {
		return nil, err
	}

	o.logger = logger.With("controller", "orders")

	if err := ioc.Resolve(&o.repo); err != nil {
		return nil, err
	}
	return &o, nil
}
