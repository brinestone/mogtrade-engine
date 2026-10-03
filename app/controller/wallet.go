package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/services/auth"
	"github.com/brinestone/mogtrade/services/billing"
	"github.com/brinestone/mogtrade/web/helpers"
	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	PayloadWalletNotFound = gin.H{"error": "wallet not found"}
)

type Wallets struct {
	repo             *db.Queries
	logger           *slog.Logger
	pool             *pgxpool.Pool
	vStartingBalance decimal.Decimal
	rStartingBalance decimal.Decimal
	idGenerator      contract.IdFactory
}

func (w *Wallets) handleGetBalance(c *gin.Context) {
	var req httppayloads.GetWalletSnapshotRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": strings.Split(err.Error(), "\n")})
		return
	}
	snapshot, err := billing.GetCurrentWalletSnapshot(c.Request.Context(), w.repo, billing.GetWalletSnapshotParams{
		Type:    req.Type,
		OwnerId: helpers.GetCurrentUserId(c),
	})
	if err != nil {
		if errors.Is(err, billing.ErrWalletNotFound) {
			w.logger.Warn("user wallet not found, checking whether user exists", "uid", helpers.GetCurrentUserId(c))
			userExists, err := auth.UserExistsWithId(c.Request.Context(), w.repo, helpers.GetCurrentUserId(c))
			if err != nil {
				w.logger.Error("could not check for user existing", "err", err.Error(), "uid", helpers.GetCurrentUserId(c))
				helpers.InternalServerError(c)
				return
			}
			if userExists {
				w.logger.Info("user exists, creating their wallets", "type", req.Type, "uid", helpers.GetCurrentUserId(c))
				err = w.createUserWallets(c)
				if err != nil {
					helpers.InternalServerError(c)
					return
				} else {
					snapshot, err = billing.GetCurrentWalletSnapshot(c.Request.Context(), w.repo, billing.GetWalletSnapshotParams{
						Type:    req.Type,
						OwnerId: helpers.GetCurrentUserId(c),
					})
					if err != nil {
						helpers.InternalServerError(c)
						return
					}
				}
			}
			w.logger.Warn("user wallet not found", "uid", helpers.GetCurrentUserId(c), "type", req.Type)
			c.AbortWithStatusJSON(http.StatusNotFound, PayloadWalletNotFound)
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}
	c.JSON(http.StatusOK, httppayloads.WalletSnapshotPayload(snapshot))
}

func (w *Wallets) createUserWallets(c *gin.Context) error {
	w.logger.Debug("opening database transaction")
	tx, err := w.pool.Begin(c.Request.Context())
	if err != nil {
		w.logger.Error("could not open database transaction", "err", err.Error())
		return err
	}
	defer tx.Rollback(c.Request.Context())

	repo := w.repo.WithTx(tx)
	realId := w.idGenerator()
	virtualId := w.idGenerator()

	err = billing.CreateUserWallet(c.Request.Context(), repo, billing.CreateWalletParams{
		OwnerId: helpers.GetCurrentUserId(c),
		Type:    db.WalletTypeReal,
		Id:      realId,
	})

	if err != nil {
		if errors.Is(err, billing.ErrWalletAlreadyExists) {
			w.logger.Warn("wallet already exists", "type", db.WalletTypeReal, "uid", helpers.GetCurrentUserId(c))
		} else {
			return err
		}
	}

	w.logger.Info("created user wallet", "uid", helpers.GetCurrentUserId(c), "type", db.WalletTypeReal)

	err = billing.CreateUserWallet(c.Request.Context(), repo, billing.CreateWalletParams{
		OwnerId: helpers.GetCurrentUserId(c),
		Type:    db.WalletTypeVirtual,
		Id:      virtualId,
	})

	if err != nil {
		if errors.Is(err, billing.ErrWalletAlreadyExists) {
			w.logger.Warn("wallet already exists", "type", db.WalletTypeVirtual, "uid", helpers.GetCurrentUserId(c))
		} else {
			return err
		}
	}

	w.logger.Info("created user wallet", "uid", helpers.GetCurrentUserId(c), "type", db.WalletTypeVirtual)
	txId := w.idGenerator()
	err = billing.RecordWalletTransaction(c.Request.Context(), repo, billing.RecordWalletTransactionParams{
		Id:               txId,
		Dest:             &virtualId,
		Intent:           "Initial deposit",
		IdempotencyToken: w.idGenerator(),
		DoneBy:           helpers.GetCurrentUserId(c),
		TracingId:        w.idGenerator(),
		Currency:         "USD", // TODO: Get from user prefs
		ExchangeRate:     decimal.NewFromInt(1),
		Value:            w.vStartingBalance,
		Src:              nil,
	})
	if err != nil {
		w.logger.Error("could not create wallet transaction", "wallet-type", db.WalletTypeVirtual, "uid", helpers.GetCurrentUserId(c), "err", err.Error())
		return err
	}

	err = billing.UpdateWalletTransactionStatus(c.Request.Context(), repo, billing.UpdateTransactionStatusParams{TransactionId: txId, Status: db.TransactionStatusProcessing})
	if err != nil {
		w.logger.Error("could not update wallet transaction", "id", txId, "uid", helpers.GetCurrentUserId(c))
		return err
	}
	err = billing.UpdateWalletTransactionStatus(c.Request.Context(), repo, billing.UpdateTransactionStatusParams{TransactionId: txId, Status: db.TransactionStatusCompleted})
	if err != nil {
		w.logger.Error("could not update wallet transaction", "id", txId, "uid", helpers.GetCurrentUserId(c))
		return err
	}
	w.logger.Info("wallet created successfully", "type", db.WalletTypeVirtual)
	tx.Commit(c.Request.Context())
	return nil
}

func (w *Wallets) MountV1(r *gin.RouterGroup) {
	jwtMiddleware := helpers.ProvideJWTMiddleware()

	secured := r.Group("/wallet", jwtMiddleware())
	secured.GET("/status/:type", w.handleGetBalance)
}

func NewWalletsController(repo *db.Queries, logger *slog.Logger, pool *pgxpool.Pool, vStart decimal.Decimal, rStart decimal.Decimal, idg contract.IdFactory) *Wallets {
	return &Wallets{
		repo,
		logger.With("controller", "wallet"),
		pool,
		vStart,
		rStart,
		idg,
	}
}
