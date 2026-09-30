package controller

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/services/auth"
	"github.com/brinestone/mogtrade/web/helpers"
	eventpayloads "github.com/brinestone/mogtrade/web/payloads/events"
	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go-slim.dev/ioc"
)

type Auth struct {
	repo            *db.Queries
	logger          *slog.Logger
	pool            *pgxpool.Pool
	refreshLifetime time.Duration
	tokenEncoder    contract.TokenEncoder
	idg             contract.IdFactory
}

const (
	EventKeyUserCreatedV1        = "user.created.v1"
	EventKeyUserSignedInV1       = "user.login.v1"
	EventKeyRefreshTokenRotateV1 = "tokens.refresh.rotate.v1"
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

	a.logger.Debug("validation successful, signing in user", "email", request.Username, "type", "credential")
	tx, err := a.pool.Begin(c.Request.Context())
	if err != nil {
		a.logger.Error("unable to open transaction, aborting", "err", err.Error())
		helpers.InternalServerError(c)
		return
	}
	defer tx.Commit(c.Request.Context())

	result, err := auth.SignInUserByCredentials(c.Request.Context(), a.repo.WithTx(tx), a.tokenEncoder, auth.CredentialSignInInput{
		Identifier:           request.Username,
		Password:             request.Password,
		DeviceId:             request.DeviceId,
		RefreshTokenId:       a.idg(),
		RefreshTokenLifetime: a.refreshLifetime,
	})
	if err != nil {
		tx.Rollback(c.Request.Context())
	}
	if err != nil {
		if errors.Is(err, auth.ErrInavlidCredentials) || errors.Is(err, auth.ErrNoAuthAccountFound) {
			a.logger.Warn("sign in failed, aborting", "email", request.Username, "type", "credential", "err", err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
			return
		}
		a.logger.Error("sign in failed, aborting", "err", err.Error(), "email", request.Username)
		helpers.InternalServerError(c)
		return
	}

	c.JSON(http.StatusOK, result)
	a.logger.Info("sign in successful, emitting event")
	helpers.PublishEvent(c.Request.Context(), EventKeyUserSignedInV1, eventpayloads.UserSignedInEventArgs{
		Timestamp:  time.Now().UTC(),
		Identifier: request.Username,
		Provider:   db.AccountProviderCredential,
	})
}

func (a *Auth) handleCredentialRegister(c *gin.Context) {
	a.logger.Info("handling user sign-up request, validating request")
	var request httppayloads.CredentialSignUpRequest
	if err := c.ShouldBind(&request); err != nil {
		a.logger.Warn("request validation failed, aborting")
		c.JSON(http.StatusBadRequest, gin.H{"error": strings.Split(err.Error(), "\n")})
		return
	}
	result, err := ioc.Call2[auth.SignUpResult](c.Request.Context(), func(p *pgxpool.Pool, q *db.Queries, idg contract.IdFactory) (auth.SignUpResult, error) {
		a.logger.Debug("validation successful, creating user", "email", request.Email, "type", "credential")
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
			Photo:      fmt.Sprintf("https://api.dicebear.com/10.x/initials/svg?seed=%s", request.Initials()),
		})
		if err != nil {
			tx.Rollback(c.Request.Context())
		}
		return result, err
	})
	if err != nil {
		if errors.Is(err, auth.ErrAccountAlreadyExists) {
			a.logger.Warn("account already exists", "email", request.Email, "type", "credential")
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		a.logger.Error("user creation failed, aborting", "err", err.Error(), "email", request.Email)
		helpers.InternalServerError(c)
		return
	}

	_, err = ioc.Invoke(c.Request.Context(), func(bus contract.EventBus) {
		go bus.Publish(EventKeyUserCreatedV1, eventpayloads.UserCreatedEventArgs{
			UserId:    result.UserId,
			Timestamp: result.Timestamp,
		})
	})
	if err != nil {
		a.logger.Error("error while sending event", "event", "user.created", "err", err.Error())
	}
	c.Status(http.StatusCreated)
}

func (a *Auth) handleAccessTokenRefresh(c *gin.Context) {
	a.logger.Info("handling token refresh request, validating payload")
	var req httppayloads.RotateRefreshTokenRequest
	err := c.ShouldBindHeader(&req)
	if err != nil {
		a.logger.Warn("validation failed, aborting")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": strings.Split(err.Error(), "\n")})
		return
	}

	tx, err := a.pool.Begin(c.Request.Context())
	if err != nil {
		a.logger.Error("could not open database transaction", "err", err.Error())
		helpers.InternalServerError(c)
		return
	}
	defer tx.Rollback(c.Request.Context())

	sResult, err := auth.RotateAccessToken(c.Request.Context(), a.repo.WithTx(tx), a.tokenEncoder, auth.RotateAccessTokenInput{
		RefreshTokenId: a.idg(),
		Lifetime:       a.refreshLifetime,
		DeviceId:       req.DeviceId,
		Hash:           req.RefreshToken,
	})

	if err != nil {
		if errors.Is(err, auth.ErrRefreshTokenUnusable) || errors.Is(err, auth.ErrUserNotFound) || errors.Is(err, auth.ErrRefreshTokenNotFound) {
			a.logger.Warn("token error", "err", err.Error())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found or expired"})
			return
		}
		a.logger.Error("error while rotating access token", "err", err.Error())
		helpers.InternalServerError(c)
		return
	}
	tx.Commit(c.Request.Context())
	c.JSON(http.StatusOK, sResult)

	helpers.PublishEvent(c.Request.Context(), EventKeyRefreshTokenRotateV1, nil) // TODO: make an event arg for this
	a.logger.Info("refresh token rotated successfully!")
}

// handleEmailExistsCheck checks whether a user exists with the specified email address
func (a *Auth) handleEmailExistsCheck(c *gin.Context) {
	queries := make(map[string]string)
	if err := c.BindQuery(&queries); err != nil {
		a.logger.Error("could not parse query parameters", "err", err.Error())
		helpers.InternalServerError(c)
		return
	}

	email, found := queries["email"]
	if !found {
		a.logger.Warn("no email provided in query")
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}

	available, err := a.repo.IsEmailAvailable(c.Request.Context(), email)
	if err != nil {
		a.logger.Error("could not check for email availability", "err", err.Error())
		helpers.InternalServerError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"available": available})
}

func (a *Auth) MountV1(r *gin.RouterGroup) {
	public := r.Group("/auth")
	public.POST("/login/credential", a.handleCredentialLogin)
	public.POST("/register/credential", a.handleCredentialRegister)
	public.GET("/refresh", a.handleAccessTokenRefresh)
	public.GET("/email-available", a.handleEmailExistsCheck)
}

func NewAuthController(idg contract.IdFactory, l *slog.Logger, q *db.Queries, p *pgxpool.Pool, te contract.TokenEncoder) *Auth {
	var lifetime time.Duration
	lifetime, err := time.ParseDuration(os.Getenv("REFRESH_LIFETIME"))
	if err != nil {
		l.Warn("error while parsing refresh token lifetime environment variable, falling back to default", "var", "REFRESH_LIFETIME")
		lifetime = 7 * 24 * time.Hour
	}
	return &Auth{
		idg:             idg,
		repo:            q,
		logger:          l.With("controller", "auth"),
		pool:            p,
		refreshLifetime: lifetime,
		tokenEncoder:    te,
	}
}
