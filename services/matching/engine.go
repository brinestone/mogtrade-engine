package matching

import (
	"context"
	"log/slog"
	"slices"
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
	logger      *slog.Logger
	orderBookMu sync.Mutex
	orderBook   map[string]*OrderBook
	bus         contract.EventBus
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
	e.logger.Info("finding matches", "symbol", p.Symbol)
	// go e.findMatches(p.Symbol)
	go e.findMatches2(p.Symbol)
}

// findMatches attempts to match as much as possible against the opposite side
// of the order book for the given symbol. It respects price‑time priority,
// updates quantities, removes fully‑filled orders, and emits a single
// OrderMatchEvent on the engine's matchCh channel.
func (e *Engine) findMatches(symbol string) {
	// 1. Acquire outer map mutex just long enough to fetch the *OrderBook.
	e.logger.Info("findMatches: acquiring outer map mutex", "symbol", symbol)
	hasMutex := e.orderBookMu.TryLock()
	if !hasMutex {
		e.logger.Info("findMatches: book mutex is busy. aborting", "symbol", symbol)
		return
	}
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
		e.bus.Publish(EventKeyOrderMatched, ev)
	}

	// 8. Release the outer map mutex.
	e.logger.Info("findMatches: releasing outer map mutex", "symbol", symbol)
	e.orderBookMu.Unlock()
}

func (e *Engine) findMatches2(symbol string) {
	l := e.logger.With("symbol", symbol)
	hasMutex := e.orderBookMu.TryLock()
	if !hasMutex {
		l.Info("book mutex is busy, aborting")
		return
	}
	book, found := e.orderBook[symbol]
	if !found {
		l.Info("no order book found")
		e.orderBookMu.Unlock()
		return
	}
	l.Info("obtained order book")

	l.Info("acquiring internal mutexes")
	book.sellMu.Lock()
	book.buyMu.Lock()
	defer book.buyMu.Unlock()
	defer book.sellMu.Unlock()

	markedForRemoval := make([]int, 0)
	for i, price := range *book.bidPrices {
		levelKey := price.String()
		bidLevel, found := book.bids[levelKey]
		if !found {
			l.Warn("bid price was found in bid price list but it has no price level. Marking for removing from price list", "bid-price", price)
			markedForRemoval = append(markedForRemoval, i)
			continue
		}

		l.Info("finding ask matches for bid", "bid-price", price)
		for eIdx := bidLevel.Orders.Front(); eIdx != nil; {
			markedForRemoval := make([]int, 0)
			bidEntry := eIdx.Value.(OrderEntry)
			for i, askPrice := range *book.sellPrices {
				if askPrice.GreaterThan(price) {
					l.Warn("there are no more asks for bids")
					break
				}
				askKey := askPrice.String()
				askLevel, found := book.asks[askKey]
				if !found {
					l.Warn("ask price was in sell price list but it has no price level. Marking for removing from price list", "ask-price", askPrice)
					markedForRemoval = append(markedForRemoval, i)
					continue
				}

				matchQty := decimal.Min(bidEntry.Quantity, askLevel.TotalVolume)
				for askOrderItem := askLevel.Orders.Front(); askOrderItem != nil && !matchQty.IsZero(); {
					askEntry := askOrderItem.Value.(OrderEntry)
					execQty := decimal.Min(matchQty, askEntry.Quantity)
					askEntry.Quantity = askEntry.Quantity.Sub(execQty)
					bidEntry.Quantity = bidEntry.Quantity.Sub(execQty)
					matchQty = matchQty.Sub(execQty)
					askLevel.TotalVolume = askLevel.TotalVolume.Sub(execQty)
					bidLevel.TotalVolume = bidLevel.TotalVolume.Sub(execQty)
					askOrderItem.Value = askEntry
					e.bus.Publish(EventKeyOrderMatched, OrderMatched{
						BuyOrder:          bidEntry.OrderId,
						SellOrder:         askEntry.OrderId,
						Symbol:            symbol,
						MatchedAt:         time.Now(),
						MatchQuantity:     execQty,
						RemainingQuantity: askEntry.Quantity.Sub(execQty),
					})
					if bidLevel.TotalVolume.IsZero() || bidEntry.Quantity.IsZero() {
						break
					}
					if askEntry.Quantity.IsZero() {
						next := askOrderItem.Next()
						askLevel.Orders.Remove(askOrderItem)
						askOrderItem = next
						continue
					}
					askOrderItem = askOrderItem.Next()
				}
				if len(markedForRemoval) > 0 {
					newAskPriceList := make(sellPriceList, len(*book.sellPrices)-len(markedForRemoval))
					for i, p := range *book.sellPrices {
						if slices.Contains(markedForRemoval, i) {
							continue
						}
						(&newAskPriceList).Push(p)
					}
					book.sellPrices = &newAskPriceList
				}

				if bidLevel.TotalVolume.IsZero() || bidEntry.Quantity.IsZero() {
					break
				}
			}
			if bidEntry.Quantity.IsZero() {
				next := eIdx.Next()
				bidLevel.Orders.Remove(eIdx)
				eIdx = next
				continue
			}
			eIdx = eIdx.Next()
		}
	}
	if len(markedForRemoval) > 0 {
		newBidPriceList := make(buyPriceList, len(*book.bidPrices)-len(markedForRemoval))
		for i, p := range *book.bidPrices {
			if slices.Contains(markedForRemoval, i) {
				continue
			}
			(&newBidPriceList).Push(p)
		}
		book.bidPrices = &newBidPriceList
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
				if len(e.orderBook) <= 0 {
					continue
				}
				hasMutex := e.orderBookMu.TryLock()
				if !hasMutex {
					e.logger.Warn("book mutex is busy, skipping auto matching")
					continue
				}
				for symbol := range e.orderBook {
					e.findMatches(symbol)
				}
				e.orderBookMu.Unlock()
			}
		}
	}(t)
}

func NewMatchingEngine(l *slog.Logger, eb contract.EventBus) *Engine {
	return &Engine{
		logger:      l.With("service", "matching-engine"),
		orderBookMu: sync.Mutex{},
		orderBook:   make(map[string]*OrderBook),
		bus:         eb,
	}
}
