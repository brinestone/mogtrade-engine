package contract

import (
	"context"
)

type Job interface {
	Name() string
	Schedule() any
	Run(context.Context) error
}

type JobScheduler interface {
	RegisterJob(Job) error
	Start()
}
