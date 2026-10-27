package contract

import (
	"context"
	"time"
)

type BusEvent struct {
	Data      any
	Timestamp time.Time
}
type DataChannel chan BusEvent
type EventBus interface {
	Subscribe(context.Context, string) <-chan BusEvent
	Publish(string, any) error
}
