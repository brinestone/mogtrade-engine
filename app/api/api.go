package api

import (
	"context"
	"net/http"
	"time"

	"github.com/brinestone/mogtrade/web/controller"
	"github.com/brinestone/mogtrade/web/middleware"
	"github.com/danielkov/gin-helmet/ginhelmet"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-slim.dev/ioc"
)

type ApiConfig struct {
	Host           string
	AllowedOrigins []string
	SessionStore   sessions.Store
}

func MountApiV1(r *gin.RouterGroup, cfg ApiConfig) error {
	ctx := context.TODO()

	router := r.Group("/v1")
	router.Use(middleware.RateLimiter())
	router.Use(ginhelmet.Default())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Device-Id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	ioc.Invoke(ctx, func(c *controller.Auth) {
		c.MountV1(router)
	})
	ioc.Invoke(ctx, func(c *controller.Orders) {
		c.MountV1(router)
	})
	ioc.Invoke(ctx, func(c *controller.Wallets) {
		c.MountV1(router)
	})
	ioc.Invoke(ctx, func(c *controller.User) {
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
	ioc.Factory(controller.NewUserController, true)
	ioc.NamedFactory("middleware.auth", middleware.RequireAuth)
	return nil
}
