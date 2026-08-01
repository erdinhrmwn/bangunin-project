package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	UserStatusActive    = "active"
	UserStatusSuspended = "suspended"
	UserStatusBanned    = "banned"
)

type User struct {
	ID              uuid.UUID
	RoleID          int16
	Name            string
	Email           string
	PasswordHash    string
	Phone           string
	Status          string
	EmailVerifiedAt *time.Time
	EmailMarketing  bool
	CreatedAt       time.Time
}
