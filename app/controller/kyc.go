package controller

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/services/kyc"
	"github.com/brinestone/mogtrade/web/helpers"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type KYC struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
	q      *db.Queries
	store  contract.ObjectStorage
	idg    contract.IdFactory
	eb     contract.EventBus
}

type beginKycPayload struct {
	Dob           time.Time       `json:"dateOfBirth" form:"dateOfBirth" binding:"required" time_format:"2006-01-02"`
	City          string          `json:"city" form:"city" binding:"required"`
	State         string          `json:"state" form:"state" binding:"required"`
	Gender        contract.Gender `json:"gender" form:"gender" binding:"required"`
	Selfie        string          `json:"selfie" form:"selfie" binding:"required"`
	Country       string          `json:"country" form:"country" binding:"required"`
	ClientId      string          `json:"clientId" form:"clientId" binding:"required"`
	LegalNames    string          `json:"legalNames" form:"legalNames" binding:"required"`
	DocumentBack  string          `json:"docBack" form:"docBack" binding:"required"`
	AddressLine1  string          `json:"addressLine1" form:"addressLine1" binding:"required"`
	AddressLine2  *string         `json:"addressLine2" form:"addressLine2" binding:"omitempty"`
	DocumentFront string          `json:"docFront" form:"docFront" binding:"required"`
	PhoneNumber   string          `json:"phoneNumber" form:"phoneNumber" binding:"required,phone"`
}

func (k *KYC) handleBeginKyc(c *gin.Context) {
	var payload beginKycPayload
	if err := c.ShouldBind(&payload); err != nil {
		msgs := strings.Split(err.Error(), "\n")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": msgs, "code": "BAD_REQUEST"})
		return
	}

	k.logger.Debug("opening database transaction")
	if err := helpers.WithTransaction(c.Request.Context(), func(tx pgx.Tx) error {
		return kyc.CreateRecord(c.Request.Context(), k.q.WithTx(tx), kyc.CreateParams{
			ID:     k.idg(),
			UserID: helpers.GetCurrentUserId(c),
		})
	}); err != nil {
		if errors.Is(err, kyc.ErrKycAlreadyExists) {
			c.Status(http.StatusProcessing)
			return
		}
		helpers.InternalServerError(c)
		return
	}

}

// GeneratePresignedUploadURL generates a presigned URL for uploading KYC documents
// to the MinIO bucket. The URL expires after the specified duration and only allows
// the specified content types (e.g., image/jpeg, image/png, application/pdf).
func (k *KYC) handleGeneratePresignedUploadURL(c *gin.Context) {
	exists, err := k.q.UserHasActiveKYCProfile(c.Request.Context(), new(helpers.GetCurrentUserId(c)))
	if err != nil {
		k.logger.Error("could not find user's active kyc profile", "uid", helpers.GetCurrentUserId(c), "err", err.Error())
		helpers.InternalServerError(c)
		return
	} else if exists {
		k.logger.Warn("user already has an active kyc profile. skipping", "uid", helpers.GetCurrentUserId(c))
		c.Status(http.StatusOK)
		return
	}
	// Get the document type from query parameter (front, back, selfie, proof_of_address)
	docTypes := strings.Split(c.Query("types"), ",")
	if len(docTypes) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing types parameter", "code": "BAD_REQUEST"})
		return
	}

	result := make(map[string]any)
	// Set allowed content types based on document type
	for _, docType := range docTypes {
		var allowedContentTypes []string
		switch docType {
		case "front":
			allowedContentTypes = []string{"image/jpeg", "image/png", "application/pdf"}
		case "back":
			allowedContentTypes = []string{"image/jpeg", "image/png", "application/pdf"}
		case "selfie":
			allowedContentTypes = []string{"image/jpeg", "image/png"}
		case "proof_of_address":
			allowedContentTypes = []string{"image/jpeg", "image/png", "application/pdf"}
		default:
			allowedContentTypes = []string{"image/jpeg", "image/png", "application/pdf"}
		}

		objectName := fmt.Sprintf("kyc/%s/%s-%s", helpers.GetCurrentUserId(c), docType, time.Now().Format("20060102150405"))
		expires := time.Hour * 24
		url, err := k.store.GeneratePresignedUploadURL(c.Request.Context(), contract.PresignUrlParams{
			ContentType: allowedContentTypes,
			ObjectName:  objectName,
			Window:      expires,
		})
		if err != nil {
			k.logger.Error("could not generate presigned url", "documentType", docType, "err", err.Error())
			helpers.InternalServerError(c)
			return
		}

		result[docType] = gin.H{
			"url":          url,
			"objectName":   objectName,
			"expiresAt":    time.Now().UTC().Add(expires).Unix(),
			"docType":      docType,
			"allowedTypes": allowedContentTypes,
		}
	}

	c.JSON(http.StatusOK, result)
}

func (k *KYC) MountV1(r *gin.RouterGroup) {
	router := r.Group("/kyc")

	// authOnly := router.Group("", helpers.ProvideAuthMiddleware())

	kycMiddleware := helpers.ProvideKYCMiddleware()
	secured := router.Group("", helpers.ProvideAuthMiddleware(), kycMiddleware(contract.KYCPending))
	secured.POST("", k.handleBeginKyc)

	// Public endpoint for generating presigned URLs for KYC document uploads
	public := router.Group("")
	public.GET("presigned-upload", k.handleGeneratePresignedUploadURL)
}

func NewKycController(l *slog.Logger, p *pgxpool.Pool, q *db.Queries, store contract.ObjectStorage, eb contract.EventBus, idg contract.IdFactory) *KYC {
	return &KYC{
		logger: l.With("controller", "kyc"),
		pool:   p,
		q:      q,
		eb:     eb,
		store:  store,
		idg:    idg,
	}
}
