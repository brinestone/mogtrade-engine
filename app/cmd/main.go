package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/brinestone/mogtrade/services/orders"
	"github.com/brinestone/mogtrade/web/api"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go-slim.dev/ioc"
)

var (
	port int
)

func main() {
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

	engine := gin.Default()
	baseRouter := engine.Group("/api")
	if err := api.MountApiV1(baseRouter); err != nil {
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
