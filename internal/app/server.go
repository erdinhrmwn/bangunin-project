package app

import (
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

	middleware.StartQueueSizeExporter(asynq.RedisClientOpt{Addr: c.Config.Redis.Addr, Password: c.Config.Redis.Password, DB: c.Config.Redis.DB}, 15*time.Second)

	app.Use(middleware.Recover(c.Logger))
	app.Get("/metrics", middleware.MetricsHandler())
	app.Use(middleware.RequestID())
	app.Use(middleware.RequestLog(c.Logger))
	app.Use(middleware.Metrics)
	app.Use(helmet.New())
	app.Use(middleware.BodyLimit)
	app.Use(middleware.CORS(nil))
	app.Use(middleware.RateLimit(c.Redis))

	route.Register(app, c.Health, c.Auth, c.User, c.Supplier, c.Media, c.AdminSupplier, c.Notification, c.Category, c.Product, c.Inventory, c.Catalog, c.Internal, c.Address, c.Cart, c.Checkout, c.Order, c.Shipment, c.Payout, c.Review, c.Wishlist, c.Banner, c.Report, c.AdminReport, c.JWT, c.AuthRepo, c.SupplierRepo, c.Redis)

	errCh := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", c.Config.App.Port)
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
		c.Logger.Info().Msg("shutting down")
	}

	return app.ShutdownWithTimeout(10 * time.Second)
}
