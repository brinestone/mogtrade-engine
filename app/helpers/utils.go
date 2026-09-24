package helpers

import (
	"net/http"

	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
)

func Ptr[T any](v T) *T {
	return &v
}

func InternalServerError(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
}
