package helpers

import (
	"context"

	"github.com/brinestone/mogtrade/web/middleware"
	"go-slim.dev/ioc"
)

func ProvideKYCMiddleware() middleware.KYCMiddleware {
	ptr, _ := ioc.NamedGet[middleware.KYCMiddleware](context.TODO(), "middleware.kyc")
	return *ptr
}
