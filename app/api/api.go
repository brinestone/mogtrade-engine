package api

import (
	"context"
	"net/http"

	"github.com/brinestone/mogtrade/web/controller"
	"github.com/gin-gonic/gin"
	"go-slim.dev/ioc"
)

func MountApiV1(r *gin.RouterGroup) error {
	ctx := context.TODO()
	router := r.Group("/v1")

	ioc.Invoke(ctx, func(c *controller.Auth) {
		c.MountV1(router)
	})
	ioc.Invoke(ctx, func(c *controller.Orders) {
		c.MountV1(router)
	})
	ioc.Invoke(ctx, func(c *controller.Wallets) {
		c.MountV1(router)
	})

	mountHealth(router)
	return nil
}

func mountHealth(r *gin.RouterGroup) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

func SetupControllers() error {
	ioc.Factory(controller.NewOrdersController, true)
	ioc.Factory(controller.NewAuthController, true)
	ioc.Factory(controller.NewWalletsController, true)
	return nil
}
