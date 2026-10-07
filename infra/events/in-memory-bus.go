package events

import (
	"context"
	"sync"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
)

type memoryEventBus struct {
	mu   sync.RWMutex
	subs map[string][]contract.DataChannel
}

func (m *memoryEventBus) Subscribe(ctx context.Context, topic string) <-chan contract.Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(contract.DataChannel)
	m.subs[topic] = append(m.subs[topic], ch)
	go func(ctx context.Context) {
		<-ctx.Done()
		close(ch)
	}(ctx)
	return ch
}

func (m *memoryEventBus) Publish(topic string, data any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if channels, found := m.subs[topic]; found {
		go func(chans []contract.DataChannel, ev contract.Event) {
			for _, ch := range chans {
				ch <- ev
			}
		}(channels, contract.Event{Data: data, Timestamp: time.Now()})
	}
	return nil
}

func UseInMemoryEventBus() contract.EventBus {
	bus := &memoryEventBus{
		mu:   sync.RWMutex{},
		subs: make(map[string][]contract.DataChannel),
	}
	return bus
}
