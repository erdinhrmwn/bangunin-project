package service

// EmailEnqueuer queues an email:send job (FR-2.2). Implemented by
// infra/queue.Enqueuer; the worker delivers via NotificationService.
type EmailEnqueuer interface {
	EnqueueEmail(to, subject, body string) error
}
