package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/service"
	"erdinhrmwn/bangunin/internal/domain/service/mocks"
	shippingusecase "erdinhrmwn/bangunin/internal/usecase/shipping"
)

func TestChargeableWeight_UsesVolumetricWhenGreater(t *testing.T) {
	items := []shippingusecase.Item{
		{Variant: &entity.ProductVariant{WeightGram: 1000, LengthCM: 60, WidthCM: 50, HeightCM: 40}, Qty: 2}, // volumetric = 60*50*40/6000 = 20 < 1000 -> actual wins
		{Variant: &entity.ProductVariant{WeightGram: 100, LengthCM: 100, WidthCM: 100, HeightCM: 100}, Qty: 1}, // volumetric = 100*100*100/6000 = 166 > 100 -> volumetric wins
	}
	require.Equal(t, 1000*2+166, shippingusecase.ChargeableWeight(items))
}

func TestGetCost_DelegatesToGateway(t *testing.T) {
	gw := mocks.NewMockShippingGateway(t)
	gw.EXPECT().GetCost(mock.Anything, 1, 2, 100).Return([]service.ShippingOption{{CourierCode: "jne", Cost: 15000}}, nil)

	uc := shippingusecase.New(gw)
	opts, err := uc.GetCost(context.Background(), 1, 2, []shippingusecase.Item{
		{Variant: &entity.ProductVariant{WeightGram: 100}, Qty: 1},
	})
	require.NoError(t, err)
	require.Len(t, opts, 1)
	require.Equal(t, "jne", opts[0].CourierCode)
}
