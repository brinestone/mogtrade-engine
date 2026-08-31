package api

import (
	"context"

	db "github.com/brinestone/mogtrade/internal/models"
	"github.com/brinestone/mogtrade/libs/contract"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApiConfig struct {
	contract.UsesIoc
}

func (c ApiConfig) GetRepo() *db.Queries {
	var repo *db.Queries
	c.Ioc.Resolve(repo)
	return repo
}

func (c ApiConfig) GetConn(ctx context.Context) (*pgxpool.Conn, error) {
	var pool *pgxpool.Pool
	if err := c.Ioc.Resolve(pool); err != nil {
		return nil, err
	}

	return pool.Acquire(ctx)
}
