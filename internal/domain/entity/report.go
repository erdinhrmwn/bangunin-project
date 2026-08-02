package entity

import "time"

// TopProduct is one row of a supplier sales report's top-products ranking.
type TopProduct struct {
	ProductName string
	Qty         int
	Revenue     float64
}

// DailySales is one row of a supplier sales report's per-day series.
type DailySales struct {
	Day    time.Time
	GMV    float64
	Orders int
}
