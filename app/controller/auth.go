package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/brinestone/mogtrade/core/encoding"
	"github.com/brinestone/mogtrade/infra"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/services/auth"
	"github.com/brinestone/mogtrade/web/payloads"
	"github.com/gin-gonic/gin"
	"go-slim.dev/ioc"
)

type Auth struct {
	repo       *db.Queries
	logger     *slog.Logger
	connGetter infra.ConnProviderFunc
}

func (a *Auth) handleCredentialLogin(c *gin.Context) {
	a.logger.Info("handling user login request, validating request")
	var request payloads.CredentialLoginRequest
	if err := c.ShouldBind(&request); err != nil {
		a.logger.Warn("request validation failed, aborting")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := ioc.Call2[auth.SignInResult](c.Request.Context(), func(q *db.Queries, te encoding.TokenEncoder, idg encoding.IdGeneratorFunc) (auth.SignInResult, error) {
		a.logger.Debug("validation successful, signing in user", "identifier", request.Username, "type", "credential")
		return auth.SignInUserByCredentials(c.Request.Context(), q, te, idg, auth.CredentialSignInInput{
			Identifier:           request.Username,
			Password:             request.Password,
			RefreshTokenLifetime: 5 * time.Hour,
		})
	})
	if err != nil {
		if errors.Is(err, auth.ErrInavlidCredentials) || errors.Is(err, auth.ErrNoAuthAccountFound) {
			a.logger.Warn("sign in failed, aborting", "identifier", request.Username, "type", "credential", "err", err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
			return
		}
		a.logger.Error("sign in failed, aborting", "err", err.Error(), "identifier", request.Username)
		c.AbortWithStatusJSON(http.StatusInternalServerError, errInternalServerErrorPayload)
		return
	}

	a.logger.Info("sign in successful")
	c.JSON(http.StatusOK, result)
}

func (a *Auth) MountV1(r *gin.RouterGroup) {
	router := r.Group("/auth")
	router.POST("/login/credential", a.handleCredentialLogin)
}

func NewAuthController(l *slog.Logger, q *db.Queries, cg infra.ConnProviderFunc) *Auth {
	return &Auth{
		q, l.With("controller", "auth"), cg,
	}
}
