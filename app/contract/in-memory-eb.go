package contract

import (
	"sync"
	"time"

	"github.com/brinestone/mogtrade/infra/events"
)

type memoryEventBus struct {
	mu   sync.RWMutex
	subs map[string][]events.DataChannel
}

func (m *memoryEventBus) Subscribe(topic string) events.DataChannel {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(events.DataChannel)
	m.subs[topic] = append(m.subs[topic], ch)
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

func UseInMemoryEventBus() events.EventBus {
	return &memoryEventBus{
		mu:   sync.RWMutex{},
		subs: make(map[string][]events.DataChannel),
	}
}
