package queue

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// Enqueuer wraps asynq.Client for task types defined in this package.
type Enqueuer struct {
	client *asynq.Client
}

func NewEnqueuer(redisOpt asynq.RedisClientOpt) *Enqueuer {
	return &Enqueuer{client: asynq.NewClient(redisOpt)}
}

func (e *Enqueuer) Close() error {
	return e.client.Close()
}

func (e *Enqueuer) EnqueueEmail(to, subject, body string) error {
	payload, err := EmailSendPayload{To: to, Subject: subject, Body: body}.Marshal()
	if err != nil {
		return fmt.Errorf("queue: marshal email payload: %w", err)
	}
	_, err = e.client.Enqueue(asynq.NewTask(TaskEmailSend, payload))
	if err != nil {
		return fmt.Errorf("queue: enqueue email: %w", err)
	}
	return nil
}

func (e *Enqueuer) EnqueueMediaProcess(key string) error {
	payload, err := MediaProcessPayload{Key: key}.Marshal()
	if err != nil {
		return fmt.Errorf("queue: marshal media process payload: %w", err)
	}
	_, err = e.client.Enqueue(asynq.NewTask(TaskMediaProcess, payload))
	if err != nil {
		return fmt.Errorf("queue: enqueue media process: %w", err)
	}
	return nil
}

func (e *Enqueuer) EnqueueReportGenerate(supplierID uuid.UUID, from, to time.Time) error {
	payload, err := ReportGeneratePayload{SupplierID: supplierID, From: from, To: to}.Marshal()
	if err != nil {
		return fmt.Errorf("queue: marshal report generate payload: %w", err)
	}
	_, err = e.client.Enqueue(asynq.NewTask(TaskReportGenerate, payload))
	if err != nil {
		return fmt.Errorf("queue: enqueue report generate: %w", err)
	}
	return nil
}
