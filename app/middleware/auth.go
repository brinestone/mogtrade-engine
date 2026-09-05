package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/brinestone/mogtrade/services/auth"
	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func RequireAuth(l *slog.Logger, v auth.TokenVerifier) gin.HandlerFunc {
	l2 := l.With("midddleware", "auth-jwt")
	return func(c *gin.Context) {
		authHeaderValue, found := c.Request.Header["Authorization"]
		if !found {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		parts := strings.Split(authHeaderValue[0], " ")
		scheme, token := parts[0], parts[1]
		if scheme != "Bearer" {
			l2.Warn("invalid authentication scheme provided", "have", scheme, "want", "Bearer")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": http.ErrSchemeMismatch.Error()})
			return
		}

		valid, err := v.VerifyToken(token, func(id string) {
			c.Set("uid", id)
		})
		if err != nil {
			if errors.Is(err, jwt.ErrTokenSignatureInvalid) || errors.Is(err, jwt.ErrTokenInvalidClaims) {
				l2.Warn("invalid bearer token", "scheme", scheme, "err", err.Error())
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			l2.Error("error while verifying token", "err", err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
			return
		}
		if !valid {
			l2.Warn("invalid bearer token", "scheme", scheme)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}
