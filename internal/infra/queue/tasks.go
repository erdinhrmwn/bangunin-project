// Package queue holds Asynq task type names and payload helpers shared by
// the API (enqueue) and worker (handle) processes.
package queue

import "encoding/json"

// Task type names.
const (
	TaskHeartbeat = "system:heartbeat"
	TaskEmailSend = "email:send"
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
