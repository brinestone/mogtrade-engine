package adapter

import (
	"context"
	"errors"
	"log/slog"
	"regexp"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/robfig/cron/v3"
)

const expressionPattern = `^(?:[1-5]?\d|\*|\*(?:\/[1-5]?\d)?|(?:[1-5]?\d)-(?:[1-5]?\d)(?:\/[1-5]?\d)?)(?:,(?:[1-5]?\d|\*(?:\/[1-5]?\d)?|(?:[1-5]?\d)-(?:[1-5]?\d)(?:\/[1-5]?\d)?))*\s+(?:[1-2]?\d|\*|\*(?:\/[1-2]?\d)?|(?:[1-2]?\d)-(?:[1-2]?\d)(?:\/[1-2]?\d)?)(?:,(?:[1-2]?\d|\*(?:\/[1-2]?\d)?|(?:[1-2]?\d)-(?:[1-2]?\d)(?:\/[1-2]?\d)?))*\s+(?:[1-3]?\d|\*|\*(?:\/[1-3]?\d)?|(?:[1-3]?\d)-(?:[1-3]?\d)(?:\/[1-3]?\d)?)(?:,(?:[1-3]?\d|\*(?:\/[1-3]?\d)?|(?:[1-3]?\d)-(?:[1-3]?\d)(?:\/[1-3]?\d)?))*\s+(?:1[0-2]|\d|\*|\*(?:\/1[0-2]|\/\d)?|(?:1[0-2]|\d)-(?:1[0-2]|\d)(?:\/1[0-2]|\/\d)?)(?:,(?:1[0-2]|\d|\*(?:\/1[0-2]|\/\d)?|(?:1[0-2]|\d)-(?:1[0-2]|\d)(?:\/1[0-2]|\/\d)?))*\s+(?:[0-7]|\*|\*(?:\/[0-7])?|(?:[0-7])-(?:[0-7])(?:\/[0-7])?)(?:,(?:[0-7]|\*|\*(?:\/[0-7])?|(?:[0-7])-(?:[0-7])(?:\/[0-7])?))*$`

var (
	ErrInvalidCronPattern = errors.New("invalid crontab value")
)

type CronJobScheduler struct {
	cron    *cron.Cron
	jobs    []contract.Job
	context context.Context
	logger  *slog.Logger
}

func NewCronJobScheduler(ctx context.Context, l *slog.Logger) *CronJobScheduler {
	return &CronJobScheduler{
		cron:    cron.New(),
		jobs:    make([]contract.Job, 0),
		context: ctx,
		logger:  l,
	}
}

func (c *CronJobScheduler) RegisterJob(j contract.Job) error {
	schedule, ok := j.Schedule().(string)
	if !ok {
		return ErrInvalidCronPattern
	}
	ok, _ = regexp.MatchString(expressionPattern, schedule)
	if ok {
		c.jobs = append(c.jobs, j)
	}
	return nil
}

func (c *CronJobScheduler) Start() {
	for _, job := range c.jobs {
		schedule, _ := job.Schedule().(string)
		if _, err := c.cron.AddFunc(schedule, func() {
			if err := job.Run(c.context); err != nil {
				c.logger.Error("job error", "err", err.Error(), "job", job.Name())
			} else {
				c.logger.Debug("job completed successfully", "job", job.Name())
			}
		}); err != nil {
			c.logger.Error("failed to schedule job", "job", job.Name(), "err", err.Error())
		}
	}
	c.cron.Start()
	c.logger.Info("started scheduled jobs")
}
