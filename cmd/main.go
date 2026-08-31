package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	db "github.com/brinestone/mogtrade/internal/models"
	"github.com/brinestone/mogtrade/libs/api"
	"github.com/brinestone/mogtrade/libs/contract"
	"github.com/gin-gonic/gin"
	"github.com/golobby/container/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	port int
	ioc  container.Container = make(container.Container)
)

func main() {
	ctx, canceller := context.WithCancel(context.Background())
	defer canceller()
	if err := parseVars(); err != nil {
		panic(err)
	}
	if err := setupDiContainer(ctx); err != nil {
		panic(err)
	}
	engine := gin.Default()
	baseRouter := engine.Group("/api")
	if err := api.SetupControllersDi(api.ApiConfig{
		UsesIoc: contract.UsesIoc{
			Ioc: &ioc,
		},
	}); err != nil {
		panic(err)
	}
	api.MountApiV1(baseRouter, &ioc)
	engine.Run(fmt.Sprintf(":%d", port))
}

func setupDiContainer(ctx context.Context) error {
	ioc.Singleton(func() (*pgxpool.Pool, error) {
		pool, err := pgxpool.New(ctx, os.Getenv("DB_URL"))
		if err != nil {
			return nil, err
		}
		if err = pool.Ping(ctx); err != nil {
			return nil, err
		}
		return pool, nil
	})
	ioc.SingletonLazy(func(pool *pgxpool.Pool) *db.Queries {
		return db.New(pool)
	})
	return nil
}

func parseVars() error {
	_port, err := strconv.ParseInt(os.Getenv("PORT"), 10, 32)
	if err != nil {
		return err
	}
	port = int(_port)
	return nil
}
