package controller

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/web/helpers"
	httppayloads "github.com/brinestone/mogtrade/web/payloads/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	logger  *slog.Logger
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewUserController(l *slog.Logger, p *pgxpool.Pool) *User {
	return &User{
		logger:  l.With("controller", "user"),
		pool:    p,
		queries: db.New(p),
	}
}

func (c *User) MountV1(r *gin.RouterGroup) {
	requireAuth := helpers.ProvideAuthMiddleware()
	router := r.Group("/user", requireAuth)
	router.GET("/prefs", c.handleGetPrefs)
	router.PUT("/prefs", c.handleUpsertPrefs)
}

func (c *User) handleUpsertPrefs(ctx *gin.Context) {
	l := c.logger.With("uid", helpers.GetCurrentUserId(ctx))
	_, err := helpers.GetCurrentUser(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Warn("user account not found", "uid", helpers.GetCurrentUserId(ctx))
			ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		l.Error("could not get user prefs", "err", err.Error())
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}
	var req httppayloads.UpdateUserPrefsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": strings.Split(err.Error(), "\n")})
		return
	}

	tx, err := c.pool.Begin(ctx.Request.Context())
	if err != nil {
		l.Error("could not open database transaction", "err", err.Error())
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}
	defer tx.Rollback(ctx.Request.Context())

	serialized, err := json.Marshal(req)
	if err != nil {
		l.Error("could not marshal prefs to json", "err", err.Error())
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}
	err = c.queries.WithTx(tx).UpdateUserPrefs(ctx.Request.Context(), db.UpdateUserPrefsParams{
		ID:    helpers.GetCurrentUserId(ctx),
		Prefs: serialized,
	})
	if err == nil {
		tx.Commit(ctx.Request.Context())
		ctx.Status(http.StatusAccepted)
		return
	}

	l.Error("could not commit transaction", "err", err.Error())
	ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
}

func (c *User) handleGetPrefs(ctx *gin.Context) {
	uid := helpers.GetCurrentUserId(ctx)
	u, err := c.queries.FindUserById(ctx.Request.Context(), uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.logger.Warn("user account not found", "uid", uid)
			ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.logger.Error("could not get user prefs", "err", err.Error())
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}

	var prefs map[string]any = make(map[string]any)

	if err = json.Unmarshal(u.Prefs, &prefs); err != nil {
		c.logger.Error("could not parse user prefs json", "err", err.Error())
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, httppayloads.ErrInternalServerErrorPayload)
		return
	}
	ctx.JSON(http.StatusOK, prefs)
}
