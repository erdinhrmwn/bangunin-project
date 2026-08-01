package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	NotificationTypeOrderUpdate = "order_update"
	NotificationTypeApproval    = "approval"
	NotificationTypePayout      = "payout"
	NotificationTypeSystem      = "system"
)

type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      string
	Title     string
	Body      string
	Data      map[string]any
	ReadAt    *time.Time
	CreatedAt time.Time
}
