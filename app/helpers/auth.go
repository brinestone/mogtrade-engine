package helpers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/web/middleware"
	"github.com/gin-gonic/gin"
	"go-slim.dev/ioc"
)

func GetSystemUserId() string {
	return os.Getenv("SYSTEM_USER_ID")
}

func GetCurrentUserIdOrDefault(c *gin.Context, def string) string {
	uid := c.GetString("uid")
	if uid == "" {
		return def
	}
	return uid
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
func ProvideJWTMiddleware() middleware.JWTMiddleware {
	ptr, _ := ioc.NamedGet[middleware.JWTMiddleware](context.TODO(), "middleware.jwt")
	return *ptr
}

func GetCookieDomain() string {
	host := os.Getenv("HOST")
	return strings.ReplaceAll(host, fmt.Sprintf(":%s", os.Getenv("PORT")), "")
}
