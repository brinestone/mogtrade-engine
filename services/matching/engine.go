package matching

import (
	"log/slog"
	"sync"
	"time"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/shopspring/decimal"
)

type MatchingEngine struct {
	logger      *slog.Logger
	orderBookMu sync.Mutex
	orderBook   map[string]*OrderBook
	matchCh     chan OrderMatchEvent
}

type PlaceMatchOrderParams struct {
	OrderId  string
	Symbol   string
	Type     db.OrderSide
	Quantity decimal.Decimal
	Price    decimal.Decimal
}

type OrderMatchEvent struct {
	BuyOrder  string
	SellOrder string
	Symbol    string
	MatchedAt time.Time
}

func (e *MatchingEngine) PlaceMatchOrder(p PlaceMatchOrderParams) {
	book, found := e.orderBook[p.Symbol]
	if !found {
		book = NewBook()
		e.orderBookMu.Lock()
		e.orderBook[p.Symbol] = book
		e.orderBookMu.Unlock()
	}

	book.AddOrder(p)
	e.findMatches()
}

func (e *MatchingEngine) findMatches() {
}

func NewMatchingEngine(l *slog.Logger) *MatchingEngine {
	return &MatchingEngine{
		logger:      l.With("service", "matching-engine"),
		orderBookMu: sync.Mutex{},
		orderBook:   make(map[string]*OrderBook),
	}
}
