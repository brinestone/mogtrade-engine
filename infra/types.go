package infra

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnProviderFunc func(context.Context) (*pgxpool.Conn, error)
