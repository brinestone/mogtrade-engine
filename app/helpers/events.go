package helpers

import (
	"context"

	"github.com/brinestone/mogtrade/infra/events"
	"go-slim.dev/ioc"
)

func PublishEvent(ctx context.Context, topic string, payload any) error {
	_, err := ioc.Invoke(ctx, func(eb events.EventBus) {
		eb.Publish(topic, payload)
	})
	return err
}
