package contract

import (
	"log/slog"

	db "github.com/brinestone/mogtrade/internal/models"
	"github.com/golobby/container/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UsesIoc struct {
	Ioc *container.Container
}
type UsesDatabase interface {
	GetConn() (*pgxpool.Conn, error)
}
type UsesRepository interface {
	GetRepo() *db.Queries
}
type UsesLogger interface {
	GetLogger() *slog.Logger
	GetLoggerGroup(group string) *slog.Logger
}
