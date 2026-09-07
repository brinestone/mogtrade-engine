package market

import (
	"context"
	"log/slog"
	"time"

	"github.com/brinestone/mogtrade/core/feed"
)

type ExchangePoller struct {
	hub       *ExchangeHub
	logger    *slog.Logger
	exchanges []feed.Datasource
}

func NewExchangePoller(l *slog.Logger, hub *ExchangeHub, sources []feed.Datasource) *ExchangePoller {
	return &ExchangePoller{logger: l, hub: hub, exchanges: sources}
}

func (p *ExchangePoller) Start(ctx context.Context) {
	p.logger.Info("pulling from exchanges", "exchange-count", len(p.exchanges))
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

		}
	}
}
