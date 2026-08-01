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
func Register(app *fiber.App, health *handler.HealthHandler, auth *handler.AuthHandler, user *handler.UserHandler, supplier *handler.SupplierHandler, media *handler.MediaHandler, adminSupplier *handler.AdminSupplierHandler, notification *handler.NotificationHandler, category *handler.CategoryHandler, product *handler.ProductHandler, inventory *handler.InventoryHandler, catalog *handler.CatalogHandler, jwtSvc *jwt.Service, authRepo repository.AuthRepository, suppliers repository.SupplierRepository) {
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
	supplierGroup.Get("/dashboard", middleware.RequireApprovedSupplier(suppliers), supplier.Profile)

	approvedSupplier := middleware.RequireApprovedSupplier(suppliers)

	supplierGroup.Post("/products", approvedSupplier, product.Create)
	supplierGroup.Get("/products", approvedSupplier, product.List)
	supplierGroup.Get("/products/:id", approvedSupplier, product.Get)
	supplierGroup.Put("/products/:id", approvedSupplier, product.Update)
	supplierGroup.Post("/products/:id/publish", approvedSupplier, product.Publish)
	supplierGroup.Post("/products/:id/variants", approvedSupplier, product.CreateVariant)
	supplierGroup.Get("/products/:id/variants", approvedSupplier, product.ListVariants)
	supplierGroup.Put("/products/:id/variants/:variantId", approvedSupplier, product.UpdateVariant)
	supplierGroup.Post("/products/:id/images", approvedSupplier, product.AttachImage)
	supplierGroup.Put("/products/:id/images/:imageId", approvedSupplier, product.SetPrimaryImage)
	supplierGroup.Delete("/products/:id/images/:imageId", approvedSupplier, product.DeleteImage)
	supplierGroup.Post("/variants/:id/stock-adjustment", approvedSupplier, inventory.AdjustStock)
	supplierGroup.Get("/variants/:id/movements", approvedSupplier, inventory.ListMovements)

	adminGroup := api.Group("/admin", authMw, middleware.RequireRole(entity.RoleAdmin))
	adminGroup.Get("/me", user.Me)
	adminGroup.Get("/suppliers", adminSupplier.List)
	adminGroup.Get("/suppliers/:id", adminSupplier.Get)
	adminGroup.Post("/suppliers/:id/approve", adminSupplier.Approve)
	adminGroup.Post("/suppliers/:id/reject", adminSupplier.Reject)
	adminGroup.Post("/suppliers/:id/suspend", adminSupplier.Suspend)
	adminGroup.Get("/audit-logs", adminSupplier.AuditLogs)
	adminGroup.Post("/categories", category.Create)
	adminGroup.Put("/categories/:id", category.Update)
	adminGroup.Delete("/categories/:id", category.Delete)

	mediaGroup := api.Group("/media", authMw)
	mediaGroup.Post("/upload", media.Upload)

	notificationGroup := api.Group("/user/notifications", authMw)
	notificationGroup.Get("/", notification.List)
	notificationGroup.Post("/:id/read", notification.MarkRead)

	api.Get("/categories", category.Tree)
	api.Get("/products", catalog.Search)
	api.Get("/products/:slug", catalog.Detail)
	api.Get("/suppliers/:slug", catalog.SupplierStore)
}
