package middleware

import (
	"net/http"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/gin-gonic/gin"
)

type RoleMiddleware func(roles ...contract.Role) gin.HandlerFunc

// RoleMiddleware creates a middleware that requires one or more roles.
func RequireRole(a contract.RoleAuthorizer) RoleMiddleware {
	return func(roles ...contract.Role) gin.HandlerFunc {
		return func(c *gin.Context) {
			uid := c.GetString("uid")
			if uid == "" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
				return
			}

			if a.HasRole(c.Request.Context(), uid, roles) {
				c.Next()
			} else {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			}
		}
	}
}
