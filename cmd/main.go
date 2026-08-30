package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/brinestone/mogtrade/libs/api"
	"github.com/gin-gonic/gin"
)

var (
	port int
)

func main() {
	ctx, canceller := context.WithCancel(context.Background())
	defer canceller()
	err := parseVars()
	if err != nil {
		panic(err)
	}
	engine := gin.Default()
	baseRouter := engine.Group("/api")
	cfg, err := api.ExtractApiConfigFromEnv(ctx)
	api.MountApiV1(baseRouter, cfg)
	engine.Run(fmt.Sprintf(":%d", port))
}

func parseVars() error {
	_port, err := strconv.ParseInt(os.Getenv("PORT"), 10, 32)
	if err != nil {
		return err
	}
	port = int(_port)
	return nil
}
