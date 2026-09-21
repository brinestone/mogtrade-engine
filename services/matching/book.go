package matching

import (
	"container/list"
	"sync"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/shopspring/decimal"
)

type priceLevel struct {
	TotalVolume decimal.Decimal
	Orders      *list.List
}

type PriceLevel struct {
	Valid       bool
	TotalVolume decimal.Decimal
	Orders      []OrderEntry
	Price       decimal.Decimal
}

type OrderEntry struct {
	OrderId  string
	Quantity decimal.Decimal
}

type OrderBook struct {
	bidBook map[string]*priceLevel
	askBook map[string]*priceLevel
	bidKeys []string
	askKeys []string
	buyMu   sync.Mutex
	sellMu  sync.Mutex
	orders  *list.List
}

func (o *OrderBook) AddOrder(p PlaceMatchOrderParams) {
	var level *priceLevel
	if p.Type == db.OrderSideSell {
		level, found := o.askBook[p.Price.String()]
		if !found {
			o.sellMu.Lock()
			level = &priceLevel{
				Orders:      list.New(),
				TotalVolume: decimal.Zero,
			}
			o.askBook[p.Price.String()] = level
			o.sellMu.Unlock()
		}

	} else {
		level, found := o.bidBook[p.Price.String()]
		if !found {
			o.buyMu.Lock()
			level = &priceLevel{
				Orders:      list.New(),
				TotalVolume: decimal.Zero,
			}
			o.bidBook[p.Price.String()] = level
			o.buyMu.Unlock()
		}
	}
	level.TotalVolume.Add(p.Quantity)
	level.Orders.PushFront(OrderEntry{
		OrderId:  p.OrderId,
		Quantity: p.Quantity,
	})
}

func (o *OrderBook) BestAsk() PriceLevel {
	var min = decimal.NewFromFloat(99999999999999)
	var e *priceLevel

	for key, entry := range o.askBook {
		price, _ := decimal.NewFromString(key)
		if price.LessThan(min) {
			min = price
			e = entry
		}
	}

	result := PriceLevel{}

	if e != nil {
		result.Valid = true
		for n := e.Orders.Front(); n != nil; n = n.Next() {
			entry, ok := n.Value.(OrderEntry)
			if ok {
				result.Orders = append(result.Orders, entry)
			}
		}
		result.Price = min
		result.TotalVolume = e.TotalVolume
	}
	return result
}

func (o *OrderBook) BestBid() PriceLevel {
	var max = decimal.NewFromFloat(-99999999999999)
	var e *priceLevel

	for key, entry := range o.bidBook {
		price, _ := decimal.NewFromString(key)
		if price.GreaterThan(max) {
			max = price
			e = entry
		}
	}

	result := PriceLevel{}

	if e != nil {
		result.Valid = true
		for n := e.Orders.Front(); n != nil; n = n.Next() {
			entry, ok := n.Value.(OrderEntry)
			if ok {
				result.Orders = append(result.Orders, entry)
			}
		}
		result.Price = max
		result.TotalVolume = e.TotalVolume
	}
	return result
}

func (o *OrderBook) findFills() {
}

func NewBook() *OrderBook {
	return &OrderBook{
		bidBook: make(map[string]*priceLevel),
		askBook: make(map[string]*priceLevel),
		orders:  list.New(),
	}
}
