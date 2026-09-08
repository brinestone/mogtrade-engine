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
	ticker := time.NewTicker(50 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.logger.Info("polling exchanges, not implemented actually", "exchange-count", len(p.exchanges))
			// for _, exchange := range p.exchanges {
			// 	res, err := exchange.Pull(feed.DatasourceQueryRequest{
			// 		Interval: feed.IVDay,
			// 		Symbol:   "NSQ",
			// 	})
			// 	if err != nil {
			// 		p.logger.Error("failed to pull from datasource", "name", exchange.Name(), "err", err.Error())
			// 		continue
			// 	}

			// 	for _, entry := range res {
			// 		p.hub.broadcaster <- ExchangeEvent{
			// 			Symbol:    "IBM",
			// 			Price:     decimal.NewFromFloat(entry.Volume),
			// 			Timestamp: time.Now().Unix(),
			// 		}
			// 	}
			// }
		}
	}
}
