package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/shopspring/decimal"
)

var (
	ErrWalletAlreadyExists          = errors.New("a wallet already exists for the user specified")
	ErrDuplicateIdempotencyKey      = errors.New("idempotency token already exists")
	ErrNoWalletTransaction          = errors.New("wallet transaction was not found")
	ErrInvalidTransactionTransition = errors.New("invalid wallet transaction transition")
)

type RecordWalletTransactionParams struct {
	Id string
	// WalletType       db.WalletType
	Src              *string
	Dest             *string
	Intent           string
	ExtraData        map[string]any
	IdempotencyToken string
	DoneBy           string
	TracingId        string
	Currency         string
	ExchangeRate     decimal.Decimal
	Value            decimal.Decimal
}

type CreateWalletParams struct {
	OwnerId string
	Type    db.WalletType
	Id      string
}

type UpdateTransactionStatusParams struct {
	TransactionId string
	Status        db.TransactionStatus
}

func UpdateWalletTransactionStatus(ctx context.Context, q *db.Queries, params UpdateTransactionStatusParams) error {
	var err error
	tx, err := q.FindWalletTransactionById(ctx, params.TransactionId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoWalletTransaction
		}
		return err
	}

	if !isTransitionValid(tx, params.Status) {
		return ErrInvalidTransactionTransition
	}

	err = q.UpdateTransactionStatusById(ctx, db.UpdateTransactionStatusByIdParams{
		Status: params.Status,
		ID:     params.TransactionId,
	})

	err = q.CreateTransactionLedgerEntry(ctx, db.CreateTransactionLedgerEntryParams{
		Wallet:      tx.Dest,
		Transaction: tx.ID,
		Notes:       fmt.Sprintf("status changed from \"%s\" to \"%s\"", tx.Status, params.Status),
	})
	if err != nil {
		return err
	}
	err = q.CreateTransactionLedgerEntry(ctx, db.CreateTransactionLedgerEntryParams{
		Wallet:      tx.Src,
		Transaction: tx.ID,
		Notes:       fmt.Sprintf("status changed from \"%s\" to \"%s\"", tx.Status, params.Status),
	})
	if err != nil {
		return err
	}
	err = q.RefreshWalletSnapshots(ctx)

	return err
}

func CreditUserWallet(ctx context.Context, q *db.Queries, req RecordWalletTransactionParams) error {
	exists, err := q.IdempotencyKeyExists(ctx, req.IdempotencyToken)
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateIdempotencyKey
	}

	extra, err := json.Marshal(req.ExtraData)
	if err != nil {
		return err
	}
	err = q.RecordWalletTransaction(ctx, db.RecordWalletTransactionParams{
		ID:               req.Id,
		Src:              req.Src,
		Dest:             req.Dest,
		Intent:           &req.Intent,
		ExtraData:        extra,
		Value:            req.Value,
		IdempotencyToken: req.IdempotencyToken,
		DoneBy:           &req.DoneBy,
		TracingID:        req.TracingId,
		Currency:         &req.Currency,
		ExchangeRate:     req.ExchangeRate,
	})
	if err != nil {
		return err
	}

	err = q.CreateTransactionLedgerEntry(ctx, db.CreateTransactionLedgerEntryParams{
		Wallet:      req.Src,
		Transaction: req.Id,
		Notes:       fmt.Sprintf("-%s%s", req.Currency, req.Value.Div(req.ExchangeRate).String()),
	})
	if err != nil {
		return err
	}
	err = q.CreateTransactionLedgerEntry(ctx, db.CreateTransactionLedgerEntryParams{
		Wallet:      req.Dest,
		Transaction: req.Id,
		Notes:       fmt.Sprintf("+%s%s", req.Currency, req.Value.Div(req.ExchangeRate).String()),
	})
	if err != nil {
		return err
	}
	err = q.RefreshWalletSnapshots(ctx)

	return err
}

func CreateUserWallet(ctx context.Context, q *db.Queries, req CreateWalletParams) error {
	exists, err := q.UserHasWallet(ctx, db.UserHasWalletParams{Owner: &req.OwnerId, Type: req.Type})
	if err != nil {
		return err
	}

	if exists {
		return ErrWalletAlreadyExists
	}
	if req.Type == db.WalletTypeReal {
		err = q.CreateRealWallet(ctx, db.CreateRealWalletParams{
			ID:              req.Id,
			Owner:           &req.OwnerId,
			StartingBalance: decimal.NewNullDecimal(decimal.Zero),
		})
	} else {
		err = q.CreateVirtualWallet(ctx, db.CreateVirtualWalletParams{
			ID:              req.Id,
			Owner:           &req.OwnerId,
			StartingBalance: decimal.NewNullDecimal(decimal.Zero),
		})
	}
	if err != nil {
		return err
	}
	return q.RefreshWalletSnapshots(ctx)
}
func isTransitionValid(tx db.WalletTransaction, status db.TransactionStatus) bool {
	switch tx.Status {
	case db.TransactionStatusProcessing:
		return status == db.TransactionStatusFailed || status == db.TransactionStatusCancelled || status == db.TransactionStatusCompleted
	case db.TransactionStatusFailed:
		return status == db.TransactionStatusProcessing
	}
	return false
}
