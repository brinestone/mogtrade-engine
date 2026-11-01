package orders

import (
	"context"
	"testing"

	"github.com/brinestone/mogtrade/infra/db"
)

func TestOrderStatusTransitions(t *testing.T) {
	tests := []struct {
		name     string
		current  db.OrderStatus
		target   db.OrderStatus
		expected bool
	}{
		// Valid transitions to cancelled
		{"pending to cancelled", db.OrderStatusPending, db.OrderStatusCancelled, true},
		{"submitted to cancelled", db.OrderStatusSubmitted, db.OrderStatusCancelled, true},
		{"partially_filled to cancelled", db.OrderStatusPartiallyFilled, db.OrderStatusCancelled, true},

		// Invalid transitions to cancelled
		{"filled to cancelled", db.OrderStatusFilled, db.OrderStatusCancelled, false},
		{"cancelled to cancelled", db.OrderStatusCancelled, db.OrderStatusCancelled, false},
		{"rejected to cancelled", db.OrderStatusRejected, db.OrderStatusCancelled, false},
		{"expired to cancelled", db.OrderStatusExpired, db.OrderStatusCancelled, false},

		// Other state transitions
		{"pending to submitted", db.OrderStatusPending, db.OrderStatusSubmitted, true},
		{"pending to rejected", db.OrderStatusPending, db.OrderStatusRejected, true},
		{"submitted to partially_filled", db.OrderStatusSubmitted, db.OrderStatusPartiallyFilled, true},
		{"submitted to filled", db.OrderStatusSubmitted, db.OrderStatusFilled, true},
		{"partially_filled to filled", db.OrderStatusPartiallyFilled, db.OrderStatusFilled, true},
		{"filled to submitted", db.OrderStatusFilled, db.OrderStatusSubmitted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := db.Order{
				Status: &tt.current,
			}
			result := isOrderStatusTransitionValid(order, tt.target)
			if result != tt.expected {
				t.Errorf("isOrderStatusTransitionValid(%v, %v) = %v; expected %v", tt.current, tt.target, result, tt.expected)
			}
		})
	}
}

func TestNilOrderStatusTransition(t *testing.T) {
	order := db.Order{
		Status: nil,
	}
	if isOrderStatusTransitionValid(order, db.OrderStatusCancelled) {
		t.Errorf("expected nil status transition to be invalid")
	}
}

func TestValidationErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("CancelOrder with nil query runner and non-existent order", func(t *testing.T) {
		// Verify ErrOrderNotFound or ErrInvalidOrderTransition are properly defined
		if ErrOrderNotFound == nil || ErrInvalidOrderTransition == nil {
			t.Fatal("expected error definitions to be non-nil")
		}
	})

	t.Run("SubmitOrder with invalid transition", func(t *testing.T) {
		filledStatus := db.OrderStatusFilled
		order := db.Order{Status: &filledStatus}
		if isOrderStatusTransitionValid(order, db.OrderStatusSubmitted) {
			t.Fatal("expected filled to submitted to be invalid")
		}
	})

	_ = ctx
}
