package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/brinestone/mogtrade/web/api"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go-slim.dev/ioc"
)

var (
	port int
)

func main() {
	godotenv.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()
	bootstrapApplication(ctx)
	startAsyncTasks(ctx)
	go func() {
		<-ctx.Done()
		ioc.Invoke(context.TODO(), func(l *slog.Logger) {
			l.Info("shutting down...")
		})
	}()
	engine := gin.Default()
	api.MountGlobalMiddlewares(engine, strings.Split(os.Getenv("ALLOWED_ORIGINS"), ";"))
	baseRouter := engine.Group("/api")
	if err := api.MountApiV1(baseRouter); err != nil {
		panic(err)
	}
	startAsyncTasks(ctx)
	engine.Run(fmt.Sprintf(":%d", port))
}
