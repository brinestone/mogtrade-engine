package controller

import (
	"encoding/json"
	"io"
	"log/slog"
	"time"

	"github.com/brinestone/mogtrade/services/market"
	"github.com/brinestone/mogtrade/web/helpers"
	"github.com/gin-gonic/gin"
)

type Market struct {
	logger *slog.Logger
	hub    *market.ExchangeHub
}

func (c *Market) handlePullLiveFeed(ctx *gin.Context) {
	l := c.logger.With("client-ip", ctx.ClientIP())
	connected := time.Now()
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("X-Accel-Buffering", "no")

	client := market.NewHubClient()
	c.hub.RegisterClient(client)
	l.Info("client connected")
	defer c.hub.UnRegisterClient(client)

	ctx.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Request.Context().Done():
			l.Info("client disconnected", "duration", time.Since(connected))
			return false
		case ev, ok := <-client:
			if !ok {
				return false
			}
			payload, err := json.Marshal(ev)
			if err != nil {
				return true
			}
			ctx.SSEvent("ticker", string(payload))
			return true
		}
	})
}

func (c *Market) MountV1(r *gin.RouterGroup) {
	authMiddleware := helpers.ProvideAuthMiddleware()
	router := r.Group("/market", authMiddleware)
	router.GET("/feed", c.handlePullLiveFeed)
}

func NewFeedController(l *slog.Logger, h *market.ExchangeHub) *Market {
	return &Market{
		logger: l.With("controller", "feed"),
		hub:    h,
	}
}
