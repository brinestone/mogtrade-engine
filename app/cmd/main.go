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

	// 1. Set up logging first and let it inject dependencies
	if err := setupLogging(ctx); err != nil {
		panic(err)
	}

	if err := parseVars(); err != nil {
		panic(err)
	}
	if err := setupDbConnection(ctx); err != nil {
		panic(err)
	}
	if err := setupServices(); err != nil {
		panic(err)
	}
	if err := setupAdapters(ctx); err != nil {
		panic(err)
	}
	if err := api.SetupControllers(); err != nil {
		panic(err)
	}
	// store, err := redis.NewStore(10, "tcp", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_USER"), os.Getenv("REDIS_PWD"))
	// if err != nil {
	// 	panic(err)
	// }

	if err := registerBackgroundJobs(ctx); err != nil {
		panic(err)
	}
	go startJobScheduler(ctx)
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
