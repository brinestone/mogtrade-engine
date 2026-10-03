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

type ValueTransactionHandler[T any] func(pgx.Tx) (T, error)

func WithValueTransaction[T any](ctx context.Context, handler ValueTransactionHandler[T]) (T, error) {
	var zero T
	pool, err := ioc.Get[*pgxpool.Pool](ctx)
	if err != nil {
		return zero, err
	}
	tx, err := (*pool).Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback(ctx)

	res, err := handler(tx)
	if err != nil {
		return zero, err
	}

	if err = tx.Commit(ctx); err != nil {
		return zero, err
	}
	return res, nil
}

func InternalServerError(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
}

// SliceDiff returns elements in 'a' that are not in 'b'
func SliceDiff[T comparable](a, b []T) []T {
	bMap := make(map[T]struct{}, len(b))
	for _, val := range b {
		bMap[val] = struct{}{}
	}

	var diff []T
	for _, val := range a {
		if _, found := bMap[val]; !found {
			diff = append(diff, val)
		}
	}

	return diff
}
