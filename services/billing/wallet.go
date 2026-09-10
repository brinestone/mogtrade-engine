package billing

import (
	"context"
	"errors"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/shopspring/decimal"
)

var (
	ErrWalletAlreadyExists = errors.New("a wallet already exists for the user specified")
)

func CreateUserWallet(ctx context.Context, q *db.Queries, idg contract.IdGeneratorFunc, userId string, _type db.WalletType, balance decimal.Decimal) error {
	exists, err := q.UserHasWallet(ctx, db.UserHasWalletParams{Owner: &userId, Type: _type})
	if err != nil {
		return err
	}

	if exists {
		return ErrWalletAlreadyExists
	}
	if _type == db.WalletTypeReal {
		err = q.CreateRealWallet(ctx, db.CreateRealWalletParams{
			ID:              idg(),
			Owner:           &userId,
			StartingBalance: decimal.NewNullDecimal(balance),
		})
	} else {
		err = q.CreateVirtualWallet(ctx, db.CreateVirtualWalletParams{
			ID:              idg(),
			Owner:           &userId,
			StartingBalance: decimal.NewNullDecimal(balance),
		})
	}
	if err != nil {
		return err
	}
	return q.RefreshWalletSnapshots(ctx)
}
