package controller

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/web/helpers"
	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
)

var (
	PayloadPaymentNotFound = gin.H{"error": "payment not found"}
)

type Payments struct {
	repo             *db.Queries
	logger           *slog.Logger
	depositProcessor contract.DepositProcessor
	idGenerator      contract.IdFactory
}

func (p *Payments) handleInitializeDeposit(c *gin.Context) {
	var req httppayloads.InitializeDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// paymentIntent, err := p.depositProcessor.InitializeDeposit(c.Request.Context(), mesomb.)
	// 	if err != nil {
	// 		p.logger.Error("failed to initialize deposit", "err", err.Error(), "user_id", helpers.GetCurrentUserId(c))
	// 		c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
	// 		return
	// 	}

	// c.JSON(http.StatusOK, httppayloads.ToPaymentIntentPayload(paymentIntent))
}

func (p *Payments) handleDepositStatus(c *gin.Context) {
	paymentID := c.Param("id")
	payment, err := p.depositProcessor.VerifyDeposit(context.Background(), paymentID)
	if err != nil {
		p.logger.Error("failed to verify deposit", "payment_id", paymentID, "err", err.Error(), "user_id", helpers.GetCurrentUserId(c))
		c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}

	c.JSON(http.StatusOK, httppayloads.ToPaymentIntentPayload(payment))
}

func (p *Payments) MountV1(r *gin.RouterGroup) {
	jwtMiddleware := helpers.ProvideJWTMiddleware()
	secured := r.Group("/payments", jwtMiddleware())
	{
		secured.POST("/deposits", p.handleInitializeDeposit)
		secured.GET("/deposits/:id", p.handleDepositStatus)
	}
}

func NewPaymentsController(repo *db.Queries, logger *slog.Logger, depositProcessor contract.DepositProcessor, idGenerator contract.IdFactory) *Payments {
	return &Payments{
		repo:             repo,
		logger:           logger.With("controller", "payments"),
		depositProcessor: depositProcessor,
		idGenerator:      idGenerator,
	}
}
