package service

// MediaEnqueuer queues a media:process job (FR-3.1) to resize an uploaded
// image. Implemented by infra/queue.Enqueuer.
type MediaEnqueuer interface {
	EnqueueMediaProcess(key string) error
}
