package api

import (
	"net/http"

	"github.com/brinestone/mogtrade/web/controller"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go-slim.dev/ioc"
)

func MountApiV1(r *gin.RouterGroup) error {
	router := r.Group("/v1")

	orders, err := controller.NewOrdersController()
	if err != nil {
		return err
	}

	orders.MountV1(router)
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
	return nil
	// if err := ioc.Singleton(func(repo *db.Queries) *controller.Orders {
	// 	orders := controller.Orders{}
	// 	return &orders
	// }); err != nil {
	// 	return err
	// }
	// return nil
}
