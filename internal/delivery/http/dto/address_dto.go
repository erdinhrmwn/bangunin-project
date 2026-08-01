package dto

type AddressRequest struct {
	Label          string `json:"label" validate:"required,oneof=home project warehouse"`
	RecipientName  string `json:"recipient_name" validate:"required"`
	RecipientPhone string `json:"recipient_phone" validate:"required"`
	ProvinceID     int    `json:"province_id" validate:"required,gt=0"`
	CityID         int    `json:"city_id" validate:"required,gt=0"`
	Subdistrict    string `json:"subdistrict" validate:"required"`
	PostalCode     string `json:"postal_code" validate:"required"`
	AddressDetail  string `json:"address_detail" validate:"required"`
	IsDefault      bool   `json:"is_default"`
}
