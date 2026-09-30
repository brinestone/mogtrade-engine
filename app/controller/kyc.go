package controller

import (
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"slices"
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
	LegalNames   string          `json:"legalNames" form:"legalNames" binding:"required"`
	Dob          time.Time       `json:"dateOfBirth" form:"dateOfBirth" binding:"required" time_format:"2006-01-02"`
	Gender       contract.Gender `json:"gender" form:"gender" binding:"required"`
	AddressLine1 string          `json:"addressLine1" form:"addressLine1" binding:"required"`
	AddressLine2 *string         `json:"addressLine2" form:"addressLine2" binding:"omitempty"`
	City         string          `json:"city" form:"city" binding:"required"`
	Country      string          `json:"country" form:"country" binding:"required"`
	State        string          `json:"state" form:"state" binding:"required"`
	ClientId     string          `json:"clientId" form:"clientId" binding:"required"`
}

func (k *KYC) handleBeginKyc(c *gin.Context) {
	var payload beginKycPayload
	if err := c.ShouldBind(&payload); err != nil {
		msgs := strings.Split(err.Error(), "\n")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": msgs, "code": "BAD_REQUEST"})
		return
	}

	front, err := c.FormFile("front")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "front file not provided", "code": "BAD_REQUEST"})
		return
	}
	if err := k.validateUploadFile(front, "image/png", "image/jpeg"); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	back, err := c.FormFile("back")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "back file not provided", "code": "BAD_REQUEST"})
		return
	}
	if err := k.validateUploadFile(back, "image/png", "image/jpeg"); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	selfie, err := c.FormFile("selfie")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "selfie file not provided", "code": "BAD_REQUEST"})
		return
	}
	if err := k.validateUploadFile(selfie, "image/png", "image/jpeg"); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}

func (k *KYC) validateUploadFile(header *multipart.FileHeader, requiredMime ...string) error {
	if header.Size > k.maxUpload {
		return fmt.Errorf("%s must be at most %.0f KB", header.Filename, float64(k.maxUpload/1024.0))
	}

	opened, err := header.Open()
	if err != nil {
		return fmt.Errorf("failed to read file")
	}
	defer opened.Close()

	buf := make([]byte, 512)
	if _, err := opened.Read(buf); err != nil {
		return err
	}

	mimeType := http.DetectContentType(buf)

	if !slices.Contains(requiredMime, mimeType) {
		return fmt.Errorf("invalid mimetype: %s", mimeType)
	}
	return nil
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
