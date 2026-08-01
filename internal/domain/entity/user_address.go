package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	AddressLabelHome      = "home"
	AddressLabelProject   = "project"
	AddressLabelWarehouse = "warehouse"
)

type UserAddress struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Label          string
	RecipientName  string
	RecipientPhone string
	ProvinceID     int
	CityID         int
	Subdistrict    string
	PostalCode     string
	AddressDetail  string
	IsDefault      bool
	CreatedAt      time.Time
}
