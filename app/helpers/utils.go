package helpers

import (
	"context"
	"net/http"

	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go-slim.dev/ioc"
)

func Ptr[T any](v T) *T {
	return &v
}

type TransactionHandler func(pgx.Tx) error

func WithTransaction(ctx context.Context, handler TransactionHandler) error {
	pool, err := ioc.Get[*pgxpool.Pool](ctx)
	if err != nil {
		return err
	}
	tx, err := (*pool).Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	err = handler(tx)
	if err == nil {
		return tx.Commit(ctx)
	} else {
		return err
	}
}

func InternalServerError(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
}
