// Command worker runs the Asynq background job server.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hibiken/asynq"

	"erdinhrmwn/bangunin/config"
	"erdinhrmwn/bangunin/internal/infra/database"
	"erdinhrmwn/bangunin/internal/infra/grpcclient"
	"erdinhrmwn/bangunin/internal/infra/queue"
	infraredis "erdinhrmwn/bangunin/internal/infra/redis"
	"erdinhrmwn/bangunin/internal/infra/storage"
	postgresrepo "erdinhrmwn/bangunin/internal/repository/postgres"
	inventoryusecase "erdinhrmwn/bangunin/internal/usecase/inventory"
	orderusecase "erdinhrmwn/bangunin/internal/usecase/order"
	"erdinhrmwn/bangunin/pkg/imageresize"
	"erdinhrmwn/bangunin/pkg/logger"
)

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
	)
	mux.HandleFunc(queue.TaskOrderExpire, func(ctx context.Context, t *asynq.Task) error {
		return orderUC.HandleExpire(ctx)
	})
	mux.HandleFunc(queue.TaskOrderAutocomplete, func(ctx context.Context, t *asynq.Task) error {
		return orderUC.HandleAutocomplete(ctx)
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
