package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/brinestone/mogtrade/core/contract"
	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type JWTMiddleware func(optional ...bool) gin.HandlerFunc

func RequireAuth(logger *slog.Logger, v contract.TokenVerifier) JWTMiddleware {
	l := logger.With("midddleware", "auth-jwt")
	return func(optional ...bool) gin.HandlerFunc {
		shouldAbort := !(len(optional) > 0 && optional[0])
		return func(c *gin.Context) {
			authHeaderValue, found := c.Request.Header["Authorization"]
			if !found {
				if shouldAbort {
					c.AbortWithStatus(http.StatusUnauthorized)
				} else {
					c.Next()
				}
				return
			}
			parts := strings.Split(authHeaderValue[0], " ")
			scheme, token := parts[0], parts[1]
			if scheme != "Bearer" {
				l.Warn("invalid authentication scheme provided", "have", scheme, "want", "Bearer")
				if shouldAbort {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": http.ErrSchemeMismatch.Error()})
				} else {
					c.Next()
				}
				return
			}

			valid, err := v.VerifyToken(token, func(id string) {
				c.Set("uid", id)
			})
			v.VerifyWithClaims(token, func(claims contract.BearerClaims) {
				c.Set("claims", claims)
			})
			if err != nil {
				if errors.Is(err, jwt.ErrTokenMalformed) || errors.Is(err, jwt.ErrTokenSignatureInvalid) || errors.Is(err, jwt.ErrTokenInvalidClaims) {
					l.Warn("invalid bearer token", "scheme", scheme, "err", err.Error())
					if shouldAbort {
						c.AbortWithStatus(http.StatusUnauthorized)
					} else {
						c.Next()
					}
					return
				}
				l.Error("error while verifying token", "err", err.Error())
				if shouldAbort {
					c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
				} else {
					c.Next()
				}
				return
			}
			if !valid {
				l.Warn("invalid bearer token", "scheme", scheme)
				if shouldAbort {
					c.AbortWithStatus(http.StatusUnauthorized)
				} else {
					c.Next()
				}
				return
			}
			c.Next()
		}
	}
}
