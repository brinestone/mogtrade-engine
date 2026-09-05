package billing

import (
	"context"
	"errors"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/services/auth"
	"github.com/shopspring/decimal"
)

var (
	ErrWalletAlreadyExists = errors.New("a wallet already exists for the user specified")
)

func CreateUserWallet(ctx context.Context, q *db.Queries, idg auth.IdGeneratorFunc, userId string) error {
	exists, err := q.UserHasWallet(ctx, &userId)
	if err != nil {
		return err
	}

	if exists {
		return ErrWalletAlreadyExists
	}

	err = q.CreateWalletForUser(ctx, db.CreateWalletForUserParams{
		ID:              idg(),
		Owner:           &userId,
		StartingBalance: decimal.NewNullDecimal(decimal.Zero),
	})

	return err
}
