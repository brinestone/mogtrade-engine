package api

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ApiConfig struct {
	// DbUrl string
	dbPool *pgxpool.Pool
}

type usesDatabase interface {
	getConn() (*pgxpool.Conn, error)
}

func (c ApiConfig) getConn(ctx context.Context) (*pgxpool.Conn, error) {
	return c.dbPool.Acquire(ctx)
}
