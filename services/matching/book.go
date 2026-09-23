package matching

import (
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
	if l.Len() == 0 {
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
	if l.Len() == 0 {
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
	var level *priceLevel
	if p.Type == db.OrderSideSell {
		l, found := o.asks[p.Price.String()]
		if !found {
			o.sellMu.Lock()
			l = &priceLevel{
				Orders:      list.New(),
				TotalVolume: decimal.Zero,
			}
			o.asks[p.Price.String()] = level
			o.sellMu.Unlock()
			o.sellPrices.Push(p.Price)
		}
		level = l
	} else {
		l, found := o.bids[p.Price.String()]
		if !found {
			o.buyMu.Lock()
			l = &priceLevel{
				Orders:      list.New(),
				TotalVolume: decimal.Zero,
			}
			o.bids[p.Price.String()] = level
			o.buyMu.Unlock()
			o.bidPrices.Push(p.Price)
		}
		level = l
	}
	level.TotalVolume.Add(p.Quantity)
	level.Orders.PushBack(OrderEntry{
		OrderId:  p.OrderId,
		Quantity: p.Quantity,
	})
}

func (o *OrderBook) BestAsk() PriceLevel {
	var bestAsk = o.sellPrices.Peek()
	result := PriceLevel{}

	if bestAsk.Valid {
		result.Valid = true
		e := o.asks[bestAsk.Decimal.String()]
		for n := e.Orders.Front(); n != nil; n = n.Next() {
			entry, ok := n.Value.(OrderEntry)
			if ok {
				result.Orders = append(result.Orders, entry)
			}
		}
		result.Price = bestAsk.Decimal
		result.TotalVolume = e.TotalVolume
	}
	return result
}

func (o *OrderBook) BestBid() PriceLevel {
	var bestBid = o.bidPrices.Peek()
	result := PriceLevel{}

	if bestBid.Valid {
		result.Valid = true
		e := o.bids[bestBid.Decimal.String()]
		for n := e.Orders.Front(); n != nil; n = n.Next() {
			entry, ok := n.Value.(OrderEntry)
			if ok {
				result.Orders = append(result.Orders, entry)
			}
		}
		result.Price = bestBid.Decimal
		result.TotalVolume = e.TotalVolume
	}
	return result
}

func (o *OrderBook) findMatches() {

}

func NewBook() *OrderBook {
	sellPrices := new(make(sellPriceList, 0))
	buyPrices := new(make(buyPriceList, 0))
	return &OrderBook{
		bids:       make(map[string]*priceLevel),
		asks:       make(map[string]*priceLevel),
		orders:     list.New(),
		sellPrices: sellPrices,
		bidPrices:  buyPrices,
	}
}
