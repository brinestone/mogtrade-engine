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

	"github.com/brinestone/mogtrade/api"
	db "github.com/brinestone/mogtrade/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golobby/container/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	port int
	ioc  container.Container = container.New()
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

	engine := gin.Default()
	baseRouter := engine.Group("/api")
	if err := api.MountApiV1(baseRouter, &ioc); err != nil {
		panic(err)
	}
	engine.Run(fmt.Sprintf(":%d", port))
}

func setupLogging(ctx context.Context) error {
	if os.Getenv("LOGGING") != "enable" {
		// Even if logging is disabled, we should register a fallback logger to the DI
		ioc.Singleton(func() *slog.Logger {
			return slog.Default()
		})
		return nil
	}

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
	logger := slog.New(slog.NewMultiHandler(
		slog.NewJSONHandler(appLogsHandle, &slog.HandlerOptions{Level: slog.LevelDebug}),
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
	))

	// Register it to the DI container right here
	ioc.Singleton(func() *slog.Logger {
		return logger
	})
	return nil
}

func setupDbConnection(ctx context.Context) error {
	ioc.Singleton(func() (*pgxpool.Pool, error) {
		pool, err := pgxpool.New(ctx, os.Getenv("DB_URL"))
		if err != nil {
			return nil, err
		}
		if err = pool.Ping(ctx); err != nil {
			return nil, err
		}
		return pool, nil
	})
	ioc.Singleton(func(pool *pgxpool.Pool) *db.Queries {
		return db.New(pool)
	})
	ioc.TransientLazy(func(pool *pgxpool.Pool) (*pgxpool.Conn, error) {
		return pool.Acquire(ctx)
	})

	// Note: The slog.Logger registration was safely moved into setupLogging!
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
