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

func (k *KYC) handleBeginKyc(c *gin.Context) {
	k.logger.Debug("opening database transaction")
	row, err := helpers.WithValueTransaction(c.Request.Context(), func(tx pgx.Tx) (db.LookupActiveKycForUserRow, error) {
		return kyc.CreateRecord(c.Request.Context(), k.q.WithTx(tx), kyc.CreateParams{
			ID:     k.idg(),
			UserID: helpers.GetCurrentUserId(c),
		})
	})
	if err != nil {
		if errors.Is(err, kyc.ErrKycAlreadyExists) {
			k.logger.Warn("kyc record already exists for user, skipping", "uid", helpers.GetCurrentUserId(c))
			c.Status(http.StatusProcessing)
			return
		}
		k.logger.Error("could not create kyc flow", "err", err.Error())
		helpers.InternalServerError(c)
		return
	}

	println(row.VerifyBefore.Time.String())
	c.JSON(http.StatusOK, gin.H{
		"expiresAt": row.VerifyBefore.Time.UTC().Format(time.RFC3339),
	})
}

// GeneratePresignedUploadURL generates a presigned URL for uploading KYC documents
// to the MinIO bucket. The URL expires after the specified duration and only allows
// the specified content types (e.g., image/jpeg, image/png, application/pdf).
func (k *KYC) handleGeneratePresignedUploadURL(c *gin.Context) {
	uid := helpers.GetCurrentUserIdOrDefault(c, fmt.Sprintf("anonymous-%s", c.ClientIP()))
	// Get the document type from query parameter (front, back, selfie, proof_of_address)
	docTypes := strings.Split(c.Query("types"), ",")
	if len(docTypes) == 0 || (len(docTypes) == 1 && docTypes[0] == "") {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing types parameter", "code": "BAD_REQUEST"})
		return
	}

	result := make(map[string]any)
	validDocTypes := []string{"selfie", "back", "front", "proof_of_address"}
	invalidDocTypes := helpers.SliceDiff(docTypes, validDocTypes)
	if len(invalidDocTypes) > 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid document types: \"%s\". document type must be among: \"%s\"", strings.Join(invalidDocTypes, ", "), strings.Join(validDocTypes, ", ")),
		})
		return
	}

	// Set allowed content types based on document type
	for _, docType := range docTypes {
		var allowedContentTypes []string
		switch docType {
		case "front":
			allowedContentTypes = []string{"image/jpeg", "image/png"}
		case "back":
			allowedContentTypes = []string{"image/jpeg", "image/png"}
		case "selfie":
			allowedContentTypes = []string{"image/jpeg", "image/png"}
		case "proof_of_address":
			allowedContentTypes = []string{"image/jpeg", "image/png", "application/pdf"}
		default:
			allowedContentTypes = []string{"image/jpeg", "image/png", "application/pdf"}
		}

		objectName := fmt.Sprintf("kyc/%s/%s-%s", uid, docType, time.Now().Format("20060102150405"))
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
			"allowedTypes": allowedContentTypes,
			"httpMethod":   "PUT",
		}
	}

	c.JSON(http.StatusOK, result)
}

func (k *KYC) handleGetPendingRequirements(c *gin.Context) {

}

func (k *KYC) MountV1(r *gin.RouterGroup) {
	router := r.Group("/kyc")

	kycMiddleware := helpers.ProvideKYCMiddleware()
	jwtMiddleware := helpers.ProvideJWTMiddleware()
	secured := router.Group("", jwtMiddleware(false))
	secured.GET("", kycMiddleware(contract.KYCPending), k.handleBeginKyc)
	secured.GET("pending", kycMiddleware(contract.KYCPending), k.handleGetPendingRequirements)

	// Public endpoint for generating presigned URLs for KYC document uploads
	public := router.Group("")
	public.GET("presigned-upload", jwtMiddleware(true), k.handleGeneratePresignedUploadURL)
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
