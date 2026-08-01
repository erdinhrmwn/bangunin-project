package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	MovementTypeIn             = "in"
	MovementTypeOut            = "out"
	MovementTypeAdjustment     = "adjustment"
	MovementTypeReserveConvert = "reserve_convert"

	RefTypeOrder  = "order"
	RefTypeManual = "manual"
	RefTypeImport = "import"
)

type StockMovement struct {
	ID         uuid.UUID
	VariantID  uuid.UUID
	Type       string
	Qty        int
	RefType    string
	RefID      *uuid.UUID
	StockAfter int
	Note       string
	CreatedAt  time.Time
}
