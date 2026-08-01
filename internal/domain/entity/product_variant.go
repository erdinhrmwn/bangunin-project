package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	UnitSak    = "sak"
	UnitBatang = "batang"
	UnitM3     = "m3"
	UnitDus    = "dus"
	UnitPcs    = "pcs"
	UnitLembar = "lembar"
	UnitKg     = "kg"
	UnitRoll   = "roll"
	UnitSet    = "set"
)

type ProductVariant struct {
	ID          uuid.UUID
	ProductID   uuid.UUID
	SupplierID  uuid.UUID
	SKU         string
	Name        string
	Unit        string
	Price       float64
	WeightGram  int
	LengthCM    int
	WidthCM     int
	HeightCM    int
	MinOrderQty int
	Stock       int
	IsActive    bool
	CreatedAt   time.Time
}
