package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/brinestone/mogtrade/core/feed"
	"github.com/brinestone/mogtrade/infra"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/infra/events"
	"github.com/brinestone/mogtrade/infra/mail"
	"github.com/brinestone/mogtrade/services/auth"
	"github.com/brinestone/mogtrade/services/jobs"
	"github.com/brinestone/mogtrade/services/market"
	"github.com/brinestone/mogtrade/services/orders"
	"github.com/brinestone/mogtrade/web/adapter"
	"github.com/brinestone/mogtrade/web/api"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go-slim.dev/ioc"
)

func setupServices() error {
	if err := ioc.Factory(func() *market.ExchangeHub {
		return market.NewExchangeHub()
	}, true); err != nil {
		return err
	}

	if err := ioc.Factory(func(h *market.ExchangeHub, l *slog.Logger) *market.ExchangePoller {
		return market.NewExchangePoller(l.With("service", "exchange-poller"), h, []feed.Datasource{
			feed.NewMassiveDatasource(os.Getenv("MASSIVE_API_KEY"), feed.MassiveConfig{RequestTimeout: 10 * time.Second}),
			feed.NewAlphaVantageDatasource(os.Getenv("ALPHAVANTAGE_API_KEY"), feed.AlphaVantageConfig{RequestTimeout: 10 * time.Second}),
		})
	}, true); err != nil {
		return err
	}
	if err := ioc.Factory(orders.NewRiskEngine, true); err != nil {
		return err
	}
	return nil
}

func logTimeFormatter(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		t := a.Value.Time()
		formatted := t.Format(time.DateTime)
		return slog.String(slog.TimeKey, formatted)
	}
	return a
}

func setupDbConnection(ctx context.Context) error {
	pool, err := pgxpool.New(ctx, os.Getenv("DB_URL"))
	if err != nil {
		return err
	}
	if err = pool.Ping(ctx); err != nil {
		return err
	}

	ioc.Bind(pool)

	ioc.Factory(func(pool *pgxpool.Pool) *db.Queries {
		return db.New(pool)
	}, true)
	ioc.Factory(func(pool *pgxpool.Pool) (*pgxpool.Conn, error) {
		return pool.Acquire(ctx)
	})
	ioc.Factory(func(pool *pgxpool.Pool) infra.ConnProviderFunc {
		return func(ctx context.Context) (*pgxpool.Conn, error) {
			return pool.Acquire(ctx)
		}
	})
	return nil
}

func setupAdapters(ctx context.Context) error {
	err := ioc.Factory(func() (mail.Mailer, error) {
		sender := os.Getenv("MAILTRAP_API_KEY")
		senderName := "MogTrade"
		mailer, err := mail.NewMailtrapMailer(sender, senderName, func() string {
			return sender
		})
		if err != nil {
			return nil, err
		}
		return mailer, nil
	}, true)
	if err != nil {
		return err
	}
	ioc.Factory(func(l *slog.Logger) contract.JobScheduler {
		return adapter.NewCronJobScheduler(ctx, l.With("service", "job-scheduler"))
	}, true)
	ioc.Factory(func() events.EventBus {
		return adapter.UseInMemoryEventBus(ctx)
	}, true)
	ioc.Bind(adapter.UlidIdGenerator)
	ioc.Factory(func() *adapter.JwtAdapter {
		lifetime, err := time.ParseDuration(os.Getenv("JWT_LIFETIME"))
		if err != nil {
			panic(err)
		}
		origins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ";")
		hosts := make([]string, 0)
		for _, origin := range origins {
			u, err := url.Parse(origin)
			if err == nil && len(u.Host) > 0 {
				hosts = append(hosts, u.Host)
			}
		}
		return adapter.NewJwtTokenEncoder(os.Getenv("JWT_SECRET"), lifetime, hosts, os.Getenv("HOST"))
	})
	ioc.Factory(func(j *adapter.JwtAdapter) auth.TokenEncoder {
		return j
	})
	ioc.Factory(func(j *adapter.JwtAdapter) auth.TokenVerifier { return j })
	return nil
}

func parseVars() error {
	_port, err := strconv.ParseInt(os.Getenv("PORT"), 10, 32)
	if err != nil {
		return err
	}
	port = int(_port)
	return nil
}

func startJobScheduler(ctx context.Context) {
	ioc.Invoke(ctx, func(s contract.JobScheduler) {
		s.Start()
	})
}
func pullMarketFeed(ctx context.Context) {
	ioc.Invoke(ctx, func(e *market.ExchangePoller) {
		e.Start(ctx)
	})
}
func registerBackgroundJobs(ctx context.Context) error {
	_, err := ioc.Invoke(ctx, func(s contract.JobScheduler, pool *pgxpool.Pool, l *slog.Logger) error {
		err := s.RegisterJob(jobs.NewStaleTokenRemoverJob("0 0 * * 6", pool, l)) // Every sunday midnight
		if err != nil {
			panic(err)
		}
		return nil
	})
	return err
}

func setupLogging(ctx context.Context) error {
	var logger *slog.Logger
	if os.Getenv("LOGGING") != "enable" {
		logger = slog.Default()
	} else {
		gin.DisableConsoleColor()
		err := os.MkdirAll("logs", 0755) // Changed to MkdirAll for Windows safety
		if err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		accessHandle, err := os.OpenFile(filepath.Join("logs", "access.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		errHandler, err := os.OpenFile(filepath.Join("logs", "errors.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		// Open app log handle safely here
		appLogsHandle, err := os.OpenFile(filepath.Join("logs", "app.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		// Clean up all hooks at once when context drops
		go func() {
			defer accessHandle.Close()
			defer errHandler.Close()
			defer appLogsHandle.Close()
			<-ctx.Done()
		}()

		gin.DefaultWriter = io.MultiWriter(accessHandle, os.Stdout)
		gin.DefaultErrorWriter = io.MultiWriter(errHandler, os.Stderr)

		// Build the slog logger inside the setup function
		logger = slog.New(slog.NewMultiHandler(
			slog.NewJSONHandler(appLogsHandle, &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: logTimeFormatter}),
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: logTimeFormatter}),
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError, ReplaceAttr: logTimeFormatter}),
			// slog.NewJSONHandler(errHandler, &slog.HandlerOptions{Level: slog.LevelError, ReplaceAttr: logTimeFormatter}),
		))
	}

	logger = logger.With("app", "mogtrade")

	// Register it to the DI container right here
	if err := ioc.Factory(func() *slog.Logger {
		return logger
	}, true); err != nil {
		return err
	}
	return nil
}
func bootstrapApplication(ctx context.Context) {
	// 1. Set up logging first and let it inject dependencies
	if err := setupLogging(ctx); err != nil {
		panic(err)
	}

	if err := parseVars(); err != nil {
		panic(err)
	}
	if err := setupDbConnection(ctx); err != nil {
		panic(err)
	}
	if err := setupServices(); err != nil {
		panic(err)
	}
	if err := setupAdapters(ctx); err != nil {
		panic(err)
	}
	if err := api.SetupControllers(); err != nil {
		panic(err)
	}

	if err := registerBackgroundJobs(ctx); err != nil {
		panic(err)
	}
}
func startAsyncTasks(ctx context.Context) {
	go pullMarketFeed(ctx)
	startJobScheduler(ctx)
	go func() {
		<-ctx.Done()
		ioc.Invoke(context.TODO(), func(l *slog.Logger) {
			l.Info("shutting down...")
		})
	}()
}
