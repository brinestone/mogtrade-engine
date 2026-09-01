package api

import (
	"net/http"

	"github.com/brinestone/mogtrade/controller"
	db "github.com/brinestone/mogtrade/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golobby/container/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func MountApiV1(r *gin.RouterGroup, ioc *container.Container) error {
	router := r.Group("/v1")

	orders, err := controller.NewOrdersController(ioc)
	if err != nil {
		return err
	}

	orders.MountV1(router)
	mountHealth(router, ioc)
	return nil
}

func mountHealth(r *gin.RouterGroup, ioc *container.Container) {
	r.GET("/health", func(c *gin.Context) {
		var conn *pgxpool.Conn
		if err := ioc.Resolve(conn); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "not available"})
			return
		}
		if err := conn.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "not available"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

func SetupControllers(ioc *container.Container) error {
	if err := ioc.Singleton(func(repo *db.Queries) *controller.Orders {
		orders := controller.Orders{}
		return &orders
	}); err != nil {
		return err
	}
	return nil
}
