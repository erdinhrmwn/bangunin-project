package queue

import (
	"fmt"

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
