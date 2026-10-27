package helpers

import (
	"context"

	"github.com/brinestone/mogtrade/core/contract"
	"go-slim.dev/ioc"
)

type EventBusConsumer func(bus contract.EventBus)

func UseEventBus(ctx context.Context, consumer EventBusConsumer) {
	ioc.Invoke(ctx, consumer)
}

func PublishEvent(ctx context.Context, topic string, payload any) error {
	_, err := ioc.Invoke(ctx, func(eb contract.EventBus) {
		eb.Publish(topic, payload)
	})
	return err
}
