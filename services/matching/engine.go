package matching

import (
	"container/heap"
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/shopspring/decimal"
)

const (
	EventKeyOrderMatched string = "orders.match"
)

type Engine struct {
	logger  *slog.Logger
	booksMu sync.Mutex
	books   map[string]*OrderBook
	bus     contract.EventBus
}

type PlaceMatchOrderParams struct {
	OrderId  string
	Symbol   string
	Type     db.OrderSide
	Quantity decimal.Decimal
	Price    decimal.Decimal
}

type OrderMatched struct {
	BuyOrder          string
	SellOrder         string
	Symbol            string
	MatchedAt         time.Time
	MatchQuantity     decimal.Decimal
	RemainingQuantity decimal.Decimal
}

func (e *Engine) PlaceMatchOrder(p PlaceMatchOrderParams) {
	e.logger.Info("PlaceMatchOrder called", "symbol", p.Symbol, "orderId", p.OrderId, "quantity", p.Quantity, "type", p.Type)
	e.booksMu.Lock()
	book, found := e.books[p.Symbol]
	if !found {
		book = NewBook()
		e.books[p.Symbol] = book
		e.logger.Info("Created new order book", "symbol", p.Symbol)
	}
	e.booksMu.Unlock()

	book.AddOrder(p)
	e.logger.Info("Order added to book", "symbol", p.Symbol, "orderId", p.OrderId, "quantity", p.Quantity)

	e.logger.Info("finding matches", "symbol", p.Symbol)
	go e.findMatches(p.Symbol)
}

func (e *Engine) CancelMatchOrder(orderId string, symbol string, side db.OrderSide, price decimal.Decimal) (bool, decimal.Decimal) {
	e.booksMu.Lock()
	book, found := e.books[symbol]
	e.booksMu.Unlock()
	if !found {
		return false, decimal.Zero
	}
	return book.RemoveOrder(orderId, side, price)
}

// findMatches continuously matches crossing orders at the top of the book
// (BestBid >= BestAsk) according to Price-Time Priority (FIFO) until the spread
// uncrosses or one side of the book is empty.
func (e *Engine) findMatches(symbol string) {
	e.booksMu.Lock()
	book, ok := e.books[symbol]
	e.booksMu.Unlock()
	if !ok {
		return
	}

	// Always acquire internal mutexes in a consistent order: sellMu → buyMu.
	book.sellMu.Lock()
	book.buyMu.Lock()
	defer book.buyMu.Unlock()
	defer book.sellMu.Unlock()

	for {
		bestAskPrice := book.sellPrices.Peek()
		bestBidPrice := book.bidPrices.Peek()

		if !bestAskPrice.Valid || !bestBidPrice.Valid {
			// One side is empty: no matches possible.
			break
		}

		// Price crossing condition: Highest Bid >= Lowest Ask
		if bestBidPrice.Decimal.LessThan(bestAskPrice.Decimal) {
			// Spread is positive (bid < ask): no crossing orders.
			break
		}

		askKey := bestAskPrice.Decimal.String()
		bidKey := bestBidPrice.Decimal.String()

		askLevel, hasAsk := book.asks[askKey]
		bidLevel, hasBid := book.bids[bidKey]

		// Purge stale empty price levels from heaps if any
		if !hasAsk || askLevel.Orders.Len() == 0 {
			if hasAsk {
				delete(book.asks, askKey)
			}
			heap.Pop(book.sellPrices)
			continue
		}

		if !hasBid || bidLevel.Orders.Len() == 0 {
			if hasBid {
				delete(book.bids, bidKey)
			}
			heap.Pop(book.bidPrices)
			continue
		}

		// Match the front of each queue (Price-Time Priority)
		bidNode := bidLevel.Orders.Front()
		askNode := askLevel.Orders.Front()

		bidEntry := bidNode.Value.(OrderEntry)
		askEntry := askNode.Value.(OrderEntry)

		execQty := decimal.Min(bidEntry.Quantity, askEntry.Quantity)
		if execQty.IsZero() {
			break
		}

		// Decrement executed quantities
		bidEntry.Quantity = bidEntry.Quantity.Sub(execQty)
		askEntry.Quantity = askEntry.Quantity.Sub(execQty)
		bidLevel.TotalVolume = bidLevel.TotalVolume.Sub(execQty)
		askLevel.TotalVolume = askLevel.TotalVolume.Sub(execQty)

		bidNode.Value = bidEntry
		askNode.Value = askEntry

		// Emit match event
		e.bus.Publish(EventKeyOrderMatched, OrderMatched{
			BuyOrder:          bidEntry.OrderId,
			SellOrder:         askEntry.OrderId,
			Symbol:            symbol,
			MatchedAt:         time.Now(),
			MatchQuantity:     execQty,
			RemainingQuantity: askEntry.Quantity,
		})

		// Evict fully filled ask order
		if askEntry.Quantity.IsZero() {
			askLevel.Orders.Remove(askNode)
			if askLevel.Orders.Len() == 0 {
				delete(book.asks, askKey)
				heap.Pop(book.sellPrices)
			}
		}

		// Evict fully filled bid order
		if bidEntry.Quantity.IsZero() {
			bidLevel.Orders.Remove(bidNode)
			if bidLevel.Orders.Len() == 0 {
				delete(book.bids, bidKey)
				heap.Pop(book.bidPrices)
			}
		}
	}
}

func (e *Engine) StartAutoMatching(ctx context.Context) {
	t := time.NewTicker(time.Second)
	go func(t *time.Ticker) {
		e.logger.Debug("starting auto matching")
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				e.booksMu.Lock()
				symbols := make([]string, 0, len(e.books))
				for symbol := range e.books {
					symbols = append(symbols, symbol)
				}
				e.booksMu.Unlock()

				for _, symbol := range symbols {
					e.findMatches(symbol)
				}
			}
		}
	}(t)
}

func NewMatchingEngine(l *slog.Logger, eb contract.EventBus) *Engine {
	return &Engine{
		logger:  l.With("service", "matching-engine"),
		booksMu: sync.Mutex{},
		books:   make(map[string]*OrderBook),
		bus:     eb,
	}
}
