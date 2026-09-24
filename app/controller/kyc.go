package controller

import (
	"log/slog"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/web/helpers"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type KYC struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
	q      *db.Queries
	store  contract.ObjectStorage
	eb     contract.EventBus
}

func (k *KYC) handleBeginKyc(c *gin.Context) {

}

func (k *KYC) MountV1(r *gin.RouterGroup) {
	router := r.Group("/kyc")

	secured := router.Group("", helpers.ProvideAuthMiddleware(), helpers.ProvideKYCMiddleware())
	secured.POST("/", k.handleBeginKyc)
}

func NewKycController(l *slog.Logger, p *pgxpool.Pool, q *db.Queries) *KYC {
	return &KYC{
		logger: l.With("controller", "kyc"),
		pool:   p,
		q:      q,
	}
}
