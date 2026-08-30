package api

import (
	"context"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func MountApiV1(r *gin.RouterGroup, c ApiConfig) {
}
func ExtractApiConfigFromEnv(ctx context.Context) (ApiConfig, error) {
	dbPool, err := pgxpool.New(ctx, os.Getenv("DB_URL"))
	if err != nil {
		return ApiConfig{}, err
	}
	if err = dbPool.Ping(ctx); err != nil {
		return ApiConfig{}, err
	}
	go func() {
		defer dbPool.Close()
		<-ctx.Done()
	}()
	config := ApiConfig{
		dbPool: dbPool,
	}
	return config, nil
}
