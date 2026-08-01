// Package app wires dependencies and runs the HTTP server.
package app

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"erdinhrmwn/bangunin/config"
	"erdinhrmwn/bangunin/internal/delivery/http/handler"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/infra/database"
	"erdinhrmwn/bangunin/internal/infra/queue"
	postgresrepo "erdinhrmwn/bangunin/internal/repository/postgres"
	redisrepo "erdinhrmwn/bangunin/internal/repository/redis"
	authusecase "erdinhrmwn/bangunin/internal/usecase/auth"
	userusecase "erdinhrmwn/bangunin/internal/usecase/user"
	"erdinhrmwn/bangunin/pkg/jwt"
	"erdinhrmwn/bangunin/pkg/logger"

	infraredis "erdinhrmwn/bangunin/internal/infra/redis"
)

// Container holds process-wide dependencies, built once at startup.
type Container struct {
	Config *config.Config
	Logger zerolog.Logger
	DB     *pgxpool.Pool
	Redis  *redis.Client

	JWT      *jwt.Service
	AuthRepo repository.AuthRepository
	Enqueuer *queue.Enqueuer

	Health *handler.HealthHandler
	Auth   *handler.AuthHandler
	User   *handler.UserHandler
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

	userRepo := postgresrepo.NewUserRepository(db)
	authRepo := redisrepo.NewAuthRepository(rdb)
	jwtSvc := jwt.NewService(cfg.JWT.Secret, cfg.JWT.AccessTTL)
	enqueuer := queue.NewEnqueuer(asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})

	authUC := authusecase.New(userRepo, authRepo, enqueuer, jwtSvc, cfg.JWT.RefreshTTL)
	userUC := userusecase.New(userRepo)

	return &Container{
		Config:   cfg,
		Logger:   log,
		DB:       db,
		Redis:    rdb,
		JWT:      jwtSvc,
		AuthRepo: authRepo,
		Enqueuer: enqueuer,
		Health:   handler.NewHealthHandler(db, rdb, "1.0.0"),
		Auth:     handler.NewAuthHandler(authUC, jwtSvc),
		User:     handler.NewUserHandler(userUC),
	}, nil
}

// Close releases DB, Redis, and queue connections.
func (c *Container) Close() {
	c.DB.Close()
	_ = c.Redis.Close()
	_ = c.Enqueuer.Close()
}
