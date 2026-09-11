package helpers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/gin-gonic/gin"
	"go-slim.dev/ioc"
)

func GetSystemUserId() string {
	return os.Getenv("SYSTEM_USER_ID")
}

func GetCurrentUserId(c *gin.Context) string {
	return c.GetString("uid")
}

func GetCurrentUser(c *gin.Context) (db.User, error) {
	uid := c.GetString("uid")
	return ioc.Call2[db.User](c.Request.Context(), func(q *db.Queries) (db.User, error) {
		return q.FindUserById(c.Request.Context(), uid)
	})
}
func ProvideAuthMiddleware() gin.HandlerFunc {
	ptr, _ := ioc.NamedGet[gin.HandlerFunc](context.TODO(), "middleware.auth")
	return *ptr
}

func GetCookieDomain() string {
	host := os.Getenv("HOST")
	return strings.ReplaceAll(host, fmt.Sprintf(":%s", os.Getenv("PORT")), "")
}
