package entity

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID         uuid.UUID
	ActorID    uuid.UUID
	Action     string
	EntityType string
	EntityID   uuid.UUID
	Metadata   map[string]any
	IPAddress  string
	CreatedAt  time.Time
}
