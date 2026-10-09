package controller

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/brinestone/mogtrade/services/auth"
	"github.com/brinestone/mogtrade/web/helpers"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// handlePasswordResetConfirm – GET /auth/password-reset/confirm?token=<token>
// Validates the token and shows the password reset form
func (a *Auth) handlePasswordResetConfirm(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
		return
	}

	// Look up the verification row
	ver, err := a.repo.GetUnusedVerificationByToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to look up verification"})
		return
	}

	if time.Now().After(ver.ExpiresAt.Time) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token expired"})
		return
	}

	err = helpers.WithTransaction(c.Request.Context(), func(tx pgx.Tx) error {
		return auth.PrimeVerification(c.Request.Context(), a.repo.WithTx(tx), token)
	})
	if err != nil {
		if errors.Is(err, auth.ErrVerificationNotFound) {
			a.logger.Warn("verification not found or used", "token", token)
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "verification expired or already used"})
			return
		}
		a.logger.Error("could not prime verification", "token", token, "err", err.Error())
		helpers.InternalServerError(c)
		return
	}

	if ver.CallbackUrl != nil {
		u, err := url.Parse(*ver.CallbackUrl)
		if err != nil {
			a.logger.Warn("invalid callback url for verification", "vId", ver.ID, "err", err.Error(), "callbackUrl", *ver.CallbackUrl)
			helpers.InternalServerError(c)
			return
		}
		u.Query().Add("vt", token)
		c.Redirect(http.StatusTemporaryRedirect, u.String())
		return
	}
	c.Status(http.StatusAccepted)
}

// handlePasswordResetSubmit – POST /auth/password-reset/submit
//
//	{ "token": "...", "new_password": "..." }
func (a *Auth) handlePasswordResetSubmit(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=12"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": strings.Split(err.Error(), "\n")})
		return
	}

	err := helpers.WithTransaction(c.Request.Context(), func(tx pgx.Tx) error {
		return auth.FinishPasswordReset(c.Request.Context(), a.repo.WithTx(tx), auth.FinishPasswordResetParams{
			NewPassword:       req.NewPassword,
			VerificationToken: req.Token,
		})
	})
	if err != nil {
		if errors.Is(err, auth.ErrVerificationNotFound) {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": "verification token either expired or already used", "status": "UNPROCESSIBLE_REQUEST"})
			return
		} else if errors.Is(err, auth.ErrNoAuthAccountFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "account not found", "status": "NOT_FOUND"})
			return
		}
		a.logger.Error("could not reset user password", "err", err.Error())
		helpers.InternalServerError(c)
		return
	}
	c.Status(http.StatusAccepted)
}
