package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	ActorTypeUser     = "user"
	ActorTypeSupplier = "supplier"
	ActorTypeAdmin    = "admin"
	ActorTypeSystem   = "system"
)

type OrderStatusHistory struct {
	ID         uuid.UUID
	OrderID    uuid.UUID
	FromStatus string
	ToStatus   string
	ActorType  string
	ActorID    *uuid.UUID
	Note       string
	CreatedAt  time.Time
}
