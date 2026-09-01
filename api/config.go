package api

import (
	"context"
	"fmt"
	"log/slog"

	db "github.com/brinestone/mogtrade/internal/models"
	"github.com/brinestone/mogtrade/libs/contract"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApiConfig struct {
	contract.UsesIoc
}

func (c ApiConfig) GetLoggerWith(args ...any) *slog.Logger {
	fmt.Printf("%v", c.UsesIoc)
	return c.GetLogger().With(args...)
}

func (c ApiConfig) GetLoggerGroup(group string) *slog.Logger {
	return c.GetLogger().WithGroup(group)
}

func (c ApiConfig) GetLogger() *slog.Logger {
	var logger slog.Logger

	if err := c.Ioc.Resolve(&logger); err != nil {
		panic(err)
	}

	return &logger
}

func (c ApiConfig) GetRepo() *db.Queries {
	var repo db.Queries
	c.Ioc.Resolve(&repo)
	return &repo
}

func (c ApiConfig) GetConn(ctx context.Context) (*pgxpool.Conn, error) {
	var pool *pgxpool.Pool
	if err := c.Ioc.Resolve(pool); err != nil {
		return nil, err
	}

	return pool.Acquire(ctx)
}
