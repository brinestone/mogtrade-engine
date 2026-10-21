package matching

import (
	"container/heap"
	"container/list"
	"sync"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/shopspring/decimal"
)

type sellPriceList []decimal.Decimal

func (l sellPriceList) Len() int           { return len(l) }
func (l sellPriceList) Less(i, j int) bool { return l[i].LessThan(l[j]) }
func (l sellPriceList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l *sellPriceList) Peek() decimal.NullDecimal {
	if len(*l) == 0 {
		return decimal.NullDecimal{}
	}
	return decimal.NewNullDecimal((*l)[0])
}
func (l *sellPriceList) Push(p any) {
	pp, _ := p.(decimal.Decimal)
	*l = append(*l, pp)
}
func (l *sellPriceList) Pop() any {
	old := *l
	n := len(old)
	x := old[n-1]
	*l = old[0 : n-1]
	return x
}

type buyPriceList []decimal.Decimal

func (l buyPriceList) Len() int           { return len(l) }
func (l buyPriceList) Less(i, j int) bool { return l[i].GreaterThan(l[j]) }
func (l buyPriceList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l *buyPriceList) Peek() decimal.NullDecimal {
	if len(*l) == 0 {
		return decimal.NullDecimal{}
	}
	return decimal.NewNullDecimal((*l)[0])
}
func (l *buyPriceList) Push(p any) {
	pp, _ := p.(decimal.Decimal)
	*l = append(*l, pp)
}
func (l *buyPriceList) Pop() any {
	old := *l
	n := len(old)
	x := old[n-1]
	*l = old[0 : n-1]
	return x
}

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
	Side     db.OrderSide
	Quantity decimal.Decimal
}

type OrderBook struct {
	bids       map[string]*priceLevel
	asks       map[string]*priceLevel
	sellPrices *sellPriceList
	bidPrices  *buyPriceList
	buyMu      sync.Mutex
	sellMu     sync.Mutex
	orders     *list.List
}

func (o *OrderBook) AddOrder(p PlaceMatchOrderParams) {
	levelKey := p.Price.String()
	if p.Type == db.OrderSideSell {
		o.sellMu.Lock()
		l, found := o.asks[levelKey]
		if !found {
			l = &priceLevel{
				Orders:      list.New(),
				TotalVolume: decimal.Zero,
			}
			o.asks[levelKey] = l
			heap.Push(o.sellPrices, p.Price)
		}
		l.TotalVolume = l.TotalVolume.Add(p.Quantity)
		l.Orders.PushBack(OrderEntry{
			OrderId:  p.OrderId,
			Side:     p.Type,
			Quantity: p.Quantity,
		})
		o.sellMu.Unlock()
	} else {
		o.buyMu.Lock()
		l, found := o.bids[levelKey]
		if !found {
			l = &priceLevel{
				Orders:      list.New(),
				TotalVolume: decimal.Zero,
			}
			o.bids[levelKey] = l
			heap.Push(o.bidPrices, p.Price)
		}
		l.TotalVolume = l.TotalVolume.Add(p.Quantity)
		l.Orders.PushBack(OrderEntry{
			OrderId:  p.OrderId,
			Side:     p.Type,
			Quantity: p.Quantity,
		})
		o.buyMu.Unlock()
	}
}

func (o *OrderBook) RemoveOrder(orderId string, side db.OrderSide, price decimal.Decimal) (bool, decimal.Decimal) {
	levelKey := price.String()
	if side == db.OrderSideSell {
		o.sellMu.Lock()
		defer o.sellMu.Unlock()
		l, found := o.asks[levelKey]
		if !found {
			return false, decimal.Zero
		}
		for e := l.Orders.Front(); e != nil; e = e.Next() {
			entry := e.Value.(OrderEntry)
			if entry.OrderId != orderId {
				continue
			}
			l.Orders.Remove(e)
			l.TotalVolume = l.TotalVolume.Sub(entry.Quantity)
			if l.Orders.Len() == 0 {
				delete(o.asks, levelKey)
				o.removeSellPriceFromHeap(price)
			}
			return true, entry.Quantity

		}
	} else {
		o.buyMu.Lock()
		defer o.buyMu.Unlock()
		l, found := o.bids[levelKey]
		if !found {
			return false, decimal.Zero
		}
		for e := l.Orders.Front(); e != nil; e = e.Next() {
			entry := e.Value.(OrderEntry)
			if entry.OrderId != orderId {
				continue
			}
			l.Orders.Remove(e)
			l.TotalVolume = l.TotalVolume.Sub(entry.Quantity)
			if l.Orders.Len() == 0 {
				delete(o.bids, levelKey)
				o.removeBuyPriceFromHeap(price)
			}
			return true, entry.Quantity
		}
	}
	return false, decimal.Zero
}

func (o *OrderBook) removeSellPriceFromHeap(price decimal.Decimal) {
	for i, p := range *o.sellPrices {
		if p.Equal(price) {
			heap.Remove(o.sellPrices, i)
			break
		}
	}
}

func (o *OrderBook) removeBuyPriceFromHeap(price decimal.Decimal) {
	for i, p := range *o.bidPrices {
		if p.Equal(price) {
			heap.Remove(o.bidPrices, i)
			break
		}
	}
}

func (o *OrderBook) BestAsk() PriceLevel {
	o.sellMu.Lock()
	defer o.sellMu.Unlock()
	return o.bestAskLocked()
}

func (o *OrderBook) bestAskLocked() PriceLevel {
	var bestAsk = o.sellPrices.Peek()
	result := PriceLevel{}

	if bestAsk.Valid {
		result.Valid = true
		e, ok := o.asks[bestAsk.Decimal.String()]
		if ok {
			for n := e.Orders.Front(); n != nil; n = n.Next() {
				entry, ok := n.Value.(OrderEntry)
				if ok {
					result.Orders = append(result.Orders, entry)
				}
			}
			result.Price = bestAsk.Decimal
			result.TotalVolume = e.TotalVolume
		}
	}
	return result
}

func (o *OrderBook) BestBid() PriceLevel {
	o.buyMu.Lock()
	defer o.buyMu.Unlock()
	return o.bestBidLocked()
}

func (o *OrderBook) bestBidLocked() PriceLevel {
	var bestBid = o.bidPrices.Peek()
	result := PriceLevel{}

	if bestBid.Valid {
		result.Valid = true
		e, ok := o.bids[bestBid.Decimal.String()]
		if ok {
			for n := e.Orders.Front(); n != nil; n = n.Next() {
				entry, ok := n.Value.(OrderEntry)
				if ok {
					result.Orders = append(result.Orders, entry)
				}
			}
			result.Price = bestBid.Decimal
			result.TotalVolume = e.TotalVolume
		}
	}
	return result
}

func NewBook() *OrderBook {
	sellPrices := new(sellPriceList)
	buyPrices := new(buyPriceList)
	heap.Init(sellPrices)
	heap.Init(buyPrices)
	return &OrderBook{
		bids:       make(map[string]*priceLevel),
		asks:       make(map[string]*priceLevel),
		orders:     list.New(),
		sellPrices: sellPrices,
		bidPrices:  buyPrices,
	}
}
