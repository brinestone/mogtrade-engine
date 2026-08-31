package api

import (
	db "github.com/brinestone/mogtrade/internal/models"
	"github.com/brinestone/mogtrade/libs/controller"
	"github.com/gin-gonic/gin"
	"github.com/golobby/container/v3"
)

func MountApiV1(r *gin.RouterGroup, ioc *container.Container) {
	router := r.Group("/v1")

	var orders *controller.Orders
	ioc.Resolve(orders)

	orders.MountV1(router)
}

func SetupControllersDi(c ApiConfig) error {
	if err := c.Ioc.SingletonLazy(func(repo *db.Queries) *controller.Orders {
		orders := controller.Orders{
			UsesRepository: c,
		}
		return &orders
	}); err != nil {
		return err
	}
	return nil
}
