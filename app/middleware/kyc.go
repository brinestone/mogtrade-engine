package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/gin-gonic/gin"
)

type KYCMiddleware func(statuses ...contract.KYCStatus) gin.HandlerFunc

func RequireKycVerified(logger *slog.Logger) KYCMiddleware {
	l := logger.With("middleware", "kyc-require-verified")
	return func(statuses ...contract.KYCStatus) gin.HandlerFunc {
		return func(c *gin.Context) {
			v, ok := c.Get("claims")
			if ok {
				claims, _ := v.(contract.BearerClaims)
				if ok && slices.Contains(statuses, claims.KYCStatus) {
					c.Next()
					return
				}
			}
			l.Warn("kyc verification failed, kyc_status claim absent or unverified")
			var s []string
			for _, ss := range statuses {
				s = append(s, string(ss))
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("KYC Status must be one of: %s", strings.Join(s, ",")),
			})
		}
	}
}
