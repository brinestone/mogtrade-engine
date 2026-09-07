package market

import (
	"sync"

	"github.com/shopspring/decimal"
)

type ExchangeEvent struct {
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Timestamp int64           `json:"timestamp"`
}

type HubClient chan ExchangeEvent

func NewHubClient() HubClient {
	return make(HubClient, 100)
}

type ExchangeHub struct {
	clients     map[HubClient]bool
	broadcaster HubClient
	register    chan HubClient
	unregister  chan HubClient
	clientsMu   sync.Mutex
}

func (e *ExchangeHub) Run() {
	for {
		select {
		case client := <-e.register:
			e.clientsMu.Lock()
			e.clients[client] = true
			e.clientsMu.Unlock()
		case client := <-e.unregister:
			e.clientsMu.Lock()
			if _, ok := e.clients[client]; ok {
				delete(e.clients, client)
				close(client)
			}
			e.clientsMu.Unlock()
		case event := <-e.broadcaster:
			e.clientsMu.Lock()
			for client := range e.clients {
				select {
				case client <- event:
				default:
					go func(c chan ExchangeEvent) {
						e.unregister <- c
					}(client)
				}
			}
			e.clientsMu.Unlock()
		}
	}
}

func (e *ExchangeHub) UnRegisterClient(c HubClient) {
	e.unregister <- c
}

func (e *ExchangeHub) RegisterClient(c HubClient) {
	e.register <- c
}

func NewExchangeHub() *ExchangeHub {
	return &ExchangeHub{
		clients:     make(map[HubClient]bool),
		broadcaster: make(chan ExchangeEvent),
		register:    make(chan HubClient),
		unregister:  make(chan HubClient),
	}
}
