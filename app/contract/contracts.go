package contract

import (
	"log/slog"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UsesDatabase interface {
	GetConn() (*pgxpool.Conn, error)
}
type UsesQueries interface {
	GetQueries() *db.Queries
}
type UsesLogger interface {
	GetLogger() *slog.Logger
	GetLoggerGroup(group string) *slog.Logger
}
