package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/brinestone/mogtrade/web/api"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	port int
)

func main() {
	godotenv.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()

	store, err := redis.NewStore(10, "tcp", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_USER"), os.Getenv("REDIS_PWD"))
	if err != nil {
		panic(err)
	}
	bootstrapApplication(ctx)
	engine := gin.Default()
	baseRouter := engine.Group("/api")
	if err := api.MountApiV1(baseRouter, api.ApiConfig{
		Host:           os.Getenv("HOST"),
		SessionStore:   store,
		AllowedOrigins: strings.Split(os.Getenv("ALLOWED_ORIGINS"), ";"),
	}); err != nil {
		panic(err)
	}
	startAsyncTasks(ctx)
	engine.Run(fmt.Sprintf(":%d", port))
}
