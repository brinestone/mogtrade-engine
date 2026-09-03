package events

import "time"

type Event struct {
	Data      any
	Timestamp time.Time
}
type DataChannel chan Event
type EventBus interface {
	Subscribe(string) DataChannel
	Publish(string, any)
}
