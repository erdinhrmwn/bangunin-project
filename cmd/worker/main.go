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
	"erdinhrmwn/bangunin/internal/infra/grpcclient"
	"erdinhrmwn/bangunin/internal/infra/queue"
	"erdinhrmwn/bangunin/internal/infra/storage"
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

	notifier := grpcclient.NewNotificationStub(log)
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

	// Scheduler is wired but intentionally empty — periodic jobs (e.g.
	// notification:lowstock) are registered in their owning phase.
	scheduler := asynq.NewScheduler(redisOpt, nil)
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
