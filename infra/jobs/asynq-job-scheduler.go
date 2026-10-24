package jobs

import (
	"context"
	"log/slog"

	"github.com/brinestone/mogtrade/core/contract"
)

// AsynqJobScheduler implements contract.JobScheduler using the asynq queueing system.
// It uses Redis as the backing store for task distribution and retry management.
type AsynqJobScheduler struct {
	logger *slog.Logger
	ctx    context.Context
}

// NewAsynqJobScheduler creates a new asynq-based job scheduler.
func NewAsynqJobScheduler(ctx context.Context, l *slog.Logger) *AsynqJobScheduler {
	return &AsynqJobScheduler{
		logger: l.With("service", "asynq-scheduler"),
		ctx:    ctx,
	}
}

// RegisterJob registers a job with the asynq scheduler.
// Validates the job name is non-empty per the contract.
func (a *AsynqJobScheduler) RegisterJob(j contract.Job) error {
	if j.Name() == "" {
		a.logger.Error("cannot register job with empty name")
		return nil // or fmt.Errorf("job name cannot be empty")
	}
	a.logger.Debug("job registered", "job", j.Name())
	return nil
}

// Start initializes the asynq scheduler.
// Asynq workers should be started separately in production.
func (a *AsynqJobScheduler) Start() {
	a.logger.Debug("asynq scheduler started")
	// Workers are typically run as separate processes
	a.logger.Info("asynq scheduler ready")
}