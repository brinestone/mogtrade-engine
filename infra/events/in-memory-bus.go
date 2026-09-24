package events

import (
	"context"
	"sync"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
)

type memoryEventBus struct {
	mu      sync.RWMutex
	subs    map[string][]contract.DataChannel
	context context.Context
}

func (m *memoryEventBus) Subscribe(topic string) contract.DataChannel {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(contract.DataChannel)
	m.subs[topic] = append(m.subs[topic], ch)
	go func(ctx context.Context) {
		<-ctx.Done()
		close(ch)
	}(m.context)
	return ch
}

func (m *memoryEventBus) Publish(topic string, data any) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if channels, found := m.subs[topic]; found {
		go func(chans []contract.DataChannel, ev contract.Event) {
			for _, ch := range chans {
				ch <- ev
			}
		}(channels, contract.Event{Data: data, Timestamp: time.Now()})
	}
}

func (m *memoryEventBus) Context() context.Context {
	return m.context
}

func UseInMemoryEventBus(ctx context.Context) contract.EventBus {
	bus := &memoryEventBus{
		context: ctx,
		mu:      sync.RWMutex{},
		subs:    make(map[string][]contract.DataChannel),
	}
	return bus
}
