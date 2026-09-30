package controller

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
	UploadLimits
}

type UploadLimits struct {
	maxUpload int64
}

type beginKycPayload struct {
	Dob           time.Time       `json:"dateOfBirth" form:"dateOfBirth" binding:"required" time_format:"2006-01-02"`
	City          string          `json:"city" form:"city" binding:"required"`
	State         string          `json:"state" form:"state" binding:"required"`
	Gender        contract.Gender `json:"gender" form:"gender" binding:"required"`
	Selfie        string          `json:"selfie" form:"selfie" binding:"required,url"`
	Country       string          `json:"country" form:"country" binding:"required"`
	ClientId      string          `json:"clientId" form:"clientId" binding:"required"`
	LegalNames    string          `json:"legalNames" form:"legalNames" binding:"required"`
	DocumentBack  string          `json:"docBack" form:"docBack" binding:"required,url"`
	AddressLine1  string          `json:"addressLine1" form:"addressLine1" binding:"required"`
	AddressLine2  *string         `json:"addressLine2" form:"addressLine2" binding:"omitempty"`
	DocumentFront string          `json:"docFront" form:"docFront" binding:"required,url"`
}

func (k *KYC) handleBeginKyc(c *gin.Context) {
	var payload beginKycPayload
	if err := c.ShouldBind(&payload); err != nil {
		msgs := strings.Split(err.Error(), "\n")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": msgs, "code": "BAD_REQUEST"})
		return
	}

}

func (k *KYC) MountV1(r *gin.RouterGroup) {
	router := r.Group("/kyc")

	kycMiddleware := helpers.ProvideKYCMiddleware()
	secured := router.Group("", helpers.ProvideAuthMiddleware(), kycMiddleware(contract.KYCPending))
	secured.POST("", k.handleBeginKyc)
}

func NewKycController(l *slog.Logger, p *pgxpool.Pool, q *db.Queries, store contract.ObjectStorage, eb contract.EventBus) *KYC {
	maxUploadSize, err := strconv.Atoi(os.Getenv("MAX_UPLOAD_LIMIT"))
	if err != nil {
		panic(err)
	}
	return &KYC{
		logger:       l.With("controller", "kyc"),
		pool:         p,
		q:            q,
		UploadLimits: UploadLimits{maxUpload: int64(maxUploadSize)},
		eb:           eb,
		store:        store,
	}
}
