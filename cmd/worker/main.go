// Command worker runs the Asynq background job server.
package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"erdinhrmwn/bangunin/config"
	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/domain/service"
	"erdinhrmwn/bangunin/internal/infra/database"
	"erdinhrmwn/bangunin/internal/infra/grpcclient"
	"erdinhrmwn/bangunin/internal/infra/queue"
	"erdinhrmwn/bangunin/internal/infra/storage"
	"erdinhrmwn/bangunin/pkg/imageresize"
	"erdinhrmwn/bangunin/pkg/logger"

	infraredis "erdinhrmwn/bangunin/internal/infra/redis"
	postgresrepo "erdinhrmwn/bangunin/internal/repository/postgres"
	inventoryusecase "erdinhrmwn/bangunin/internal/usecase/inventory"
	orderusecase "erdinhrmwn/bangunin/internal/usecase/order"
	reportusecase "erdinhrmwn/bangunin/internal/usecase/report"
)

// deliverReport uploads a generated report CSV, presigns a 24h download
// link, records an in-app notification, and emails the recipient — the
// tail shared by report:generate and report:generate-admin.
func deliverReport(ctx context.Context, mediaStorage *storage.MinIOStorage, notificationRepo repository.NotificationRepository, notifier service.NotificationService, key string, data []byte, recipient *entity.User, reportKind string) error {
	if _, err := mediaStorage.Upload(ctx, key, bytes.NewReader(data), int64(len(data)), "text/csv"); err != nil {
		return fmt.Errorf("upload %s: %w", key, err)
	}
	downloadURL, err := mediaStorage.PresignedURL(ctx, key, 24*time.Hour)
	if err != nil {
		return fmt.Errorf("presign %s: %w", key, err)
	}

	title := reportKind + " report ready"
	body := fmt.Sprintf("Your %s report is ready. Download it here (valid 24h): %s", strings.ToLower(reportKind), downloadURL)
	notifID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate notification id: %w", err)
	}
	if err := notificationRepo.Create(ctx, &entity.Notification{
		ID: notifID, UserID: recipient.ID, Type: entity.NotificationTypeSystem, Title: title, Body: body,
	}); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return notifier.SendEmail(ctx, recipient.Email, title, body)
}

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		fatal(err)
	}
	log := logger.New(cfg.App.Env, "info", cfg.App.Name+"-worker")

	redisOpt := asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB}

	srv := asynq.NewServer(redisOpt, asynq.Config{})
	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TaskHeartbeat, func(ctx context.Context, t *asynq.Task) error {
		log.Info().Msg("heartbeat")
		return nil
	})

	notifier, err := grpcclient.NewNotificationClient(cfg.Notification.GRPCAddr)
	if err != nil {
		fatal(err)
	}
	mux.HandleFunc(queue.TaskEmailSend, func(ctx context.Context, t *asynq.Task) error {
		var p queue.EmailSendPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("email:send: unmarshal payload: %w", err)
		}
		return notifier.SendEmail(ctx, p.To, p.Subject, p.Body)
	})

	mediaStorage, err := storage.New(context.Background(), cfg.R2)
	if err != nil {
		fatal(err)
	}
	mux.HandleFunc(queue.TaskMediaProcess, func(ctx context.Context, t *asynq.Task) error {
		var p queue.MediaProcessPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("media:process: unmarshal payload: %w", err)
		}
		obj, err := mediaStorage.Download(ctx, p.Key)
		if err != nil {
			return fmt.Errorf("media:process: download %s: %w", p.Key, err)
		}
		data, contentType, err := imageresize.Resize(obj)
		_ = obj.Close()
		if err != nil {
			return fmt.Errorf("media:process: resize %s: %w", p.Key, err)
		}
		if _, err := mediaStorage.Upload(ctx, p.Key, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
			return fmt.Errorf("media:process: re-upload %s: %w", p.Key, err)
		}
		return nil
	})

	db, err := database.NewPool(context.Background(), cfg.DB)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	rdb, err := infraredis.NewClient(context.Background(), cfg.Redis)
	if err != nil {
		fatal(err)
	}
	defer func() { _ = rdb.Close() }()

	inventoryUC := inventoryusecase.New(
		postgresrepo.NewProductVariantRepository(db),
		postgresrepo.NewStockMovementRepository(db),
		postgresrepo.NewSupplierRepository(db),
		postgresrepo.NewNotificationRepository(db),
		rdb,
		cfg.Catalog.LowStockThreshold,
	)
	mux.HandleFunc(queue.TaskLowStockCheck, func(ctx context.Context, t *asynq.Task) error {
		return inventoryUC.CheckLowStock(ctx)
	})

	enqueuer := queue.NewEnqueuer(redisOpt)
	defer func() { _ = enqueuer.Close() }()
	orderUC := orderusecase.New(
		postgresrepo.NewOrderRepository(db),
		postgresrepo.NewOrderStatusHistoryRepository(db),
		postgresrepo.NewShipmentRepository(db),
		postgresrepo.NewCheckoutGroupRepository(db),
		postgresrepo.NewPaymentRepository(db),
		postgresrepo.NewStockReservationRepository(db),
		postgresrepo.NewProductVariantRepository(db),
		postgresrepo.NewSupplierRepository(db),
		postgresrepo.NewUserRepository(db),
		postgresrepo.NewNotificationRepository(db),
		postgresrepo.NewAuditLogRepository(db),
		enqueuer,
		postgresrepo.NewLedgerEntryRepository(db),
		postgresrepo.NewSupplierBalanceRepository(db),
	)
	mux.HandleFunc(queue.TaskOrderExpire, func(ctx context.Context, t *asynq.Task) error {
		return orderUC.HandleExpire(ctx)
	})
	mux.HandleFunc(queue.TaskOrderAutocomplete, func(ctx context.Context, t *asynq.Task) error {
		return orderUC.HandleAutocomplete(ctx)
	})

	supplierRepo := postgresrepo.NewSupplierRepository(db)
	userRepo := postgresrepo.NewUserRepository(db)
	notificationRepo := postgresrepo.NewNotificationRepository(db)
	reportUC := reportusecase.New(postgresrepo.NewOrderRepository(db), enqueuer)
	mux.HandleFunc(queue.TaskReportGenerate, func(ctx context.Context, t *asynq.Task) error {
		var p queue.ReportGeneratePayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("report:generate: unmarshal payload: %w", err)
		}
		summary, err := reportUC.GetSummary(ctx, p.SupplierID, p.From, p.To)
		if err != nil {
			return fmt.Errorf("report:generate: get summary: %w", err)
		}

		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		_ = w.Write([]string{"GMV", "Orders", "AOV"})
		_ = w.Write([]string{fmt.Sprintf("%.2f", summary.GMV), fmt.Sprintf("%d", summary.OrdersCount), fmt.Sprintf("%.2f", summary.AOV)})
		_ = w.Write([]string{})
		_ = w.Write([]string{"Product", "Qty", "Revenue"})
		for _, tp := range summary.TopProducts {
			_ = w.Write([]string{tp.ProductName, fmt.Sprintf("%d", tp.Qty), fmt.Sprintf("%.2f", tp.Revenue)})
		}
		_ = w.Write([]string{})
		_ = w.Write([]string{"Day", "GMV", "Orders"})
		for _, d := range summary.SalesPerDay {
			_ = w.Write([]string{d.Day.Format("2006-01-02"), fmt.Sprintf("%.2f", d.GMV), fmt.Sprintf("%d", d.Orders)})
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return fmt.Errorf("report:generate: write csv: %w", err)
		}
		data := buf.Bytes()

		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("report:generate: generate id: %w", err)
		}
		s, err := supplierRepo.FindByID(ctx, p.SupplierID)
		if err != nil {
			return fmt.Errorf("report:generate: find supplier: %w", err)
		}
		usr, err := userRepo.FindByID(ctx, s.UserID)
		if err != nil {
			return fmt.Errorf("report:generate: find user: %w", err)
		}
		key := fmt.Sprintf("reports/%s/%s.csv", p.SupplierID, id)
		if err := deliverReport(ctx, mediaStorage, notificationRepo, notifier, key, data, usr, "Sales"); err != nil {
			return fmt.Errorf("report:generate: %w", err)
		}
		return nil
	})

	adminReportUC := reportusecase.NewAdmin(
		postgresrepo.NewOrderRepository(db),
		postgresrepo.NewLedgerEntryRepository(db),
		supplierRepo,
		userRepo,
		enqueuer,
	)
	mux.HandleFunc(queue.TaskAdminReportGenerate, func(ctx context.Context, t *asynq.Task) error {
		var p queue.AdminReportGeneratePayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("report:generate-admin: unmarshal payload: %w", err)
		}
		summary, err := adminReportUC.GetSummary(ctx, p.From, p.To)
		if err != nil {
			return fmt.Errorf("report:generate-admin: get summary: %w", err)
		}

		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		_ = w.Write([]string{"GMV", "Commission", "ActiveSuppliers", "NewUsers"})
		_ = w.Write([]string{
			fmt.Sprintf("%.2f", summary.GMV), fmt.Sprintf("%.2f", summary.Commission),
			fmt.Sprintf("%d", summary.ActiveSuppliers), fmt.Sprintf("%d", summary.NewUsers),
		})
		_ = w.Write([]string{})
		_ = w.Write([]string{"Status", "Count"})
		for status, count := range summary.OrdersByStatus {
			_ = w.Write([]string{status, fmt.Sprintf("%d", count)})
		}
		_ = w.Write([]string{})
		_ = w.Write([]string{"Day", "GMV", "Orders"})
		for _, d := range summary.SalesPerDay {
			_ = w.Write([]string{d.Day.Format("2006-01-02"), fmt.Sprintf("%.2f", d.GMV), fmt.Sprintf("%d", d.Orders)})
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return fmt.Errorf("report:generate-admin: write csv: %w", err)
		}
		data := buf.Bytes()

		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("report:generate-admin: generate id: %w", err)
		}
		usr, err := userRepo.FindByID(ctx, p.AdminID)
		if err != nil {
			return fmt.Errorf("report:generate-admin: find admin: %w", err)
		}
		key := fmt.Sprintf("reports/admin/%s.csv", id)
		if err := deliverReport(ctx, mediaStorage, notificationRepo, notifier, key, data, usr, "Platform"); err != nil {
			return fmt.Errorf("report:generate-admin: %w", err)
		}
		return nil
	})

	scheduler := asynq.NewScheduler(redisOpt, nil)
	if _, err := scheduler.Register("0 2 * * *", asynq.NewTask(queue.TaskLowStockCheck, nil)); err != nil {
		fatal(err)
	}
	if _, err := scheduler.Register("* * * * *", asynq.NewTask(queue.TaskOrderExpire, nil)); err != nil {
		fatal(err)
	}
	if _, err := scheduler.Register("0 3 * * *", asynq.NewTask(queue.TaskOrderAutocomplete, nil)); err != nil {
		fatal(err)
	}
	if err := scheduler.Start(); err != nil {
		fatal(err)
	}
	defer scheduler.Shutdown()

	if err := srv.Run(mux); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
