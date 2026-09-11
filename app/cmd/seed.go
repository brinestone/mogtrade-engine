package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/web/helpers"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
	"github.com/shopspring/decimal"
)

func performSeeding(ctx context.Context, p *pgxpool.Pool, l *slog.Logger, idg contract.IdGeneratorFunc) {
	l.Debug("seeding database")
	tx, err := p.Begin(ctx)
	if err != nil {
		l.Error("could not open database transaction", "err", err.Error())
		panic(err)
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)
	if err = createSystemUser(ctx, q, l.With("seeder", "system-user"), idg); err != nil {
		panic(err)
	}
	l.Info("database seeded successfully")
	tx.Commit(ctx)
}

func createSystemUser(ctx context.Context, q *db.Queries, l *slog.Logger, idg contract.IdGeneratorFunc) error {
	l.Debug("creating system user account")
	userId := os.Getenv("SYSTEM_USER_ID")
	if len(userId) == 0 {
		l.Warn("skipping seeding of user account, SYSTEM_USER_ID environment variable not defined")
		return nil
	}
	exists, err := q.UserExistsWithId(ctx, userId)
	if err != nil {
		return err
	}
	if exists {
		l.Warn("system user account already exists, skipping")
		return nil
	}
	l.Debug("seeding system user account")
	if _, err := q.CreateUser(ctx, db.CreateUserParams{
		ID:    userId,
		Name:  os.Getenv("SYSTEM_NAME"),
		Email: os.Getenv("SYSTEM_EMAIL"),
	}); err != nil {
		return err
	}
	l.Debug("verifying system user email")
	if err := q.VerifyUserEmail(ctx, userId); err != nil {
		return err
	}

	l.Debug("seeding system user credential account")
	if _, err := q.CreateCredentialAccount(ctx, db.CreateCredentialAccountParams{
		ID:        idg(),
		AccountID: os.Getenv("SYSTEM_EMAIl"),
		UserID:    userId,
		Password:  helpers.Ptr(os.Getenv("SYSTEM_PWD")),
	}); err != nil {
		return err
	}
	l.Info("seeded system user")
	return createSystemWallets(ctx, q, l, userId)
}
func createSystemWallets(ctx context.Context, q *db.Queries, l *slog.Logger, uid string) error {
	l.Debug("creating system wallets")
	realWalletId := os.Getenv("SYSTEM_REAL_WALLET_ID")
	virtualWalletId := os.Getenv("SYSTEM_VIRTUAL_WALLET_ID")

	if len(virtualWalletId) != 26 || len(realWalletId) != 26 {
		l.Warn("skipping wallet seeding, SYSTEM_REAL_WALLET_ID and SYSTEM_VIRTUAL_WALLET_ID are not properly define")
		return nil
	}

	if _, err := ulid.Parse(realWalletId); err != nil {
		l.Warn("skipping wallet seeding, SYSTEM_REAL_WALLET_ID is not a valid ULID value")
		return nil
	}
	if _, err := ulid.Parse(virtualWalletId); err != nil {
		l.Warn("skipping wallet seeding, SYSTEM_VIRTUAL_WALLET_ID is not a valid ULID value")
		return nil
	}
	virtualStartingBalance, err := decimal.NewFromString(os.Getenv("SYSTEM_USER_VIRTUAL_WALLET_BALANCE"))
	if err != nil {
		l.Warn("skipping wallet seeding, SYSTEM_USER_VIRTUAL_WALLET_BALANCE is not a valid number")
	}

	realStartingBalance, err := decimal.NewFromString(os.Getenv("SYSTEM_USER_REAL_WALLET_BALANCE"))
	if err != nil {
		l.Warn("skipping wallet seeding, SYSTEM_USER_REAL_WALLET_BALANCE is not a valid number")
		return nil
	}

	exists, err := q.UserHasWallet(ctx, db.UserHasWalletParams{Owner: &uid, Type: db.WalletTypeReal})
	if err != nil {
		return err
	}
	if exists {
		l.Warn("system user real wallet already exists, skipping seeding")
		return nil
	}

	exists, err = q.UserHasWallet(ctx, db.UserHasWalletParams{Owner: &uid, Type: db.WalletTypeVirtual})
	if err != nil {
		return err
	}
	if exists {
		l.Warn("system user virtual wallet already exists, skipping seeding")
		return nil
	}
	if err := q.CreateRealWallet(ctx, db.CreateRealWalletParams{
		ID:              realWalletId,
		Owner:           &uid,
		StartingBalance: decimal.NewNullDecimal(realStartingBalance),
	}); err != nil {
		return err
	}

	if err := q.CreateVirtualWallet(ctx, db.CreateVirtualWalletParams{
		ID:              virtualWalletId,
		Owner:           &uid,
		StartingBalance: decimal.NewNullDecimal(virtualStartingBalance),
	}); err != nil {
		return err
	}
	q.RefreshWalletSnapshots(ctx)

	l.Info("seeded system wallets")
	return nil
}
