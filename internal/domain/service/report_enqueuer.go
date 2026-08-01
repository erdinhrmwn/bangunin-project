package service

import (
	"time"

	"github.com/google/uuid"
)

// ReportEnqueuer queues a report:generate job (FR-7.3) to build a CSV export
// and notify the supplier with a download link. Implemented by infra/queue.Enqueuer.
type ReportEnqueuer interface {
	EnqueueReportGenerate(supplierID uuid.UUID, from, to time.Time) error
}
