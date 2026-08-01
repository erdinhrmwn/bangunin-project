package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	"erdinhrmwn/bangunin/internal/delivery/http/middleware"
	"erdinhrmwn/bangunin/internal/delivery/http/route"
)

// Run starts the Fiber server and blocks until SIGINT/SIGTERM, then shuts
// down gracefully (max 10s for in-flight requests) before returning.
func (c *Container) Run() error {
	app := fiber.New()

	app.Use(middleware.Recover(c.Logger))
	app.Use(middleware.RequestID())
	app.Use(middleware.RequestLog(c.Logger))
	app.Use(middleware.CORS(nil))
	app.Use(middleware.RateLimit(c.Redis))

	route.Register(app, c.Health, c.Auth, c.User, c.Supplier, c.Media, c.AdminSupplier, c.Notification, c.JWT, c.AuthRepo, c.SupplierRepo)

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
