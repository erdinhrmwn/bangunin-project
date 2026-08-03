package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"erdinhrmwn/bangunin/pkg/response"
)

// HealthHandler reports app, DB, and Redis liveness.
type HealthHandler struct {
	db        *pgxpool.Pool
	redis     *redis.Client
	version   string
	startedAt time.Time
}

func NewHealthHandler(db *pgxpool.Pool, rdb *redis.Client, version string) *HealthHandler {
	return &HealthHandler{db: db, redis: rdb, version: version, startedAt: time.Now()}
}

// Check handles GET /health — 200 if db and redis both ping OK, 503 otherwise.
func (h *HealthHandler) Check(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := h.db.Ping(ctx); err != nil {
		dbStatus = "down"
	}

	redisStatus := "ok"
	if err := h.redis.Ping(ctx).Err(); err != nil {
		redisStatus = "down"
	}

	healthy := dbStatus == "ok" && redisStatus == "ok"
	data := fiber.Map{
		"db":      dbStatus,
		"redis":   redisStatus,
		"version": h.version,
		"uptime":  time.Since(h.startedAt).String(),
	}
	if healthy {
		return c.Status(fiber.StatusOK).JSON(response.Success("ok", data))
	}
	return c.Status(fiber.StatusServiceUnavailable).JSON(response.Envelope{Success: false, Message: "degraded", Data: data})
}
