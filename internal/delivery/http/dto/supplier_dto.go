package dto

type UpsertProfileRequest struct {
	StoreName       string  `json:"store_name" validate:"required"`
	Description     string  `json:"description"`
	OriginCityID    int     `json:"origin_city_id" validate:"required,gt=0"`
	PickupAddress   string  `json:"pickup_address" validate:"required"`
	OwnFleetEnabled bool    `json:"own_fleet_enabled"`
	FleetCoverageKM int     `json:"fleet_coverage_km"`
	FleetFlatRate   float64 `json:"fleet_flat_rate"`
}

type UploadDocumentRequest struct {
	DocType string `json:"doc_type" validate:"required,oneof=nib ktp npwp"`
	FileKey string `json:"file_key" validate:"required"`
}

type BankAccountRequest struct {
	BankCode      string `json:"bank_code" validate:"required"`
	AccountNumber string `json:"account_number" validate:"required,min=6,max=20,numeric"`
	AccountName   string `json:"account_name" validate:"required"`
	IsDefault     bool   `json:"is_default"`
}
