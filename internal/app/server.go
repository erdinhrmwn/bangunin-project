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

	route.Register(app, route.Deps{
		Health: c.health, Auth: c.auth, User: c.user, Supplier: c.supplier, Media: c.media,
		AdminSupplier: c.adminSupplier, Notification: c.notification, Category: c.category, Product: c.product, Inventory: c.inventory,
		Catalog: c.catalog, Internal: c.internal, Address: c.address, Cart: c.cart, Checkout: c.checkout,
		Order: c.order, Shipment: c.shipment, Payout: c.payout, Review: c.review, Wishlist: c.wishlist,
		Banner: c.banner, Report: c.report, AdminReport: c.adminReport,
		JWT: c.jwt, AuthRepo: c.authRepo, Suppliers: c.supplierRepo, Redis: c.redis,
	})

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
