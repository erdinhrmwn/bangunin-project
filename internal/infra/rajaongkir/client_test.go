package rajaongkir_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/config"
	"erdinhrmwn/bangunin/internal/infra/rajaongkir"
)

func TestClient_GetCost_Mock(t *testing.T) {
	c := rajaongkir.New(config.RajaOngkirConfig{Mock: true})

	opts, err := c.GetCost(context.Background(), 1, 2, 2500)
	require.NoError(t, err)
	require.Len(t, opts, 2)
	for _, o := range opts {
		assert.Equal(t, 2.5*15000.0, o.Cost)
		assert.NotEmpty(t, o.CourierCode)
	}
}

func TestClient_GetCost_Mock_MinimumOneKg(t *testing.T) {
	c := rajaongkir.New(config.RajaOngkirConfig{Mock: true})

	opts, err := c.GetCost(context.Background(), 1, 2, 300)
	require.NoError(t, err)
	require.NotEmpty(t, opts)
	assert.Equal(t, 15000.0, opts[0].Cost)
}

func TestClient_TrackWaybill_NotImplemented(t *testing.T) {
	c := rajaongkir.New(config.RajaOngkirConfig{Mock: true})

	_, err := c.TrackWaybill(context.Background(), "AB123")
	assert.Error(t, err)
}
