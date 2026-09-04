package contract

import (
	"context"
	"sync"
	"time"

	"github.com/brinestone/mogtrade/infra/events"
)

type memoryEventBus struct {
	mu       sync.RWMutex
	subs     map[string][]events.DataChannel
	_Context context.Context
}

func (m *memoryEventBus) Subscribe(topic string) events.DataChannel {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(events.DataChannel)
	m.subs[topic] = append(m.subs[topic], ch)
	go func(ctx context.Context) {
		defer close(ch)
		<-ctx.Done()
	}(m._Context)
	return ch
}

func (m *memoryEventBus) Publish(topic string, data any) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if channels, found := m.subs[topic]; found {
		go func(chans []events.DataChannel, ev events.Event) {
			for _, ch := range chans {
				ch <- ev
			}
		}(channels, events.Event{Data: data, Timestamp: time.Now()})
	}
}

func UseInMemoryEventBus(ctx context.Context) events.EventBus {
	bus := &memoryEventBus{
		mu:       sync.RWMutex{},
		_Context: ctx,
		subs:     make(map[string][]events.DataChannel),
	}
	return bus
}
