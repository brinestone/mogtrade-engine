package middleware

import (
	"log/slog"
	"net/http"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/gin-gonic/gin"
)

func RequireKycVerified(logger *slog.Logger) gin.HandlerFunc {
	l := logger.With("middleware", "kyc-require-verified")
	return func(c *gin.Context) {
		v, ok := c.Get("claims")
		if ok {
			claims, _ := v.(map[string]any)
			status, ok := claims["kyc_status"].(contract.KYCStatus)
			if ok && status == contract.KYCVerified {
				c.Next()
				return
			}
		}
		if !ok {
			l.Warn("kyc verification failed, kyc_status claim absent or unverified")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "kyc status must be verified",
			})
			return
		}
	}
}
