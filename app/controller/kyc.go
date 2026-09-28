package controller

import (
	"io"
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
	frontFile, header, err := c.Request.FormFile("front")
	if err != nil {
		panic(err)
	}
	front, err := io.ReadAll(frontFile)
	println(front)
	println(header.Filename)
}

func (k *KYC) MountV1(r *gin.RouterGroup) {
	router := r.Group("/kyc")

	kycMiddleware := helpers.ProvideKYCMiddleware()
	secured := router.Group("", helpers.ProvideAuthMiddleware(), kycMiddleware(contract.KYCPending))
	secured.POST("/", k.handleBeginKyc)
}

func NewKycController(l *slog.Logger, p *pgxpool.Pool, q *db.Queries) *KYC {
	return &KYC{
		logger: l.With("controller", "kyc"),
		pool:   p,
		q:      q,
	}
}
