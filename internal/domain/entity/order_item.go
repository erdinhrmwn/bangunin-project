package entity

import "github.com/google/uuid"

type OrderItem struct {
	ID          uuid.UUID
	OrderID     uuid.UUID
	VariantID   uuid.UUID
	ProductName string
	VariantName string
	Unit        string
	Price       float64
	Qty         int
	WeightGram  int
}
