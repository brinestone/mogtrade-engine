package controller

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/infra/events"
	"github.com/brinestone/mogtrade/services/billing"
	"github.com/brinestone/mogtrade/web/helpers"
	eventpayloads "github.com/brinestone/mogtrade/web/payloads/events"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go-slim.dev/ioc"
)

type Wallets struct {
	repo             *db.Queries
	logger           *slog.Logger
	pool             *pgxpool.Pool
	vStartingBalance decimal.Decimal
	rStartingBalance decimal.Decimal
	idGenerator      contract.IdGeneratorFunc
}

func onUserCreated(ctx context.Context, idg contract.IdGeneratorFunc, l *slog.Logger, pool *pgxpool.Pool, vsb decimal.Decimal, rsb decimal.Decimal, e eventpayloads.UserCreatedEventArgs) error {
	l.Info("creating wallet for new user")
	timedC, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	l.Debug("opening transaction")
	tx, err := pool.Begin(timedC)
	if err != nil {
		l.Error("failed to open transaction", "err", err.Error())
		return err
	}
	defer tx.Rollback(timedC)

	repo := db.New(tx)
	err = billing.CreateUserWallet(timedC, repo, idg, e.UserId, db.WalletTypeReal, rsb)
	if err == nil {
		l.Info("wallet created successfully", "type", db.WalletTypeReal)
	} else {
		l.Warn("failed to create wallet for user", "err", err.Error())
	}

	err = billing.CreateUserWallet(timedC, repo, idg, e.UserId, db.WalletTypeVirtual, vsb)
	if err == nil {
		tx.Commit(timedC)
		l.Info("wallet created successfully", "type", db.WalletTypeVirtual)
	} else {
		l.Warn("failed to create wallet for user", "err", err.Error(), "type", db.WalletTypeVirtual)
	}

	return err
}

func (w *Wallets) handleGetBalance(c *gin.Context) {
	// TODO: stub
	user, _ := helpers.GetCurrentUser(c)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (w *Wallets) MountV1(r *gin.RouterGroup) {
	authMiddleware := helpers.ProvideAuthMiddleware()
	router := r.Group("/wallet", authMiddleware)
	router.GET("/", w.handleGetBalance)
	ioc.Invoke(context.TODO(), w.subscribeToEventsV1)
}

func (w *Wallets) subscribeToEventsV1(eb events.EventBus) {
	userCreatedCh := eb.Subscribe(EventKeyUserCreatedV1)
	go func(ch events.DataChannel, p *pgxpool.Pool, l *slog.Logger, vsb decimal.Decimal, rsb decimal.Decimal, idg contract.IdGeneratorFunc) {
		for event := range ch {
			ev, ok := event.Data.(eventpayloads.UserCreatedEventArgs)
			if !ok {
				w.logger.Warn("event payload is not of type eventpayloads.UserCreatedEventArgs")
				continue
			}
			err := onUserCreated(eb.Context(), idg, l, p, vsb, rsb, ev)
			if err != nil {
				l.Error("event handler failed", "err", err.Error(), "event", EventKeyUserCreatedV1)
			}
		}
	}(userCreatedCh, w.pool, w.logger, w.vStartingBalance, w.rStartingBalance, w.idGenerator)
}

func NewWalletsController(repo *db.Queries, logger *slog.Logger, pool *pgxpool.Pool, vStart decimal.Decimal, rStart decimal.Decimal, idg contract.IdGeneratorFunc) *Wallets {
	return &Wallets{
		repo,
		logger.With("controller", "wallet"),
		pool,
		vStart,
		rStart,
		idg,
	}
}
