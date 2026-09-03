package api

import (
	"context"
	"net/http"

	"github.com/brinestone/mogtrade/web/controller"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go-slim.dev/ioc"
)

func MountApiV1(r *gin.RouterGroup) error {
	router := r.Group("/v1")

	oc, err := ioc.Get[*controller.Orders](context.TODO())
	if err != nil {
		return err
	}

	(*oc).MountV1(router)
	mountHealth(router)
	return nil
}

func mountHealth(r *gin.RouterGroup) {
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

func SetupControllers() error {
	ioc.Factory(controller.NewOrdersController, true)
	return nil
}
