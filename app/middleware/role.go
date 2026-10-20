package middleware

import (
	"net/http"
	"slices"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/gin-gonic/gin"
)

type RoleMiddleware func(roles ...contract.Role) gin.HandlerFunc

// RequireRole creates a middleware that requires one or more roles.
func RequireRole(a contract.RoleAuthorizer) RoleMiddleware {
	return func(roles ...contract.Role) gin.HandlerFunc {
		return func(c *gin.Context) {
			uid := c.GetString("uid")
			if uid == "" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
				return
			}

			if v, ok := c.Get("claims"); ok {
				if claims, ok := v.(contract.BearerClaims); ok {
					for _, role := range roles {
						if slices.Contains(claims.Roles, string(role)) {
							c.Next()
							return
						}
					}
				}
			}

			if a != nil && a.HasRole(c.Request.Context(), uid, roles) {
				c.Next()
				return
			}

			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		}
	}
}
