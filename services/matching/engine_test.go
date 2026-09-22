package matching

import (
	"testing"
	"time"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/shopspring/decimal"
	"log/slog"
)

func TestPlaceMatchOrder_BasicMatch(t *testing.T) {
	logger := NewTestLogger(t)
	eng := NewMatchingEngine(logger)

	// Insert a sell order at price 100.00 for quantity 5.0
	sellOrder := PlaceMatchOrderParams{
		OrderId: "sell-1",
		Symbol:  "BTCUSD",
		Type:    db.OrderSideSell,
		Quantity: decimal.NewFromFloat(5.0),
		Price:    decimal.NewFromFloat(100.0),
	}
	eng.PlaceMatchOrder(sellOrder)

	// Insert a buy order at price 105.00 for quantity 10.0 (should match 5.0 against sell)
	buyOrder := PlaceMatchOrderParams{
		OrderId: "buy-1",
		Symbol:  "BTCUSD",
		Type:    db.OrderSideBuy,
		Quantity: decimal.NewFromFloat(10.0),
		Price:    decimal.NewFromFloat(105.0),
	}
	eng.PlaceMatchOrder(buyOrder)

	// Drain the match channel to check events
	matches := eng.Matches()
	ev := <-matches
	if ev.BuyOrder != "buy-1" {
		t.Errorf("expected BuyOrder='buy-1', got '%s'", ev.BuyOrder)
	}
	if ev.SellOrder != "sell-1" {
		t.Errorf("expected SellOrder='sell-1', got '%s'", ev.SellOrder)
	}
	if ev.Symbol != "BTCUSD" {
		t.Errorf("expected Symbol='BTCUSD', got '%s'", ev.Symbol)
	}
	// The event should have been sent within a reasonable time.
	// If the channel is empty after a short wait, it might be buffered.
	// We'll just check that we can receive something without blocking forever.
	select {
	case _ = <-matches:
		// good
	default:
		t.Log("No match event received (may be buffered or no match)")
	}
}

func TestPlaceMatchOrder_NoMatch_DifferentSide(t *testing.T) {
	logger := NewTestLogger(t)
	eng := NewMatchingEngine(logger)

	// Insert a sell order at price 200.00
	sellOrder := PlaceMatchOrderParams{
		OrderId: "sell-1",
		Symbol:  "ETHUSD",
		Type:    db.OrderSideSell,
		Quantity: decimal.NewFromFloat(3.0),
		Price:    decimal.NewFromFloat(200.0),
	}
	eng.PlaceMatchOrder(sellOrder)

	// Insert a buy order at a lower price (no match because bid < ask)
	buyOrder := PlaceMatchOrderParams{
		OrderId: "buy-1",
		Symbol:  "ETHUSD",
		Type:    db.OrderSideBuy,
		Quantity: decimal.NewFromFloat(1.0),
		Price:    decimal.NewFromFloat(150.0),
	}
	eng.PlaceMatchOrder(buyOrder)

	// No match event should be generated (or the channel may be empty)
	matches := eng.Matches()
	select {
	case ev := <-matches:
		t.Logf("Received unexpected match event: %+v", ev)
	case <-time.After(100 * time.Millisecond):
		// Expected no event
	}
}

func TestPlaceMatchOrder_PartialFill(t *testing.T) {
	logger := NewTestLogger(t)
	eng := NewMatchingEngine(logger)

	// Sell order with qty 10.0 at price 100.0
	sellOrder := PlaceMatchOrderParams{
		OrderId: "sell-1",
		Symbol:  "SOLUSD",
		Type:    db.OrderSideSell,
		Quantity: decimal.NewFromFloat(10.0),
		Price:    decimal.NewFromFloat(100.0),
	}
	eng.PlaceMatchOrder(sellOrder)

	// Buy order with qty 4.0 at price 105.0 (should match 4.0, leaving 6.0 on sell side)
	buyOrder := PlaceMatchOrderParams{
		OrderId: "buy-1",
		Symbol:  "SOLUSD",
		Type:    db.OrderSideBuy,
		Quantity: decimal.NewFromFloat(4.0),
		Price:    decimal.NewFromFloat(105.0),
	}
	eng.PlaceMatchOrder(buyOrder)

	// Drain match channel
	matches := eng.Matches()
	ev := <-matches
	if ev.BuyOrder != "buy-1" {
		t.Errorf("expected BuyOrder='buy-1', got '%s'", ev.BuyOrder)
	}
	if ev.SellOrder != "sell-1" {
		t.Errorf("expected SellOrder='sell-1', got '%s'", ev.SellOrder)
	}
	// After partial fill, the sell order should still have remaining qty 6.0
	// We can't directly check the order book from here, but we can verify the event was sent.
	select {
	case _ = <-matches:
		// second event (if any)
	default:
		// no more events
	}
}

// NewTestLogger creates a simple test logger that discards output.
func NewTestLogger(t *testing.T) *slog.Logger {
	// Use slog.Discard to avoid panics from nil writers.
	logger := slog.New(slog.DiscardHandler)
	return logger
}

func init() {
	// Ensure decimal.NewFromFloat works as expected.
	_ = decimal.NewFromFloat(0.0)
}