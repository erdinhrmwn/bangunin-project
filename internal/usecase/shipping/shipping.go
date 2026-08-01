// Package shipping resolves courier shipping cost quotes (FR-5.2): computes
// chargeable weight (max of actual vs. volumetric) and delegates to
// ShippingGateway.
package shipping

import (
	"context"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/service"
)

type Usecase struct {
	gateway service.ShippingGateway
}

func New(gateway service.ShippingGateway) *Usecase {
	return &Usecase{gateway: gateway}
}

// GetCost returns courier quotes for a set of variants (with quantities)
// shipped from originCityID to destCityID. Chargeable weight per CLAUDE.md
// §5.2: max(actual, volumetric = l*w*h/6000), summed across items.
func (u *Usecase) GetCost(ctx context.Context, originCityID, destCityID int, items []Item) ([]service.ShippingOption, error) {
	weight := ChargeableWeight(items)
	return u.gateway.GetCost(ctx, originCityID, destCityID, weight)
}

// Item is a variant+qty pair used to compute chargeable weight.
type Item struct {
	Variant *entity.ProductVariant
	Qty     int
}

// ChargeableWeight sums, per item, max(actual weight, volumetric weight) * qty.
func ChargeableWeight(items []Item) int {
	total := 0
	for _, it := range items {
		volumetric := it.Variant.LengthCM * it.Variant.WidthCM * it.Variant.HeightCM / 6000
		w := it.Variant.WeightGram
		if volumetric > w {
			w = volumetric
		}
		total += w * it.Qty
	}
	return total
}
