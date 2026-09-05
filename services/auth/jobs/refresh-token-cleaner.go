package jobs

import (
	"context"
	"log/slog"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type refreshTokenRemoverJob struct {
	schedule string
	pool     *pgxpool.Pool
	logger   *slog.Logger
}

func (j *refreshTokenRemoverJob) Run(ctx context.Context) error {
	j.logger.Info("removing stale refresh tokens")
	j.logger.Debug("acquiring database transaction")
	tx, err := j.pool.Begin(ctx)
	if err != nil {
		j.logger.Error("could not open transaction", "err", err.Error())
		return err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			j.logger.Error("could not rollback transaction", "err", err.Error())
		}
	}()

	q := db.New(tx)
	err = q.RemoveStaleRefreshTokens(ctx)
	if err != nil {
		j.logger.Error("could not remove stale refresh tokens", "err", err.Error())
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		j.logger.Error("could not commit transaction", "err", err.Error())
	}
	return err
}

func (j *refreshTokenRemoverJob) Schedule() any {
	return j.schedule
}

func (j *refreshTokenRemoverJob) Name() string {
	return "jobs.token-remover.refresh"
}

func NewStaleTokenRemoverJob(schedule string, p *pgxpool.Pool, l *slog.Logger) contract.Job {
	return &refreshTokenRemoverJob{
		schedule, p, l,
	}
}
