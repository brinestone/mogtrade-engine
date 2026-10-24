package helpers

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/http"

	"github.com/brinestone/mogtrade/infra/db"
	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go-slim.dev/ioc"
)

func HashThese(args ...string) string {
	cipher := sha1.New()
	for _, arg := range args {
		cipher.Write([]byte(arg))
	}
	return hex.EncodeToString(cipher.Sum([]byte{}))
}

func Ptr[T any](v T) *T {
	return &v
}

type TransactionRepoHandler func(*db.Queries) error
type ValueTransactionRepoHandler[T any] func(*db.Queries) (T, error)

func WithTransactionRepo(ctx context.Context, handler TransactionRepoHandler) error {
	return WithTransaction(ctx, func(tx pgx.Tx) error {
		return handler(db.New(tx))
	})
}

func WithValueTransactionRepo[T any](ctx context.Context, handler ValueTransactionRepoHandler[T]) (T, error) {
	return WithValueTransaction(ctx, func(tx pgx.Tx) (T, error) {
		return handler(db.New(tx))
	})
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
