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
	ErrDuplicateOrder          = errors.New("order already exists")
	ErrExecutionNotFound       = errors.New("execution not found")
	ErrDuplicateExecutionFound = errors.New("execution already exists")
	ErrOrderNotFound           = errors.New("order was not found")
	ErrInvalidOrderTransition  = errors.New("invalid order transition")
	ErrUnsupportedOrderType    = errors.New("unsupported order type detected")
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
	Id              string
	Bid             string
	Ask             string
	User            string
	TracingId       string
	MatchedQuantity decimal.Decimal
	BidExchangeRate decimal.Decimal
	AskExchangeRate decimal.Decimal
}

func CreateExecution(ctx context.Context, q *db.Queries, p CreateExecutionParams) (db.Execution, error) {
	var exec db.Execution

	exists, err := q.ExecutionExists(ctx, db.ExecutionExistsParams{
		Ask: &p.Ask,
		Bid: &p.Bid,
	})
	if err != nil {
		return exec, err
	} else if exists {
		return exec, ErrDuplicateExecutionFound
	}
	bid, err := q.FindOrderById(ctx, p.Bid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return exec, fmt.Errorf("%w: bid order not found: %s", ErrOrderNotFound, p.Bid)
		}
		return exec, err
	}
	bidWallet, _ := q.FindWalletFromWalletTransactionSrc(ctx, *bid.BidTransaction)
	ask, err := q.FindOrderById(ctx, p.Ask)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return exec, fmt.Errorf("%w: ask order not found: %s", ErrOrderNotFound, p.Ask)
		}
		return exec, err
	}

	execValue := p.BidExchangeRate.Mul(bid.LimitPrice.Decimal).Mul(p.MatchedQuantity)
	exec, err = q.CreateExecution(ctx, db.CreateExecutionParams{
		ID:                      p.Id,
		MatchQuantity:           p.MatchedQuantity,
		Symbol:                  bid.Symbol,
		BidPrice:                bid.LimitPrice.Decimal,
		BidStatus:               *bid.Status,
		BidCurrency:             bid.Currency,
		BidExchangeRateSnapshot: p.BidExchangeRate,
		ExecValue:               execValue,
		Ask:                     &p.Ask,
		AskPrice:                ask.LimitPrice.Decimal,
		AskStatus:               *ask.Status,
		AskCurrency:             ask.Currency,
		AskExchangeRateSnapshot: p.AskExchangeRate,
		TracingID:               p.TracingId,
		User:                    &p.User,
		BidWallet:               &bidWallet.ID,
		AskWallet:               ask.AskWallet,
		WalletsType:             bidWallet.Type,
		Bid:                     &p.Bid,
	})
	return exec, nil
}

func ApplyExecutionFill(ctx context.Context, q *db.Queries, execId string) error {
	exec, err := q.FindExecutionById(ctx, execId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: could not find execution: %s", ErrExecutionNotFound, execId)
		}
		return err
	}

	err = q.FillOrderById(ctx, db.FillOrderByIdParams{
		Quantity: exec.MatchQuantity,
		ID:       *exec.Ask,
	})

	if err != nil {
		return err
	}

	err = q.FillOrderById(ctx, db.FillOrderByIdParams{
		Quantity: exec.MatchQuantity,
		ID:       *exec.Bid,
	})
	if err != nil {
		return err
	}

	err = q.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
		ExecStatus: db.ExecutionStatusFilled,
		ID:         execId,
	})
	if err != nil {
		return err
	}

	err = q.UpdateExecutionOrdersStatuses(ctx, execId)

	return err
}
