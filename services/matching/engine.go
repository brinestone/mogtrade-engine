package matching

import (
	"log/slog"
	"sync"
	"time"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/infra/events"
	"github.com/shopspring/decimal"
)

const (
	EventKeyOrderMatched string = "orders.match"
)

type MatchingEngine struct {
	logger      *slog.Logger
	orderBookMu sync.Mutex
	orderBook   map[string]*OrderBook
	eb          events.EventBus
}

type PlaceMatchOrderParams struct {
	OrderId  string
	Symbol   string
	Type     db.OrderSide
	Quantity decimal.Decimal
	Price    decimal.Decimal
}

type OrderMatched struct {
	BuyOrder  string
	SellOrder string
	Symbol    string
	MatchedAt time.Time
}

func (e *MatchingEngine) PlaceMatchOrder(p PlaceMatchOrderParams) {
	// acquire / create the OrderBook for this symbol
	e.logger.Info("PlaceMatchOrder called", "symbol", p.Symbol, "orderId", p.OrderId, "quantity", p.Quantity, "type", p.Type)
	book, found := e.orderBook[p.Symbol]
	if !found {
		e.orderBookMu.Lock()
		// double‑check under the mutex
		book, found = e.orderBook[p.Symbol]
		if !found {
			book = NewBook()
			e.orderBook[p.Symbol] = book
			e.logger.Info("Created new OrderBook for symbol", "symbol", p.Symbol)
		}
		e.orderBookMu.Unlock()
	}

	// add the order to the book (outer lock already released)
	book.AddOrder(p)
	e.logger.Info("Order added to book", "symbol", p.Symbol, "orderId", p.OrderId, "quantity", p.Quantity)

	// start matching without holding the outer map mutex
	e.logger.Info("Starting findMatches", "symbol", p.Symbol)
	e.findMatches(p.Symbol)
	e.logger.Info("findMatches completed", "symbol", p.Symbol)
}

// findMatches attempts to match as much as possible against the opposite side
// of the order book for the given symbol. It respects price‑time priority,
// updates quantities, removes fully‑filled orders, and emits a single
// OrderMatchEvent on the engine's matchCh channel.
func (e *MatchingEngine) findMatches(symbol string) {
	// 1. Acquire outer map mutex just long enough to fetch the *OrderBook.
	e.logger.Info("findMatches: acquiring outer map mutex", "symbol", symbol)
	e.orderBookMu.Lock()
	book, ok := e.orderBook[symbol]
	if !ok {
		e.logger.Info("findMatches: no order book for symbol", "symbol", symbol)
		e.orderBookMu.Unlock()
		return
	}
	e.logger.Info("findMatches: obtained OrderBook", "symbol", symbol)

	// 2. Acquire internal mutexes in the safe order: sellMu → buyMu.
	e.logger.Info("findMatches: acquiring internal mutexes", "symbol", symbol)
	book.sellMu.Lock()
	book.buyMu.Lock()
	// release in reverse order via defer.
	defer book.buyMu.Unlock()
	defer book.sellMu.Unlock()

	// 3. Retrieve the best price levels (snapshots).
	e.logger.Info("findMatches: retrieving best price levels", "symbol", symbol)
	bestAsk := book.BestAsk()
	bestBid := book.BestBid()
	if !bestAsk.Valid || !bestBid.Valid {
		// one side of the book is empty – nothing to match.
		e.logger.Info("findMatches: no valid best price levels", "symbol", symbol)
		return
	}

	// 4. Compute the maximum quantity that can be matched at these prices.
	//    decimal.Decimal has LessThan but no Min; we do it manually.
	e.logger.Info("findMatches: computing max match quantity", "symbol", symbol,
		"bestAskVolume", bestAsk.TotalVolume, "bestBidVolume", bestBid.TotalVolume)
	var matchQty decimal.Decimal
	if bestAsk.TotalVolume.LessThan(bestBid.TotalVolume) {
		matchQty = bestAsk.TotalVolume
	} else {
		matchQty = bestBid.TotalVolume
	}
	if matchQty.IsZero() {
		e.logger.Info("findMatches: max match quantity is zero", "symbol", symbol)
		return
	}

	// Containers for the first order ID matched on each side.
	var matchedSellOrder, matchedBuyOrder string

	// -------------------------------------------------------
	// 5. Match from the ask side (sellers) – FIFO using the
	//    internal *list.List stored in the book's askLevel.
	// -------------------------------------------------------
	askPriceKey := bestAsk.Price.String()
	e.logger.Info("findMatches: matching ask side", "symbol", symbol,
		"askPriceKey", askPriceKey, "matchQty", matchQty)
	askLevel, ok := book.asks[askPriceKey]
	if !ok {
		// should not happen, but bail out cleanly.
		e.logger.Info("findMatches: ask level not found", "symbol", symbol)
		return
	}
	// askLevel.Orders is a *list.List; we iterate and remove nodes.
	for eIdx := askLevel.Orders.Front(); eIdx != nil && !matchQty.IsZero(); {
		orderEntry := eIdx.Value.(OrderEntry) // OrderEntry{OrderId, Quantity}
		// determine executed quantity as min(matchQty, orderEntry.Quantity)
		var execQty decimal.Decimal
		if orderEntry.Quantity.LessThan(matchQty) {
			execQty = orderEntry.Quantity
		} else {
			execQty = matchQty
		}
		// update the order's remaining quantity.
		orderEntry.Quantity = orderEntry.Quantity.Sub(execQty)
		// persist the updated value back into the list node.
		eIdx.Value = orderEntry
		if matchedSellOrder == "" {
			matchedSellOrder = orderEntry.OrderId
		}
		matchQty = matchQty.Sub(execQty)
		// update the price‑level's total volume.
		askLevel.TotalVolume = askLevel.TotalVolume.Sub(execQty)
		e.logger.Info("findMatches: ask order updated", "symbol", symbol,
			"orderId", orderEntry.OrderId, "executedQty", execQty, "remainingQty", orderEntry.Quantity)
		if orderEntry.Quantity.IsZero() {
			// remove the node from the list and advance.
			next := eIdx.Next()
			askLevel.Orders.Remove(eIdx)
			eIdx = next
			continue
		}
		eIdx = eIdx.Next()
	}

	// -------------------------------------------------------
	// 6. If any quantity remains, match from the bid side (buyers).
	// -------------------------------------------------------
	if !matchQty.IsZero() {
		bidPriceKey := bestBid.Price.String()
		e.logger.Info("findMatches: matching bid side", "symbol", symbol,
			"bidPriceKey", bidPriceKey, "matchQty", matchQty)
		bidLevel, ok := book.bids[bidPriceKey]
		if !ok {
			// should not happen; bail out.
			e.logger.Info("findMatches: bid level not found", "symbol", symbol)
			return
		}
		for eIdx := bidLevel.Orders.Front(); eIdx != nil && !matchQty.IsZero(); {
			orderEntry := eIdx.Value.(OrderEntry)
			// determine executed quantity
			var execQty decimal.Decimal
			if orderEntry.Quantity.LessThan(matchQty) {
				execQty = orderEntry.Quantity
			} else {
				execQty = matchQty
			}
			orderEntry.Quantity = orderEntry.Quantity.Sub(execQty)
			// persist the updated value back into the list node.
			eIdx.Value = orderEntry
			if matchedBuyOrder == "" {
				matchedBuyOrder = orderEntry.OrderId
			}
			matchQty = matchQty.Sub(execQty)
			bidLevel.TotalVolume = bidLevel.TotalVolume.Sub(execQty)
			e.logger.Info("findMatches: bid order updated", "symbol", symbol,
				"orderId", orderEntry.OrderId, "executedQty", execQty, "remainingQty", orderEntry.Quantity)
			if orderEntry.Quantity.IsZero() {
				next := eIdx.Next()
				bidLevel.Orders.Remove(eIdx)
				eIdx = next
				continue
			}
			eIdx = eIdx.Next()
		}
	}

	// -------------------------------------------------------
	// 7. Emit a single match event (non‑blocking) on the channel.
	// -------------------------------------------------------
	e.logger.Info("findMatches: emitting match event", "symbol", symbol,
		"matchedSellOrder", matchedSellOrder, "matchedBuyOrder", matchedBuyOrder)
	if matchedSellOrder != "" || matchedBuyOrder != "" {
		ev := OrderMatched{
			BuyOrder:  matchedBuyOrder,
			SellOrder: matchedSellOrder,
			Symbol:    symbol,
			MatchedAt: time.Now(),
		}
		e.eb.Publish(EventKeyOrderMatched, ev)
	}

	// 8. Release the outer map mutex.
	e.logger.Info("findMatches: releasing outer map mutex", "symbol", symbol)
	e.orderBookMu.Unlock()
}

func NewMatchingEngine(l *slog.Logger, eb events.EventBus) *MatchingEngine {
	return &MatchingEngine{
		logger:      l.With("service", "matching-engine"),
		orderBookMu: sync.Mutex{},
		orderBook:   make(map[string]*OrderBook),
		eb:          eb,
	}
}
