package contract

import (
	"context"
	"time"
)

type Event struct {
	Data      any
	Timestamp time.Time
}
type DataChannel chan Event
type EventBus interface {
	Subscribe(context.Context, string) <-chan Event
	Publish(string, any) error
}
