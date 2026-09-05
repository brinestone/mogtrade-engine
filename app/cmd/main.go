package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions/redis"

	"github.com/brinestone/mogtrade/infra"
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/infra/events"
	"github.com/brinestone/mogtrade/services/auth"
	"github.com/brinestone/mogtrade/services/orders"
	"github.com/brinestone/mogtrade/web/api"
	"github.com/brinestone/mogtrade/web/contract"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go-slim.dev/ioc"
)

var (
	port int
)

func main() {
	godotenv.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	store, err := redis.NewStore(10, "tcp", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_USER"), os.Getenv("REDIS_PWD"))
	if err != nil {
		panic(err)
	}

	engine := gin.Default()
	baseRouter := engine.Group("/api")
	if err := api.MountApiV1(baseRouter, api.ApiConfig{
		Host:           os.Getenv("HOST"),
		SessionStore:   store,
		AllowedOrigins: strings.Split(os.Getenv("ALLOWED_ORIGINS"), ";"),
	}); err != nil {
		panic(err)
	}
	engine.Run(fmt.Sprintf(":%d", port))
}

func setupServices() error {
	if err := ioc.Factory(orders.NewRiskEngine, true); err != nil {
		return err
	}
	return nil
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
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo, ReplaceAttr: logTimeFormatter}),
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError, ReplaceAttr: logTimeFormatter}),
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
	ioc.Factory(func() events.EventBus {
		return contract.UseInMemoryEventBus(ctx)
	}, true)
	ioc.Bind(contract.UlidIdGenerator)
	ioc.Factory(func() *contract.JwtAdapter {
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
		return contract.NewJwtTokenEncoder(os.Getenv("JWT_SECRET"), lifetime, hosts, os.Getenv("HOST"))
	})
	ioc.Factory(func(j *contract.JwtAdapter) auth.TokenEncoder {
		return j
	})
	ioc.Factory(func(j *contract.JwtAdapter) auth.TokenVerifier { return j })
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
