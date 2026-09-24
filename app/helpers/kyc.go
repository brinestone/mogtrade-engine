package helpers

import (
	"context"

	"github.com/gin-gonic/gin"
	"go-slim.dev/ioc"
)

func ProvideKYCMiddleware() gin.HandlerFunc {
	ptr, _ := ioc.NamedGet[gin.HandlerFunc](context.TODO(), "middleware.kyc")
	return *ptr
}
