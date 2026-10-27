package matching

import (
	"log/slog"
	"testing"
	"time"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/infra/events"
	"github.com/shopspring/decimal"
)

func NewTestLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestPlaceMatchOrder_BasicMatch(t *testing.T) {
	logger := NewTestLogger(t)
	eb := events.UseInMemoryEventBus()
	eng := NewMatchingEngine(logger, eb)

	matches := eb.Subscribe(t.Context(), EventKeyOrderMatched)

	sellOrder := PlaceMatchOrderParams{
		OrderId:  "sell-1",
		Symbol:   "BTCUSD",
		Type:     db.OrderSideSell,
		Quantity: decimal.NewFromFloat(5.0),
		Price:    decimal.NewFromFloat(100.0),
	}
	eng.PlaceMatchOrder(sellOrder)

	buyOrder := PlaceMatchOrderParams{
		OrderId:  "buy-1",
		Symbol:   "BTCUSD",
		Type:     db.OrderSideBuy,
		Quantity: decimal.NewFromFloat(10.0),
		Price:    decimal.NewFromFloat(105.0),
	}
	eng.PlaceMatchOrder(buyOrder)

	select {
	case _ev := <-matches:
		ev, ok := _ev.Data.(OrderMatchedEventArgs)
		if !ok {
			t.Fatalf("expected OrderMatched event data, got %T", _ev.Data)
		}
		if ev.Bid != "buy-1" {
			t.Errorf("expected BuyOrder='buy-1', got '%s'", ev.Bid)
		}
		if ev.Ask != "sell-1" {
			t.Errorf("expected SellOrder='sell-1', got '%s'", ev.Ask)
		}
		if ev.Symbol != "BTCUSD" {
			t.Errorf("expected Symbol='BTCUSD', got '%s'", ev.Symbol)
		}
		if !ev.MatchQuantity.Equal(decimal.NewFromFloat(5.0)) {
			t.Errorf("expected MatchQuantity=5.0, got '%s'", ev.MatchQuantity)
		}
	case <-time.After(time.Second * 10):
		t.Fatal("timed out waiting for match event")
	}
}

func TestPlaceMatchOrder_NoMatch_DifferentSide(t *testing.T) {
	logger := NewTestLogger(t)
	eb := events.UseInMemoryEventBus()
	eng := NewMatchingEngine(logger, eb)

	matches := eb.Subscribe(t.Context(), EventKeyOrderMatched)

	sellOrder := PlaceMatchOrderParams{
		OrderId:  "sell-1",
		Symbol:   "ETHUSD",
		Type:     db.OrderSideSell,
		Quantity: decimal.NewFromFloat(3.0),
		Price:    decimal.NewFromFloat(200.0),
	}
	eng.PlaceMatchOrder(sellOrder)

	buyOrder := PlaceMatchOrderParams{
		OrderId:  "buy-1",
		Symbol:   "ETHUSD",
		Type:     db.OrderSideBuy,
		Quantity: decimal.NewFromFloat(1.0),
		Price:    decimal.NewFromFloat(150.0),
	}
	eng.PlaceMatchOrder(buyOrder)

	select {
	case ev := <-matches:
		t.Fatalf("received unexpected match event: %+v", ev)
	case <-time.After(100 * time.Millisecond):
		// Expected: no event because bid (150) < ask (200)
	}
}

func TestPlaceMatchOrder_PartialFill(t *testing.T) {
	logger := NewTestLogger(t)
	eb := events.UseInMemoryEventBus()
	eng := NewMatchingEngine(logger, eb)

	matches := eb.Subscribe(t.Context(), EventKeyOrderMatched)

	sellOrder := PlaceMatchOrderParams{
		OrderId:  "sell-1",
		Symbol:   "SOLUSD",
		Type:     db.OrderSideSell,
		Quantity: decimal.NewFromFloat(10.0),
		Price:    decimal.NewFromFloat(100.0),
	}
	eng.PlaceMatchOrder(sellOrder)

	buyOrder := PlaceMatchOrderParams{
		OrderId:  "buy-1",
		Symbol:   "SOLUSD",
		Type:     db.OrderSideBuy,
		Quantity: decimal.NewFromFloat(4.0),
		Price:    decimal.NewFromFloat(105.0),
	}
	eng.PlaceMatchOrder(buyOrder)

	select {
	case _ev := <-matches:
		ev, ok := _ev.Data.(OrderMatchedEventArgs)
		if !ok {
			t.Fatalf("expected OrderMatched, got %T", _ev.Data)
		}
		if ev.Bid != "buy-1" {
			t.Errorf("expected BuyOrder='buy-1', got '%s'", ev.Bid)
		}
		if ev.Ask != "sell-1" {
			t.Errorf("expected SellOrder='sell-1', got '%s'", ev.Ask)
		}
		if !ev.MatchQuantity.Equal(decimal.NewFromFloat(4.0)) {
			t.Errorf("expected MatchQuantity=4.0, got '%s'", ev.MatchQuantity)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for match event")
	}
}

func TestCancelMatchOrder(t *testing.T) {
	logger := NewTestLogger(t)
	eb := events.UseInMemoryEventBus()
	eng := NewMatchingEngine(logger, eb)

	orderPrice := decimal.NewFromFloat(100.0)
	orderQty := decimal.NewFromFloat(5.0)

	eng.PlaceMatchOrder(PlaceMatchOrderParams{
		OrderId:  "order-to-cancel",
		Symbol:   "BTCUSD",
		Type:     db.OrderSideSell,
		Quantity: orderQty,
		Price:    orderPrice,
	})

	// Cancel the order before any buy comes in
	removed, remQty := eng.CancelMatchOrder("order-to-cancel", "BTCUSD", db.OrderSideSell, orderPrice)
	if !removed {
		t.Fatal("expected order to be successfully removed from order book")
	}
	if !remQty.Equal(orderQty) {
		t.Fatalf("expected remaining quantity %s, got %s", orderQty, remQty)
	}

	// Try removing again - should return false
	removed2, _ := eng.CancelMatchOrder("order-to-cancel", "BTCUSD", db.OrderSideSell, orderPrice)
	if removed2 {
		t.Fatal("expected second cancel to return false")
	}

	// Now place a crossing buy order - should NOT match since sell was cancelled
	matches := eb.Subscribe(t.Context(), EventKeyOrderMatched)
	eng.PlaceMatchOrder(PlaceMatchOrderParams{
		OrderId:  "buy-2",
		Symbol:   "BTCUSD",
		Type:     db.OrderSideBuy,
		Quantity: decimal.NewFromFloat(5.0),
		Price:    decimal.NewFromFloat(105.0),
	})

	select {
	case ev := <-matches:
		t.Fatalf("did not expect match after cancellation, but received: %+v", ev)
	case <-time.After(100 * time.Millisecond):
		// Success: no match occurred
	}
}
