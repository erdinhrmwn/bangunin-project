package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/hibiken/asynq"

	"erdinhrmwn/bangunin/internal/delivery/http/middleware"
	"erdinhrmwn/bangunin/internal/delivery/http/route"
)

// Run starts the Fiber server and blocks until SIGINT/SIGTERM, then shuts
// down gracefully (max 10s for in-flight requests) before returning.
func (c *Container) Run() error {
	app := fiber.New()

	exporterCtx, cancelExporter := context.WithCancel(context.Background())
	stopExporter := middleware.StartQueueSizeExporter(exporterCtx, asynq.RedisClientOpt{Addr: c.config.Redis.Addr, Password: c.config.Redis.Password, DB: c.config.Redis.DB}, 15*time.Second)
	defer stopExporter()
	defer cancelExporter()

	app.Use(middleware.Recover(c.logger))
	app.Get("/metrics", middleware.MetricsHandler())
	app.Use(middleware.RequestID())
	app.Use(middleware.RequestLog(c.logger))
	app.Use(middleware.Metrics)
	app.Use(helmet.New())
	app.Use(middleware.BodyLimit)
	app.Use(middleware.CORS(nil))
	app.Use(middleware.RateLimit(c.redis))

	route.Register(app, c.health, c.auth, c.user, c.supplier, c.media, c.adminSupplier, c.notification, c.category, c.product, c.inventory, c.catalog, c.internal, c.address, c.cart, c.checkout, c.order, c.shipment, c.payout, c.review, c.wishlist, c.banner, c.report, c.adminReport, c.jwt, c.authRepo, c.supplierRepo, c.redis)

	errCh := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", c.config.App.Port)
		if err := app.Listen(addr); err != nil {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-sigCh:
		c.logger.Info().Msg("shutting down")
	}

	return app.ShutdownWithTimeout(10 * time.Second)
}
