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
	health, auth, user, supplier, media := d.Health, d.Auth, d.User, d.Supplier, d.Media
	adminSupplier, notification, category, product, inventory := d.AdminSupplier, d.Notification, d.Category, d.Product, d.Inventory
	catalog, internal, address, cart, checkout := d.Catalog, d.Internal, d.Address, d.Cart, d.Checkout
	order, shipment, payout, review, wishlist := d.Order, d.Shipment, d.Payout, d.Review, d.Wishlist
	banner, report, adminReport := d.Banner, d.Report, d.AdminReport
	jwtSvc, authRepo, suppliers, redisClient := d.JWT, d.AuthRepo, d.Suppliers, d.Redis

	app.Get("/health", health.Check)

	// No auth middleware — protected by X-Internal-Secret header check in the handler (FR-5.6).
	app.Post("/internal/payments/callback", internal.PaymentCallback)

	api := app.Group("/api/v1")

	authGroup := api.Group("/auth", middleware.AuthRateLimit(redisClient))
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
	userGroup.Patch("/me/notification-settings", user.UpdateNotificationSettings)
	userGroup.Get("/addresses", address.List)
	userGroup.Post("/addresses", address.Create)
	userGroup.Put("/addresses/:id", address.Update)
	userGroup.Delete("/addresses/:id", address.Delete)
	userGroup.Get("/cart", cart.Get)
	userGroup.Post("/cart/items", cart.AddItem)
	userGroup.Patch("/cart/items/:id", cart.UpdateItem)
	userGroup.Delete("/cart/items/:id", cart.RemoveItem)
	userGroup.Post("/checkout/preview", checkout.Preview)
	userGroup.Post("/checkout/confirm", checkout.Confirm)
	userGroup.Get("/orders", order.ListMine)
	userGroup.Get("/orders/:id", order.GetMine)
	userGroup.Post("/orders/:id/cancel", order.Cancel)
	userGroup.Get("/orders/:id/shipment", shipment.Get)
	userGroup.Post("/orders/:order_item_id/reviews", review.Create)
	userGroup.Get("/reviews", review.ListMine)
	userGroup.Get("/wishlists", wishlist.ListMine)
	userGroup.Post("/wishlists", wishlist.Add)
	userGroup.Delete("/wishlists/:product_id", wishlist.Remove)

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
	supplierGroup.Get("/orders", approvedSupplier, order.ListSupplier)
	supplierGroup.Get("/orders/:id", approvedSupplier, order.GetSupplier)
	supplierGroup.Post("/orders/:id/process", approvedSupplier, order.Process)
	supplierGroup.Post("/orders/:id/ship", approvedSupplier, order.Ship)
	supplierGroup.Post("/orders/:id/deliver", approvedSupplier, order.Deliver)
	supplierGroup.Get("/balance", approvedSupplier, payout.Balance)
	supplierGroup.Post("/withdraws", approvedSupplier, payout.Request)
	supplierGroup.Get("/withdraws", approvedSupplier, payout.ListMine)
	supplierGroup.Get("/withdraws/:id", approvedSupplier, payout.GetMine)
	supplierGroup.Get("/reports/summary", approvedSupplier, report.Summary)
	supplierGroup.Get("/reports/export", approvedSupplier, report.Export)

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
	adminGroup.Get("/orders", order.ListAdmin)
	adminGroup.Get("/orders/:id", order.GetAdmin)
	adminGroup.Post("/orders/:id/force-status", order.ForceStatus)
	adminGroup.Get("/withdraws", payout.ListAdmin)
	adminGroup.Get("/withdraws/:id", payout.GetAdmin)
	adminGroup.Post("/withdraws/:id/approve", payout.Approve)
	adminGroup.Post("/withdraws/:id/reject", payout.Reject)
	adminGroup.Post("/banners", banner.Create)
	adminGroup.Put("/banners/:id", banner.Update)
	adminGroup.Delete("/banners/:id", banner.Delete)
	adminGroup.Get("/reports/summary", adminReport.Summary)
	adminGroup.Get("/reports/export", adminReport.Export)
	adminGroup.Get("/banners", banner.ListAdmin)

	mediaGroup := api.Group("/media", authMw)
	mediaGroup.Post("/upload", media.Upload)

	notificationGroup := api.Group("/user/notifications", authMw)
	notificationGroup.Get("/", notification.List)
	notificationGroup.Post("/:id/read", notification.MarkRead)

	optionalAuthMw := middleware.OptionalAuth(jwtSvc, authRepo)

	api.Get("/categories", category.Tree)
	api.Get("/products", catalog.Search)
	api.Get("/products/:slug", optionalAuthMw, catalog.Detail)
	api.Get("/suppliers/:slug", catalog.SupplierStore)
	api.Get("/products/:slug/reviews", review.ListByProduct)
	api.Get("/banners", banner.ListActive)
}
