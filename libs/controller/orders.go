package controller

import (
	"net/http"

	"github.com/brinestone/mogtrade/libs/contract"
	"github.com/brinestone/mogtrade/libs/payloads"
	"github.com/gin-gonic/gin"
)

type Orders struct {
	contract.UsesRepository
}

func (o *Orders) HandlePlaceOrder(ctx *gin.Context) {
	var payload payloads.PlaceOrderPayload

	if err := ctx.ShouldBind(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctx.ShouldBindHeader(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

}

func (c *Orders) MountV1(r *gin.RouterGroup) {
	router := r.Group("/orders")
	router.POST("", c.HandlePlaceOrder)
}
