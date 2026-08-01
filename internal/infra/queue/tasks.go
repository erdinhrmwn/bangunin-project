// Package queue holds Asynq task type names and payload helpers shared by
// the API (enqueue) and worker (handle) processes.
package queue

import "encoding/json"

// Task type names.
const (
	TaskHeartbeat     = "system:heartbeat"
	TaskEmailSend     = "email:send"
	TaskMediaProcess  = "media:process"
	TaskLowStockCheck = "notification:lowstock"
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
