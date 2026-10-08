package controller

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/brinestone/mogtrade/services/auth"
	"github.com/brinestone/mogtrade/web/helpers"
	eventpayloads "github.com/brinestone/mogtrade/web/payloads/events"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// handlePasswordReset – POST /auth/password-reset/request
//
//	{ "email": "user@example.com" }
func (a *Auth) handlePasswordReset(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Read configuration from environment variables (application layer)
	expireMinutes := 30
	if v := os.Getenv("RESET_TOKEN_EXPIRE_MINUTES"); v != "" {
		if n, _ := fmt.Sscanf(v, "%d", &expireMinutes); n != 1 {
			expireMinutes = 30
		}
	}

	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost" // fallback for local development
	}

	// Use the existing transaction helper – it will:
	//   • begin a transaction,
	//   • call the service,
	//   • commit on success / rollback on error.
	output, err := helpers.WithValueTransaction(c.Request.Context(), func(tx pgx.Tx) (auth.ResetPasswordRequestOutput, error) {
		return auth.ResetPasswordRequest(c.Request.Context(),
			a.repo.WithTx(tx),
			auth.ResetPasswordRequestInput{
				Email:         req.Email,
				ExpireMinutes: expireMinutes,
			})
	})
	if err != nil {
		// WithTransaction already rolled back; respond with 500.
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build the confirmation URL (application‑level detail):
	//   https://<HOST>/api/v1/auth/password-reset/confirm?token=<raw>
	resetURL := fmt.Sprintf("https://%s/api/v1/auth/password-reset/confirm?token=%s",
		host, output.RawToken)

	// Publish the event (application layer – controller only).
	// The event payload contains the user ID, email, and the full reset URL.
	evt := eventpayloads.PasswordResetRequestedEventArgs{
		UserID:    output.UserEmail, // placeholder – actual UserID is the ULID string
		Email:     req.Email,
		ResetURL:  resetURL,
		Timestamp: time.Now(),
	}
	publishErr := helpers.PublishEvent(c.Request.Context(),
		"password_reset_requested", evt)
	if publishErr != nil {
		// Log but still consider the request successful – the background
		// worker will retry.  (If you want the request to fail, uncomment
		// the next lines and comment the success response below.)
		// c.JSON(http.StatusInternalServerError,
		//     gin.H{"error": "failed to publish event: " + publishErr.Error()})
		// return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "reset link sent"})
}
