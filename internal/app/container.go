// Package app wires dependencies and runs the HTTP server.
package app

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"erdinhrmwn/bangunin/config"
	"erdinhrmwn/bangunin/internal/delivery/http/handler"
	"erdinhrmwn/bangunin/internal/infra/database"
	infraredis "erdinhrmwn/bangunin/internal/infra/redis"
	"erdinhrmwn/bangunin/pkg/logger"
)

// Container holds process-wide dependencies, built once at startup.
type Container struct {
	Config *config.Config
	Logger zerolog.Logger
	DB     *pgxpool.Pool
	Redis  *redis.Client

	Health *handler.HealthHandler
}

// NewContainer connects to Postgres/Redis and builds all handlers. Callers
// must call Close when done.
func NewContainer(ctx context.Context, cfg *config.Config) (*Container, error) {
	log := logger.New(cfg.App.Env, "info", cfg.App.Name)

	db, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		return nil, err
	}

	rdb, err := infraredis.NewClient(ctx, cfg.Redis)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Container{
		Config: cfg,
		Logger: log,
		DB:     db,
		Redis:  rdb,
		Health: handler.NewHealthHandler(db, rdb, "1.0.0"),
	}, nil
}

// Close releases DB and Redis connections.
func (c *Container) Close() {
	c.DB.Close()
	_ = c.Redis.Close()
}
