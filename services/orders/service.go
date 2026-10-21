package orders

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

var (
	ErrDuplicateOrder         = errors.New("order already exists")
	ErrOrderNotFound          = errors.New("order was not found")
	ErrInvalidOrderTransition = errors.New("invalid order transition")
	ErrUnsupportedOrderType   = errors.New("unsupported order type detected")
)

type SubmitOrderParams struct {
	OwnerId string
	OrderId string
}

func SubmitOrder(ctx context.Context, q *db.Queries, params SubmitOrderParams) error {
	order, err := q.FindOwnedOrder(ctx, db.FindOwnedOrderParams{
		UserID: &params.OwnerId,
		ID:     params.OrderId,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOrderNotFound
		}
		return err
	}

	if !isTransitionValid(order, db.OrderStatusSubmitted) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidOrderTransition, *order.Status, db.OrderStatusSubmitted)
	}
	return q.MarkOrderSubmitted(ctx, db.MarkOrderSubmittedParams{UserID: &params.OwnerId, ID: params.OrderId})
}

type RejectOrderParams struct {
	OwnerId string
	OrderId string
}

func RejectOrder(ctx context.Context, q *db.Queries, params RejectOrderParams) error {
	order, err := q.FindOwnedOrder(ctx, db.FindOwnedOrderParams{
		UserID: &params.OwnerId,
		ID:     params.OrderId,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOrderNotFound
		}
		return err
	}

	if !isTransitionValid(order, db.OrderStatusRejected) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidOrderTransition, *order.Status, db.OrderStatusRejected)
	}

	return q.MarkOrderRejected(ctx, db.MarkOrderRejectedParams{UserID: &params.OwnerId, ID: params.OrderId})
}

type CancelOrderParams struct {
	OwnerId string
	OrderId string
}

func CancelOrder(ctx context.Context, q *db.Queries, params CancelOrderParams) (db.Order, error) {
	order, err := q.FindOwnedOrder(ctx, db.FindOwnedOrderParams{
		UserID: &params.OwnerId,
		ID:     params.OrderId,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return order, ErrOrderNotFound
		}
		return order, err
	}

	if !isTransitionValid(order, db.OrderStatusCancelled) {
		return order, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidOrderTransition, *order.Status, db.OrderStatusCancelled)
	}

	err = q.MarkOrderCancelled(ctx, db.MarkOrderCancelledParams{UserID: &params.OwnerId, ID: params.OrderId})

	return order, err
}

func isTransitionValid(o db.Order, status db.OrderStatus) bool {
	if o.Status == nil {
		return false
	}
	switch *o.Status {
	case db.OrderStatusPending:
		return slices.Contains([]db.OrderStatus{
			db.OrderStatusCancelled,
			db.OrderStatusRejected,
			db.OrderStatusSubmitted,
			db.OrderStatusExpired,
		}, status)
	case db.OrderStatusSubmitted:
		return slices.Contains([]db.OrderStatus{
			db.OrderStatusPartiallyFilled,
			db.OrderStatusFilled,
			db.OrderStatusCancelled,
			db.OrderStatusRejected,
			db.OrderStatusExpired,
		}, status)
	case db.OrderStatusPartiallyFilled:
		return slices.Contains([]db.OrderStatus{
			db.OrderStatusFilled,
			db.OrderStatusCancelled,
		}, status)
	default:
		return false
	}
}

type PlaceOrderParams struct {
	TracingId            string
	OrderId              string
	IdempotencyToken     string
	PlacedBy             string
	Symbol               string
	WalletId             string
	SystemWalletId       string
	WalletTransactionId  string
	Currency             string
	Side                 db.OrderSide
	Type                 db.OrderType
	Quantity             decimal.Decimal
	ExchangeRateSnapshot decimal.Decimal
	LimitPrice           decimal.NullDecimal
	StopPrice            decimal.NullDecimal
}

func PlaceOrder(ctx context.Context, q *db.Queries, p PlaceOrderParams) error {
	if p.Type != db.OrderTypeLimit {
		return fmt.Errorf("%w: non-limit orders are not yet supported", ErrUnsupportedOrderType)
	}
	err := q.PlaceOrder(ctx, db.PlaceOrderParams{
		ID:            p.OrderId,
		UserID:        &p.PlacedBy,
		TracingID:     p.TracingId,
		Symbol:        p.Symbol,
		Side:          p.Side,
		OrderType:     p.Type,
		Quantity:      p.Quantity,
		LimitPrice:    p.LimitPrice,
		StopPrice:     p.StopPrice,
		Fee:           p.LimitPrice.Decimal,
		Currency:      p.Currency,
		ExchangeRate:  p.ExchangeRateSnapshot,
		ClientOrderID: &p.IdempotencyToken,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "orders_user_id_client_order_id_uidx" {
			return ErrDuplicateOrder
		}
		return err
	}

	return err
}

type CreateExecutionParams struct {
	OrderId     string
	Id          string
	FeeCurrency string
	TracingId   string
	FeeRate     decimal.Decimal
	FeeAmount   decimal.Decimal
	Price       decimal.Decimal
	Quantity    decimal.Decimal
}

func CreateExecution(ctx context.Context, q *db.Queries, p CreateExecutionParams) error {
	order, err := q.FindOrderById(ctx, p.OrderId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOrderNotFound
		}
		return err
	}
	err = q.CreateExecution(ctx, db.CreateExecutionParams{
		ID:          p.Id,
		User:        order.UserID,
		Order:       p.OrderId,
		Price:       p.Price,
		Symbol:      order.Symbol,
		FeeRate:     p.FeeRate,
		Quantity:    p.Quantity,
		FeeAmount:   p.FeeAmount,
		OrderSide:   order.Side,
		OrderType:   order.OrderType,
		TracingID:   p.TracingId,
		ExecStatus:  db.ExecutionStatusFilled,
		OrderStatus: *order.Status,
		FeeCurrency: p.FeeCurrency,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "orders_user_id_client_order_id_uidx" {
			return ErrDuplicateOrder
		}
		return err
	}
	return nil
}
