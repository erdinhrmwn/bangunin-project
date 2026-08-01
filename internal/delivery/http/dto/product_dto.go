package dto

type ProductRequest struct {
	CategoryID  int            `json:"category_id" validate:"required,gt=0"`
	Name        string         `json:"name" validate:"required"`
	Description string         `json:"description"`
	Specs       map[string]any `json:"specs"`
}

type VariantRequest struct {
	SKU         string  `json:"sku" validate:"required"`
	Name        string  `json:"name" validate:"required"`
	Unit        string  `json:"unit" validate:"required,oneof=sak batang m3 dus pcs lembar kg roll set"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	WeightGram  int     `json:"weight_gram" validate:"required,gt=0"`
	LengthCM    int     `json:"length_cm"`
	WidthCM     int     `json:"width_cm"`
	HeightCM    int     `json:"height_cm"`
	MinOrderQty int     `json:"min_order_qty" validate:"omitempty,gte=1"`
	Stock       int     `json:"stock" validate:"gte=0"`
	IsActive    bool    `json:"is_active"`
}

type AttachImageRequest struct {
	URL       string `json:"url" validate:"required,url"`
	IsPrimary bool   `json:"is_primary"`
}

type StockAdjustmentRequest struct {
	Qty  int    `json:"qty" validate:"required"`
	Note string `json:"note"`
}
