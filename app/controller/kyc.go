package controller

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
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

	c.JSON(http.StatusOK, gin.H{
		"expiresAt": row.VerifyBefore.Time.UTC().Format(time.RFC3339),
	})
}

// GeneratePresignedUploadURL generates a presigned URL for uploading KYC documents
// to the object store.
func (k *KYC) handleGeneratePresignedUploadURL(c *gin.Context) {
	uid := helpers.GetCurrentUserIdOrDefault(c, fmt.Sprintf("anonymous-%s", c.ClientIP()))
	docTypes := strings.Split(c.Query("types"), ",")
	if len(docTypes) == 0 || (len(docTypes) == 1 && docTypes[0] == "") {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing types parameter", "code": "BAD_REQUEST"})
		return
	}

	validDocTypes := []string{"selfie", "back", "front", "proof_of_address"}
	invalidDocTypes := helpers.SliceDiff(docTypes, validDocTypes)
	if len(invalidDocTypes) > 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid document types: \"%s\". document type must be among: \"%s\"", strings.Join(invalidDocTypes, ", "), strings.Join(validDocTypes, ", ")),
		})
		return
	}

	result := make(map[string]any)
	for _, docType := range docTypes {
		var allowedContentTypes []string
		switch docType {
		case "front", "back", "selfie":
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
	userID := helpers.GetCurrentUserId(c)
	statusStr := string(contract.KYCPending)

	record, err := kyc.GetByUser(c.Request.Context(), k.q, userID)
	if err != nil {
		if errors.Is(err, kyc.ErrKycNotFound) {
			k.logger.Warn("kyc record not found", "uid", userID)
			c.JSON(http.StatusNoContent, gin.H{
				"status":  "no_kyc_record",
				"pending": []string{"submit_kyc_application"},
				"message": "No KYC record found for this user. Please start the KYC verification process.",
			})
			return
		}
		k.logger.Error("could not get kyc record", "uid", userID, "err", err.Error())
		helpers.InternalServerError(c)
		return
	}

	statusStr = string(record.Status)
	var pending []string
	switch statusStr {
	case string(contract.KYCPending):
		if record.Ready == nil || !*record.Ready {
			pending = append(pending, "complete_profile")
		}
		pending = append(pending, "upload_documents")
	case string(contract.KYCVerified):
		pending = append(pending, "none")
	case string(contract.KYCVerifying):
		pending = append(pending, "admin_review_in_progress")
	case string(contract.KYCRejected):
		pending = append(pending, "resubmit_kyc", "provide_additional_documents")
	case string(contract.KYCExpired):
		pending = append(pending, "renew_kyc")
	default:
		pending = append(pending, "review_kyc_status")
	}

	result := gin.H{
		"status":       statusStr,
		"expiresAt":    record.ExpiresAt,
		"verifyBefore": record.VerifyBefore,
		"pending":      pending,
		"isReady":      record.Ready != nil && *record.Ready,
	}

	c.JSON(http.StatusOK, result)
}

type UpdateProfileInput struct {
	LegalNames string          `json:"legal_names" binding:"required"`
	Gender     contract.Gender `json:"gender" binding:"required"`
	Dob        string          `json:"dob" binding:"required"`
	Address    any             `json:"address"`
}

func (k *KYC) handleUpdateProfile(c *gin.Context) {
	userID := helpers.GetCurrentUserId(c)

	var input UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	validGenders := []contract.Gender{contract.GenderMale, contract.GenderFemale, contract.GenderUndefined}
	if !slices.Contains(validGenders, input.Gender) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid gender", "validGenders": validGenders})
		return
	}

	dob, err := time.Parse("2006-01-02", input.Dob)
	if err != nil {
		dob, err = time.Parse(time.RFC3339, input.Dob)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dob format, expected YYYY-MM-DD"})
			return
		}
	}

	err = kyc.UpdateProfile(c.Request.Context(), k.q, kyc.UpdateProfileParams{
		UserID:     userID,
		LegalNames: input.LegalNames,
		Gender:     input.Gender,
		Dob:        dob,
		Address:    input.Address,
	})
	if err != nil {
		k.logger.Error("could not update kyc profile", "uid", userID, "err", err.Error())
		helpers.InternalServerError(c)
		return
	}

	k.logger.Info("kyc profile updated", "uid", userID)
	c.JSON(http.StatusOK, gin.H{
		"status":  "profile_updated",
		"message": "KYC profile updated successfully",
	})
}

type DocumentUploadItem struct {
	ObjectName  string `json:"object_name"`
	URL         string `json:"url"`
	Fingerprint string `json:"fingerprint" binding:"required,len=64"`
}

type BatchUpdateDocsInput struct {
	Front          *DocumentUploadItem `json:"front"`
	Back           *DocumentUploadItem `json:"back"`
	Selfie         *DocumentUploadItem `json:"selfie"`
	ProofOfAddress *DocumentUploadItem `json:"proof_of_address"`
	BaseURI        string              `json:"base_uri"`
}

func (k *KYC) handleUpdateDocuments(c *gin.Context) {
	userID := helpers.GetCurrentUserId(c)

	var input BatchUpdateDocsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	if input.Front == nil && input.Back == nil && input.Selfie == nil && input.ProofOfAddress == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one document must be provided"})
		return
	}

	validateDoc := func(docType string, item *DocumentUploadItem) error {
		if item == nil {
			return nil
		}
		obj, err := k.store.FindObjectWithFingerprint(c.Request.Context(), contract.FindObjectWithFingerprintParams{
			ObjectName:  item.ObjectName,
			Fingerprint: item.Fingerprint,
		})
		if err != nil {
			k.logger.Error("failed checking object fingerprint", "type", docType, "err", err.Error())
			return err
		}
		if obj.Fingerprint == "" && obj.Name == "" {
			return fmt.Errorf("document '%s' not found or fingerprint mismatch in storage", docType)
		}
		return nil
	}

	docs := map[string]*DocumentUploadItem{
		"front":            input.Front,
		"back":             input.Back,
		"selfie":           input.Selfie,
		"proof_of_address": input.ProofOfAddress,
	}

	for docType, docItem := range docs {
		if err := validateDoc(docType, docItem); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	identityDoc := contract.IdentityDocument{
		URI: input.BaseURI,
	}
	if input.Front != nil {
		identityDoc.FrontHash = input.Front.Fingerprint
		if identityDoc.URI == "" {
			identityDoc.URI = input.Front.URL
		}
	}
	if input.Back != nil {
		identityDoc.BackHash = input.Back.Fingerprint
	}
	if input.Selfie != nil {
		identityDoc.SelfieHash = input.Selfie.Fingerprint
	}

	proofOfAddress := contract.ProofOfAddressDocument{}
	if input.ProofOfAddress != nil {
		proofOfAddress.DocHash = input.ProofOfAddress.Fingerprint
		proofOfAddress.URI = input.ProofOfAddress.URL
	}

	err := kyc.UpdateDocuments(c.Request.Context(), k.q, kyc.UpdateDocumentsParams{
		UserID:         userID,
		IdentityDoc:    identityDoc,
		ProofOfAddress: proofOfAddress,
	})
	if err != nil {
		k.logger.Error("failed to update kyc documents", "uid", userID, "err", err.Error())
		helpers.InternalServerError(c)
		return
	}

	k.logger.Info("kyc documents batch updated", "uid", userID)
	c.JSON(http.StatusOK, gin.H{
		"status":  "documents_updated",
		"message": "KYC documents successfully updated",
	})
}

type BeginVerifyingInput struct {
	UserID string `json:"user_id" binding:"required"`
}

func (k *KYC) handleBeginVerifying(c *gin.Context) {
	var input BeginVerifyingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	err := kyc.MarkVerifying(c.Request.Context(), k.q, input.UserID)
	if err != nil {
		k.logger.Error("could not mark kyc as verifying", "target_uid", input.UserID, "err", err.Error())
		helpers.InternalServerError(c)
		return
	}

	k.logger.Info("kyc marked as verifying", "target_uid", input.UserID)
	c.JSON(http.StatusOK, gin.H{
		"status":  "verifying",
		"user_id": input.UserID,
		"message": "KYC verification started",
	})
}

type ApproveInput struct {
	RiskProfile contract.KYCRiskProfile `json:"risk_profile"`
}

func (k *KYC) handleApprove(c *gin.Context) {
	targetUser := c.Param("user")
	verifierID := helpers.GetCurrentUserId(c)
	if targetUser == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}

	var input ApproveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		input.RiskProfile = contract.KYCRiskLow
	}
	if input.RiskProfile == "" {
		input.RiskProfile = contract.KYCRiskLow
	}

	err := kyc.Verify(c.Request.Context(), k.q, kyc.VerifyParams{
		UserID:      targetUser,
		VerifiedBy:  verifierID,
		RiskProfile: input.RiskProfile,
	})
	if err != nil {
		k.logger.Error("could not approve kyc", "target_uid", targetUser, "verifier", verifierID, "err", err.Error())
		helpers.InternalServerError(c)
		return
	}

	k.logger.Info("kyc verified", "target_uid", targetUser, "verifier", verifierID, "risk_profile", input.RiskProfile)
	c.JSON(http.StatusOK, gin.H{
		"status":       "verified",
		"user_id":      targetUser,
		"risk_profile": input.RiskProfile,
		"message":      "KYC verified successfully",
	})
}

type RejectInput struct {
	Reason string `json:"reason" binding:"required"`
}

func (k *KYC) handleReject(c *gin.Context) {
	targetUser := c.Param("user")
	verifierID := helpers.GetCurrentUserId(c)
	if targetUser == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}

	var input RejectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required"})
		return
	}

	err := kyc.Reject(c.Request.Context(), k.q, kyc.RejectParams{
		UserID:          targetUser,
		VerifiedBy:      verifierID,
		RejectionReason: input.Reason,
	})
	if err != nil {
		k.logger.Error("could not reject kyc", "target_uid", targetUser, "verifier", verifierID, "err", err.Error())
		helpers.InternalServerError(c)
		return
	}

	k.logger.Info("kyc rejected", "target_uid", targetUser, "verifier", verifierID, "reason", input.Reason)
	c.JSON(http.StatusOK, gin.H{
		"status":   "rejected",
		"user_id":  targetUser,
		"reason":   input.Reason,
		"message":  "KYC application rejected",
	})
}

func (k *KYC) handleListApplications(c *gin.Context) {
	riskParam := c.Query("risk_profile")
	var records []contract.LookupKycRecord
	var err error

	if riskParam != "" {
		records, err = kyc.ListByRiskProfile(c.Request.Context(), k.q, contract.KYCRiskProfile(riskParam))
	} else {
		records, err = kyc.List(c.Request.Context(), k.q)
	}

	if err != nil {
		k.logger.Error("failed to list kyc records", "err", err.Error())
		helpers.InternalServerError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"count":   len(records),
	})
}

func (k *KYC) MountV1(r *gin.RouterGroup) {
	router := r.Group("/kyc")

	kycMiddleware := helpers.ProvideKYCMiddleware()
	jwtMiddleware := helpers.ProvideJWTMiddleware()
	verifierMiddleware := helpers.ProvideRoleMiddleware()

	secured := router.Group("", jwtMiddleware(false))
	secured.POST("", kycMiddleware(contract.KYCPending), k.handleBeginKyc)
	secured.GET("", kycMiddleware(contract.KYCPending), k.handleBeginKyc)
	secured.GET("/pending", kycMiddleware(contract.KYCPending), k.handleGetPendingRequirements)
	secured.POST("/profile", kycMiddleware(contract.KYCPending), k.handleUpdateProfile)
	secured.POST("/documents", kycMiddleware(contract.KYCPending), k.handleUpdateDocuments)
	secured.POST("/update-docs", kycMiddleware(contract.KYCPending), k.handleUpdateDocuments)

	verifier := secured.Group("", verifierMiddleware(contract.RoleKycVerifier, contract.RoleAdmin, contract.RoleCompliance))
	verifier.GET("/applications", k.handleListApplications)
	verifier.POST("/verify", k.handleBeginVerifying)
	verifier.PATCH("/:user/reject", k.handleReject)
	verifier.PATCH("/:user/approve", k.handleApprove)

	// Presigned URLs for KYC document uploads
	public := router.Group("")
	public.GET("/presigned-upload", jwtMiddleware(true), k.handleGeneratePresignedUploadURL)
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
