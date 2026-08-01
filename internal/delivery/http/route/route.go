// Package route wires HTTP routes to handlers.
package route

import (
	"github.com/gofiber/fiber/v3"

	"erdinhrmwn/bangunin/internal/delivery/http/handler"
	"erdinhrmwn/bangunin/internal/delivery/http/middleware"
	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/pkg/jwt"
)

// Register mounts all routes on app.
func Register(app *fiber.App, health *handler.HealthHandler, auth *handler.AuthHandler, user *handler.UserHandler, supplier *handler.SupplierHandler, media *handler.MediaHandler, jwtSvc *jwt.Service, authRepo repository.AuthRepository) {
	app.Get("/health", health.Check)

	api := app.Group("/api/v1")

	authGroup := api.Group("/auth")
	authGroup.Post("/register", auth.Register)
	authGroup.Post("/verify-email", auth.VerifyEmail)
	authGroup.Post("/resend-otp", auth.ResendOTP)
	authGroup.Post("/login", auth.Login)
	authGroup.Post("/refresh", auth.Refresh)
	authGroup.Post("/logout", auth.Logout)
	authGroup.Post("/forgot-password", auth.ForgotPassword)
	authGroup.Post("/reset-password", auth.ResetPassword)

	authMw := middleware.Auth(jwtSvc, authRepo)

	userGroup := api.Group("/user", authMw, middleware.RequireRole(entity.RoleUser))
	userGroup.Get("/me", user.Me)
	userGroup.Patch("/me", user.UpdateMe)
	userGroup.Patch("/me/password", user.UpdatePassword)

	supplierGroup := api.Group("/supplier", authMw, middleware.RequireRole(entity.RoleSupplier))
	supplierGroup.Get("/me", user.Me)
	supplierGroup.Post("/profile", supplier.CreateProfile)
	supplierGroup.Put("/profile", supplier.UpdateProfile)
	supplierGroup.Get("/profile", supplier.Profile)
	supplierGroup.Post("/documents", supplier.UploadDocument)
	supplierGroup.Get("/documents", supplier.ListDocuments)
	supplierGroup.Post("/bank-accounts", supplier.CreateBankAccount)
	supplierGroup.Get("/bank-accounts", supplier.ListBankAccounts)
	supplierGroup.Put("/bank-accounts/:id", supplier.UpdateBankAccount)
	supplierGroup.Delete("/bank-accounts/:id", supplier.DeleteBankAccount)
	supplierGroup.Post("/submit", supplier.Submit)

	adminGroup := api.Group("/admin", authMw, middleware.RequireRole(entity.RoleAdmin))
	adminGroup.Get("/me", user.Me)

	mediaGroup := api.Group("/media", authMw)
	mediaGroup.Post("/upload", media.Upload)
}
