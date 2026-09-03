package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/brinestone/mogtrade/core/encoding"
	"github.com/brinestone/mogtrade/infra"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/infra/events"
	"github.com/brinestone/mogtrade/services/auth"
	eventpayloads "github.com/brinestone/mogtrade/web/payloads/events"
	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go-slim.dev/ioc"
)

type Auth struct {
	repo       *db.Queries
	logger     *slog.Logger
	connGetter infra.ConnProviderFunc
}

const (
	EventKeyUserCreated = "user.created"
)

func (a *Auth) handleCredentialLogin(c *gin.Context) {
	a.logger.Info("handling user login request, validating request")
	var request httppayloads.CredentialLoginRequest
	if err := c.ShouldBind(&request); err != nil {
		a.logger.Warn("request validation failed, aborting")
		c.JSON(http.StatusBadRequest, gin.H{"error": strings.Split(err.Error(), "\n")})
		return
	}
	c.BindHeader(&request)

	errs := request.Validate()
	if len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": errs})
		return
	}

	result, err := ioc.Call2[auth.SignInResult](c.Request.Context(), func(cp infra.ConnProviderFunc, p *pgxpool.Pool, q *db.Queries, te encoding.TokenEncoder, idg encoding.IdGeneratorFunc) (auth.SignInResult, error) {
		a.logger.Debug("validation successful, signing in user", "identifier", request.Username, "type", "credential")
		tx, err := p.Begin(c.Request.Context())
		if err != nil {
			a.logger.Error("unable to open transaction, aborting")
			c.AbortWithStatusJSON(http.StatusInternalServerError, errInternalServerErrorPayload)
			return auth.SignInResult{}, err
		}
		defer tx.Commit(c.Request.Context())

		result, err := auth.SignInUserByCredentials(c.Request.Context(), q.WithTx(tx), te, idg, auth.CredentialSignInInput{
			Identifier:           request.Username,
			Password:             request.Password,
			DeviceId:             request.DeviceId,
			RefreshTokenLifetime: 7 * 24 * time.Hour,
		})
		if err != nil {
			tx.Rollback(c.Request.Context())
		}
		return result, err
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

func (a *Auth) handleCredentialRegister(c *gin.Context) {
	a.logger.Info("handling user sign-up request, validating request")
	var request httppayloads.CredentialSignUpRequest
	if err := c.ShouldBind(&request); err != nil {
		a.logger.Warn("request validation failed, aborting")
		c.JSON(http.StatusBadRequest, gin.H{"error": strings.Split(err.Error(), "\n")})
		return
	}
	result, err := ioc.Call2[auth.SignUpResult](c.Request.Context(), func(p *pgxpool.Pool, q *db.Queries, idg encoding.IdGeneratorFunc) (auth.SignUpResult, error) {
		a.logger.Debug("validation successful, creating user", "identifier", request.Email, "type", "credential")
		tx, err := p.Begin(c.Request.Context())
		if err != nil {
			a.logger.Error("unable to open transaction", "err", err.Error())
			return auth.SignUpResult{}, err
		}
		defer tx.Commit(c.Request.Context())

		result, err := auth.SignUpUserByCredentials(c.Request.Context(), q.WithTx(tx), idg, auth.CredentialSignUpInput{
			Name:       request.Names,
			Identifier: request.Email,
			Password:   request.Password,
			Email:      request.Email,
		})
		if err != nil {
			tx.Rollback(c.Request.Context())
		}
		return result, err
	})
	if err != nil {
		if errors.Is(err, auth.ErrAccountAlreadyExists) {
			a.logger.Warn("account already exists", "identifier", request.Email, "type", "credential")
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		a.logger.Error("user creation failed, aborting", "err", err.Error(), "identifier", request.Email)
		c.AbortWithStatusJSON(http.StatusInternalServerError, errInternalServerErrorPayload)
		return
	}

	_, err = ioc.Invoke(c.Request.Context(), func(bus events.EventBus) {
		bus.Publish(EventKeyUserCreated, eventpayloads.UserCreatedEventArgs{
			UserId:    result.UserId,
			Timestamp: result.Timestamp,
		})
	})
	if err != nil {
		a.logger.Error("error while sending event", "event", "user.created", "err", err.Error())
	}
	c.Status(http.StatusCreated)
}

func (a *Auth) MountV1(r *gin.RouterGroup) {
	router := r.Group("/auth")
	router.POST("/login/credential", a.handleCredentialLogin)
	router.POST("/register/credential", a.handleCredentialRegister)
}

func NewAuthController(l *slog.Logger, q *db.Queries, cg infra.ConnProviderFunc) *Auth {
	return &Auth{
		q, l.With("controller", "auth"), cg,
	}
}
