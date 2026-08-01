// Package queue holds Asynq task type names and payload helpers shared by
// the API (enqueue) and worker (handle) processes.
package queue

// Task type names. Real task payloads/handlers land alongside their owning
// phase (e.g. email:send in Fase 2); only the heartbeat dummy exists here.
const (
	TaskHeartbeat = "system:heartbeat"
)
