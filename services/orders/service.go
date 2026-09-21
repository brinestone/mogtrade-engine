package orders

import (
	"context"
	"database/sql"
	"errors"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

type PlaceOrderParams struct {
	TracingId        string
	OrderId          string
	IdempotencyToken string
	PlacedBy         string
	Symbol           string
	ExecutionId      string
	Side             db.OrderSide
	Type             db.OrderType
	Quantity         decimal.Decimal
	LimitPrice       decimal.NullDecimal
	StopPrice        decimal.NullDecimal
}

var (
	ErrDuplicateOrder = errors.New("order already exists")
	ErrOrderNotFound  = errors.New("order was not found")
)

func CreateOrder(ctx context.Context, q *db.Queries, p PlaceOrderParams) error {
	err := q.PlaceOrder(ctx, db.PlaceOrderParams{
		ID:            p.OrderId,
		UserID:        &p.PlacedBy,
		Symbol:        p.Symbol,
		Side:          p.Side,
		OrderType:     p.Type,
		Quantity:      p.Quantity,
		LimitPrice:    p.LimitPrice,
		StopPrice:     p.StopPrice,
		ClientOrderID: &p.IdempotencyToken,
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
