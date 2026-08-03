// Package route wires HTTP routes to handlers.
package route

import (
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"erdinhrmwn/bangunin/internal/delivery/http/handler"
	"erdinhrmwn/bangunin/internal/delivery/http/middleware"
	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/pkg/jwt"
)

// Deps groups the handlers and shared dependencies route registration needs,
// replacing a long positional Register(...) parameter list.
type Deps struct {
	Health        *handler.HealthHandler
	Auth          *handler.AuthHandler
	User          *handler.UserHandler
	Supplier      *handler.SupplierHandler
	Media         *handler.MediaHandler
	AdminSupplier *handler.AdminSupplierHandler
	Notification  *handler.NotificationHandler
	Category      *handler.CategoryHandler
	Product       *handler.ProductHandler
	Inventory     *handler.InventoryHandler
	Catalog       *handler.CatalogHandler
	Internal      *handler.InternalHandler
	Address       *handler.AddressHandler
	Cart          *handler.CartHandler
	Checkout      *handler.CheckoutHandler
	Order         *handler.OrderHandler
	Shipment      *handler.ShipmentHandler
	Payout        *handler.PayoutHandler
	Review        *handler.ReviewHandler
	Wishlist      *handler.WishlistHandler
	Banner        *handler.BannerHandler
	Report        *handler.ReportHandler
	AdminReport   *handler.AdminReportHandler
	JWT           *jwt.Service
	AuthRepo      repository.AuthRepository
	Suppliers     repository.SupplierRepository
	Redis         *redis.Client
}

// Register mounts all routes on app.
func Register(app *fiber.App, d Deps) {
	app.Get("/health", d.Health.Check)

	// No auth middleware — protected by X-Internal-Secret header check in the handler (FR-5.6).
	app.Post("/internal/payments/callback", d.Internal.PaymentCallback)

	api := app.Group("/api/v1")

	authGroup := api.Group("/auth", middleware.AuthRateLimit(d.Redis))
	authGroup.Post("/register", d.Auth.Register)
	authGroup.Post("/verify-email", d.Auth.VerifyEmail)
	authGroup.Post("/resend-otp", d.Auth.ResendOTP)
	authGroup.Post("/login", d.Auth.Login)
	authGroup.Post("/refresh", d.Auth.Refresh)
	authGroup.Post("/logout", d.Auth.Logout)
	authGroup.Post("/forgot-password", d.Auth.ForgotPassword)
	authGroup.Post("/reset-password", d.Auth.ResetPassword)

	authMw := middleware.Auth(d.JWT, d.AuthRepo)

	userGroup := api.Group("/user", authMw, middleware.RequireRole(entity.RoleUser))
	userGroup.Get("/me", d.User.Me)
	userGroup.Patch("/me", d.User.UpdateMe)
	userGroup.Patch("/me/password", d.User.UpdatePassword)
	userGroup.Patch("/me/notification-settings", d.User.UpdateNotificationSettings)
	userGroup.Get("/addresses", d.Address.List)
	userGroup.Post("/addresses", d.Address.Create)
	userGroup.Put("/addresses/:id", d.Address.Update)
	userGroup.Delete("/addresses/:id", d.Address.Delete)
	userGroup.Get("/cart", d.Cart.Get)
	userGroup.Post("/cart/items", d.Cart.AddItem)
	userGroup.Patch("/cart/items/:id", d.Cart.UpdateItem)
	userGroup.Delete("/cart/items/:id", d.Cart.RemoveItem)
	userGroup.Post("/checkout/preview", d.Checkout.Preview)
	userGroup.Post("/checkout/confirm", d.Checkout.Confirm)
	userGroup.Get("/orders", d.Order.ListMine)
	userGroup.Get("/orders/:id", d.Order.GetMine)
	userGroup.Post("/orders/:id/cancel", d.Order.Cancel)
	userGroup.Get("/orders/:id/shipment", d.Shipment.Get)
	userGroup.Post("/orders/:order_item_id/reviews", d.Review.Create)
	userGroup.Get("/reviews", d.Review.ListMine)
	userGroup.Get("/wishlists", d.Wishlist.ListMine)
	userGroup.Post("/wishlists", d.Wishlist.Add)
	userGroup.Delete("/wishlists/:product_id", d.Wishlist.Remove)

	supplierGroup := api.Group("/supplier", authMw, middleware.RequireRole(entity.RoleSupplier))
	supplierGroup.Get("/me", d.User.Me)
	supplierGroup.Post("/profile", d.Supplier.CreateProfile)
	supplierGroup.Put("/profile", d.Supplier.UpdateProfile)
	supplierGroup.Get("/profile", d.Supplier.Profile)
	supplierGroup.Post("/documents", d.Supplier.UploadDocument)
	supplierGroup.Get("/documents", d.Supplier.ListDocuments)
	supplierGroup.Post("/bank-accounts", d.Supplier.CreateBankAccount)
	supplierGroup.Get("/bank-accounts", d.Supplier.ListBankAccounts)
	supplierGroup.Put("/bank-accounts/:id", d.Supplier.UpdateBankAccount)
	supplierGroup.Delete("/bank-accounts/:id", d.Supplier.DeleteBankAccount)
	supplierGroup.Post("/submit", d.Supplier.Submit)
	supplierGroup.Get("/dashboard", middleware.RequireApprovedSupplier(d.Suppliers), d.Supplier.Profile)

	approvedSupplier := middleware.RequireApprovedSupplier(d.Suppliers)

	supplierGroup.Post("/products", approvedSupplier, d.Product.Create)
	supplierGroup.Get("/products", approvedSupplier, d.Product.List)
	supplierGroup.Get("/products/:id", approvedSupplier, d.Product.Get)
	supplierGroup.Put("/products/:id", approvedSupplier, d.Product.Update)
	supplierGroup.Post("/products/:id/publish", approvedSupplier, d.Product.Publish)
	supplierGroup.Post("/products/:id/variants", approvedSupplier, d.Product.CreateVariant)
	supplierGroup.Get("/products/:id/variants", approvedSupplier, d.Product.ListVariants)
	supplierGroup.Put("/products/:id/variants/:variantId", approvedSupplier, d.Product.UpdateVariant)
	supplierGroup.Post("/products/:id/images", approvedSupplier, d.Product.AttachImage)
	supplierGroup.Put("/products/:id/images/:imageId", approvedSupplier, d.Product.SetPrimaryImage)
	supplierGroup.Delete("/products/:id/images/:imageId", approvedSupplier, d.Product.DeleteImage)
	supplierGroup.Post("/variants/:id/stock-adjustment", approvedSupplier, d.Inventory.AdjustStock)
	supplierGroup.Get("/variants/:id/movements", approvedSupplier, d.Inventory.ListMovements)
	supplierGroup.Get("/orders", approvedSupplier, d.Order.ListSupplier)
	supplierGroup.Get("/orders/:id", approvedSupplier, d.Order.GetSupplier)
	supplierGroup.Post("/orders/:id/process", approvedSupplier, d.Order.Process)
	supplierGroup.Post("/orders/:id/ship", approvedSupplier, d.Order.Ship)
	supplierGroup.Post("/orders/:id/deliver", approvedSupplier, d.Order.Deliver)
	supplierGroup.Get("/balance", approvedSupplier, d.Payout.Balance)
	supplierGroup.Post("/withdraws", approvedSupplier, d.Payout.Request)
	supplierGroup.Get("/withdraws", approvedSupplier, d.Payout.ListMine)
	supplierGroup.Get("/withdraws/:id", approvedSupplier, d.Payout.GetMine)
	supplierGroup.Get("/reports/summary", approvedSupplier, d.Report.Summary)
	supplierGroup.Get("/reports/export", approvedSupplier, d.Report.Export)

	adminGroup := api.Group("/admin", authMw, middleware.RequireRole(entity.RoleAdmin))
	adminGroup.Get("/me", d.User.Me)
	adminGroup.Get("/suppliers", d.AdminSupplier.List)
	adminGroup.Get("/suppliers/:id", d.AdminSupplier.Get)
	adminGroup.Post("/suppliers/:id/approve", d.AdminSupplier.Approve)
	adminGroup.Post("/suppliers/:id/reject", d.AdminSupplier.Reject)
	adminGroup.Post("/suppliers/:id/suspend", d.AdminSupplier.Suspend)
	adminGroup.Get("/audit-logs", d.AdminSupplier.AuditLogs)
	adminGroup.Post("/categories", d.Category.Create)
	adminGroup.Put("/categories/:id", d.Category.Update)
	adminGroup.Delete("/categories/:id", d.Category.Delete)
	adminGroup.Get("/orders", d.Order.ListAdmin)
	adminGroup.Get("/orders/:id", d.Order.GetAdmin)
	adminGroup.Post("/orders/:id/force-status", d.Order.ForceStatus)
	adminGroup.Get("/withdraws", d.Payout.ListAdmin)
	adminGroup.Get("/withdraws/:id", d.Payout.GetAdmin)
	adminGroup.Post("/withdraws/:id/approve", d.Payout.Approve)
	adminGroup.Post("/withdraws/:id/reject", d.Payout.Reject)
	adminGroup.Post("/banners", d.Banner.Create)
	adminGroup.Put("/banners/:id", d.Banner.Update)
	adminGroup.Delete("/banners/:id", d.Banner.Delete)
	adminGroup.Get("/reports/summary", d.AdminReport.Summary)
	adminGroup.Get("/reports/export", d.AdminReport.Export)
	adminGroup.Get("/banners", d.Banner.ListAdmin)

	mediaGroup := api.Group("/media", authMw)
	mediaGroup.Post("/upload", d.Media.Upload)

	notificationGroup := api.Group("/user/notifications", authMw)
	notificationGroup.Get("/", d.Notification.List)
	notificationGroup.Post("/:id/read", d.Notification.MarkRead)

	optionalAuthMw := middleware.OptionalAuth(d.JWT, d.AuthRepo)

	api.Get("/categories", d.Category.Tree)
	api.Get("/products", d.Catalog.Search)
	api.Get("/products/:slug", optionalAuthMw, d.Catalog.Detail)
	api.Get("/suppliers/:slug", d.Catalog.SupplierStore)
	api.Get("/products/:slug/reviews", d.Review.ListByProduct)
	api.Get("/banners", d.Banner.ListActive)
}
