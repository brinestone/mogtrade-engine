package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/infra"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/infra/events"
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
	connGetter      infra.ConnProviderFunc
	refreshLifetime time.Duration
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

	result, err := ioc.Call2[auth.SignInResult](c.Request.Context(), func(cp infra.ConnProviderFunc, p *pgxpool.Pool, q *db.Queries, te auth.TokenEncoder, idg contract.IdGeneratorFunc) (auth.SignInResult, error) {
		a.logger.Debug("validation successful, signing in user", "identifier", request.Username, "type", "credential")
		tx, err := p.Begin(c.Request.Context())
		if err != nil {
			a.logger.Error("unable to open transaction, aborting")
			c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
			return auth.SignInResult{}, err
		}
		defer tx.Commit(c.Request.Context())

		result, err := auth.SignInUserByCredentials(c.Request.Context(), q.WithTx(tx), te, idg, auth.CredentialSignInInput{
			Identifier:           request.Username,
			Password:             request.Password,
			DeviceId:             request.DeviceId,
			RefreshTokenLifetime: a.refreshLifetime,
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
		c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
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
	result, err := ioc.Call2[auth.SignUpResult](c.Request.Context(), func(p *pgxpool.Pool, q *db.Queries, idg contract.IdGeneratorFunc) (auth.SignUpResult, error) {
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
		c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}

	_, err = ioc.Invoke(c.Request.Context(), func(bus events.EventBus) {
		bus.Publish(EventKeyUserCreatedV1, eventpayloads.UserCreatedEventArgs{
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

	result, err := ioc.Call2[auth.SignInResult](c.Request.Context(), func(p *pgxpool.Pool, q *db.Queries, idg contract.IdGeneratorFunc, t auth.TokenEncoder) (auth.SignInResult, error) {
		tx, err := p.Begin(c.Request.Context())
		if err != nil {
			a.logger.Error("could not open database transaction", "err", err.Error())
			return auth.SignInResult{}, err
		}
		defer tx.Rollback(c.Request.Context())

		sResult, err := auth.RotateAccessToken(c.Request.Context(), q.WithTx(tx), idg, t, auth.RotateAccessTokenInput{
			Lifetime: a.refreshLifetime,
			DeviceId: req.Deviceid,
			Hash:     req.RefreshToken,
		})

		tx.Commit(c.Request.Context())
		return sResult, err
	})
	if err != nil {
		if errors.Is(err, auth.ErrRefreshTokenUnusable) || errors.Is(err, auth.ErrUserNotFound) || errors.Is(err, auth.ErrRefreshTokenNotFound) {
			a.logger.Warn("token error", "err", err.Error())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found or expired"})
			return
		}
		a.logger.Error("error while rotating access token", "err", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}

	helpers.PublishEvent(c.Request.Context(), EventKeyRefreshTokenRotateV1, nil) // TODO: make an event arg for this
	a.logger.Info("refresh token rotated successfully!")
	c.JSON(http.StatusOK, result)
}

// handleEmailExistsCheck checks whether a user exists with the specified email address
func (a *Auth) handleEmailExistsCheck(c *gin.Context) {
	queries := make(map[string]string)
	if err := c.BindQuery(&queries); err != nil {
		a.logger.Error("could not parse query parameters", "err", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
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
		c.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}
	c.JSON(http.StatusOK, gin.H{"available": available})
}

func (a *Auth) MountV1(r *gin.RouterGroup) {
	router := r.Group("/auth")
	router.POST("/login/credential", a.handleCredentialLogin)
	router.POST("/register/credential", a.handleCredentialRegister)
	router.GET("/refresh", a.handleAccessTokenRefresh)
	router.GET("/email-available", a.handleEmailExistsCheck)
}

func NewAuthController(l *slog.Logger, q *db.Queries, cg infra.ConnProviderFunc) *Auth {
	var lifetime time.Duration
	lifetime, err := time.ParseDuration(os.Getenv("REFRESH_LIFETIME"))
	if err != nil {
		l.Warn("error while parsing refresh token lifetime environment variable, falling back to default", "var", "REFRESH_LIFETIME")
		lifetime = 7 * 24 * time.Hour
	}
	return &Auth{
		q, l.With("controller", "auth"), cg, lifetime,
	}
}
