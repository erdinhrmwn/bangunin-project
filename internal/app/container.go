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
	"erdinhrmwn/bangunin/internal/infra/grpcclient"
	"erdinhrmwn/bangunin/internal/infra/queue"
	"erdinhrmwn/bangunin/internal/infra/rajaongkir"
	"erdinhrmwn/bangunin/internal/infra/storage"
	"erdinhrmwn/bangunin/pkg/jwt"
	"erdinhrmwn/bangunin/pkg/logger"

	infraredis "erdinhrmwn/bangunin/internal/infra/redis"
	postgresrepo "erdinhrmwn/bangunin/internal/repository/postgres"
	redisrepo "erdinhrmwn/bangunin/internal/repository/redis"
	adminsupplierusecase "erdinhrmwn/bangunin/internal/usecase/adminsupplier"
	authusecase "erdinhrmwn/bangunin/internal/usecase/auth"
	bannerusecase "erdinhrmwn/bangunin/internal/usecase/banner"
	cartusecase "erdinhrmwn/bangunin/internal/usecase/cart"
	catalogusecase "erdinhrmwn/bangunin/internal/usecase/catalog"
	categoryusecase "erdinhrmwn/bangunin/internal/usecase/category"
	checkoutusecase "erdinhrmwn/bangunin/internal/usecase/checkout"
	inventoryusecase "erdinhrmwn/bangunin/internal/usecase/inventory"
	mediausecase "erdinhrmwn/bangunin/internal/usecase/media"
	notificationusecase "erdinhrmwn/bangunin/internal/usecase/notification"
	orderusecase "erdinhrmwn/bangunin/internal/usecase/order"
	payoutusecase "erdinhrmwn/bangunin/internal/usecase/payout"
	productusecase "erdinhrmwn/bangunin/internal/usecase/product"
	reportusecase "erdinhrmwn/bangunin/internal/usecase/report"
	reviewusecase "erdinhrmwn/bangunin/internal/usecase/review"
	supplierusecase "erdinhrmwn/bangunin/internal/usecase/supplier"
	userusecase "erdinhrmwn/bangunin/internal/usecase/user"
	wishlistusecase "erdinhrmwn/bangunin/internal/usecase/wishlist"
)

// Container holds process-wide dependencies, built once at startup.
type Container struct {
	Config *config.Config
	Logger zerolog.Logger
	DB     *pgxpool.Pool
	Redis  *redis.Client

	JWT          *jwt.Service
	AuthRepo     repository.AuthRepository
	SupplierRepo repository.SupplierRepository
	Enqueuer     *queue.Enqueuer

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
	supplierRepo := postgresrepo.NewSupplierRepository(db)
	supplierDocRepo := postgresrepo.NewSupplierDocumentRepository(db)
	supplierBankRepo := postgresrepo.NewSupplierBankAccountRepository(db)
	notificationRepo := postgresrepo.NewNotificationRepository(db)
	auditLogRepo := postgresrepo.NewAuditLogRepository(db)
	categoryRepo := postgresrepo.NewCategoryRepository(db)
	productRepo := postgresrepo.NewProductRepository(db)
	productVariantRepo := postgresrepo.NewProductVariantRepository(db)
	productImageRepo := postgresrepo.NewProductImageRepository(db)
	stockMovementRepo := postgresrepo.NewStockMovementRepository(db)
	orderRepo := postgresrepo.NewOrderRepository(db)
	orderHistoryRepo := postgresrepo.NewOrderStatusHistoryRepository(db)
	shipmentRepo := postgresrepo.NewShipmentRepository(db)
	checkoutGroupRepo := postgresrepo.NewCheckoutGroupRepository(db)
	paymentRepo := postgresrepo.NewPaymentRepository(db)
	stockReservationRepo := postgresrepo.NewStockReservationRepository(db)
	userAddressRepo := postgresrepo.NewUserAddressRepository(db)
	cartRepo := postgresrepo.NewCartRepository(db)
	withdrawRepo := postgresrepo.NewWithdrawRequestRepository(db)
	reviewRepo := postgresrepo.NewReviewRepository(db)
	wishlistRepo := postgresrepo.NewWishlistRepository(db)
	bannerRepo := postgresrepo.NewBannerRepository(db)
	jwtSvc := jwt.NewService(cfg.JWT.Secret, cfg.JWT.AccessTTL)
	enqueuer := queue.NewEnqueuer(asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	stockLock := redisrepo.NewStockLock(rdb)
	shippingGW := rajaongkir.New(cfg.RajaOngkir)
	paymentClient, err := grpcclient.NewPaymentClient(cfg.Payment.GRPCAddr)
	if err != nil {
		db.Close()
		_ = rdb.Close()
		return nil, err
	}

	mediaStorage, err := storage.New(ctx, cfg.R2)
	if err != nil {
		db.Close()
		_ = rdb.Close()
		return nil, err
	}

	authUC := authusecase.New(userRepo, authRepo, enqueuer, jwtSvc, cfg.JWT.RefreshTTL)
	userUC := userusecase.New(userRepo, userAddressRepo)
	supplierUC := supplierusecase.New(supplierRepo, supplierDocRepo, supplierBankRepo)
	mediaUC := mediausecase.New(mediaStorage, enqueuer)
	adminSupplierUC := adminsupplierusecase.New(supplierRepo, supplierDocRepo, userRepo, auditLogRepo, notificationRepo, enqueuer)
	notificationUC := notificationusecase.New(notificationRepo)
	categoryUC := categoryusecase.New(categoryRepo, rdb)
	productUC := productusecase.New(productRepo, productVariantRepo, productImageRepo, rdb)
	inventoryUC := inventoryusecase.New(productVariantRepo, stockMovementRepo, supplierRepo, notificationRepo, rdb, cfg.Catalog.LowStockThreshold)
	catalogUC := catalogusecase.New(productRepo, supplierRepo, rdb)
	ledgerRepo := postgresrepo.NewLedgerEntryRepository(db)
	supplierBalanceRepo := postgresrepo.NewSupplierBalanceRepository(db)
	orderUC := orderusecase.New(
		orderRepo, orderHistoryRepo, shipmentRepo, checkoutGroupRepo, paymentRepo, stockReservationRepo,
		supplierRepo, userRepo, notificationRepo, auditLogRepo, enqueuer,
		ledgerRepo, supplierBalanceRepo,
	)
	cartUC := cartusecase.New(cartRepo, productVariantRepo, productRepo)
	payoutUC := payoutusecase.New(
		withdrawRepo, supplierBalanceRepo, supplierBankRepo, supplierRepo, userRepo,
		auditLogRepo, notificationRepo, enqueuer, ledgerRepo, paymentClient,
	)
	reviewUC := reviewusecase.New(reviewRepo, orderRepo, productVariantRepo, productRepo)
	wishlistUC := wishlistusecase.New(wishlistRepo, productRepo, rdb)
	bannerUC := bannerusecase.New(bannerRepo, rdb)
	reportUC := reportusecase.New(orderRepo, enqueuer)
	adminReportUC := reportusecase.NewAdmin(orderRepo, ledgerRepo, supplierRepo, userRepo, enqueuer)
	checkoutUC := checkoutusecase.New(
		checkoutGroupRepo, paymentRepo, stockReservationRepo, userAddressRepo, cartRepo,
		productVariantRepo, productRepo, supplierRepo, shippingGW, paymentClient, stockLock,
	)

	return &Container{
		Config:        cfg,
		Logger:        log,
		DB:            db,
		Redis:         rdb,
		JWT:           jwtSvc,
		AuthRepo:      authRepo,
		SupplierRepo:  supplierRepo,
		Enqueuer:      enqueuer,
		Health:        handler.NewHealthHandler(db, rdb, "1.0.0"),
		Auth:          handler.NewAuthHandler(authUC, jwtSvc),
		User:          handler.NewUserHandler(userUC),
		Supplier:      handler.NewSupplierHandler(supplierUC),
		Media:         handler.NewMediaHandler(mediaUC),
		AdminSupplier: handler.NewAdminSupplierHandler(adminSupplierUC),
		Notification:  handler.NewNotificationHandler(notificationUC),
		Category:      handler.NewCategoryHandler(categoryUC),
		Product:       handler.NewProductHandler(productUC, supplierRepo),
		Inventory:     handler.NewInventoryHandler(inventoryUC, supplierRepo),
		Catalog:       handler.NewCatalogHandler(catalogUC, wishlistUC),
		Internal:      handler.NewInternalHandler(cfg.Payment.InternalSecret, cfg.Payment.InternalIPAllowlist, orderUC),
		Address:       handler.NewAddressHandler(userUC),
		Cart:          handler.NewCartHandler(cartUC),
		Checkout:      handler.NewCheckoutHandler(checkoutUC),
		Order:         handler.NewOrderHandler(orderUC, supplierRepo),
		Shipment:      handler.NewShipmentHandler(orderUC),
		Payout:        handler.NewPayoutHandler(payoutUC, supplierRepo),
		Review:        handler.NewReviewHandler(reviewUC, productRepo),
		Wishlist:      handler.NewWishlistHandler(wishlistUC),
		Banner:        handler.NewBannerHandler(bannerUC),
		Report:        handler.NewReportHandler(reportUC, supplierRepo),
		AdminReport:   handler.NewAdminReportHandler(adminReportUC),
	}, nil
}

// Close releases DB, Redis, and queue connections.
func (c *Container) Close() {
	c.DB.Close()
	_ = c.Redis.Close()
	_ = c.Enqueuer.Close()
}
