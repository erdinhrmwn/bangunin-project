// Package queue holds Asynq task type names and payload helpers shared by
// the API (enqueue) and worker (handle) processes.
package queue

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Task type names.
const (
	TaskHeartbeat           = "system:heartbeat"
	TaskEmailSend           = "email:send"
	TaskMediaProcess        = "media:process"
	TaskLowStockCheck       = "notification:lowstock"
	TaskOrderExpire         = "order:expire"
	TaskOrderAutocomplete   = "order:autocomplete"
	TaskReportGenerate      = "report:generate"
	TaskAdminReportGenerate = "report:generate-admin"
)

// EmailSendPayload is TaskEmailSend's JSON payload.
type EmailSendPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (p EmailSendPayload) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

// MediaProcessPayload is TaskMediaProcess's JSON payload — Key is the
// object's storage key, resized in place after upload.
type MediaProcessPayload struct {
	Key string `json:"key"`
}

func (p MediaProcessPayload) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

// ReportGeneratePayload is TaskReportGenerate's JSON payload.
type ReportGeneratePayload struct {
	SupplierID uuid.UUID `json:"supplier_id"`
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
}

func (p ReportGeneratePayload) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

// AdminReportGeneratePayload is TaskAdminReportGenerate's JSON payload.
type AdminReportGeneratePayload struct {
	AdminID uuid.UUID `json:"admin_id"`
	From    time.Time `json:"from"`
	To      time.Time `json:"to"`
}

func (p AdminReportGeneratePayload) Marshal() ([]byte, error) {
	return json.Marshal(p)
}
