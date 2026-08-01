// Package rajaongkir implements domain/service.ShippingGateway against the
// RajaOngkir (Komerce) HTTP API. In Mock mode (dev/test default) it returns
// deterministic fixture costs instead of calling out.
package rajaongkir

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"erdinhrmwn/bangunin/config"
	"erdinhrmwn/bangunin/internal/domain/service"
)

const baseURL = "https://rajaongkir.komerce.id/api/v1/calculate/domestic-cost"

// mockCouriers/mockRatePerKg: ponytail: fixed fixture values for local dev
// (Mock=true), promote to config if a real sandbox account is wired up.
var mockCouriers = []string{"jne", "sicepat"}

const mockRatePerKg = 15000.0

type Client struct {
	apiKey     string
	mock       bool
	httpClient *http.Client
}

func New(cfg config.RajaOngkirConfig) *Client {
	return &Client{apiKey: cfg.APIKey, mock: cfg.Mock, httpClient: &http.Client{Timeout: 5 * time.Second}}
}

// GetCost tries the real API once, retries once on failure (FR-5.2), and
// maps any remaining error to service.ErrShippingUnavailable.
func (c *Client) GetCost(ctx context.Context, originCityID, destCityID int, weightGram int) ([]service.ShippingOption, error) {
	if c.mock {
		return c.mockCost(weightGram), nil
	}
	opts, err := c.realCost(ctx, originCityID, destCityID, weightGram)
	if err != nil {
		opts, err = c.realCost(ctx, originCityID, destCityID, weightGram)
	}
	if err != nil {
		return nil, service.ErrShippingUnavailable
	}
	return opts, nil
}

func (c *Client) mockCost(weightGram int) []service.ShippingOption {
	kg := float64(weightGram) / 1000
	if kg < 1 {
		kg = 1
	}
	opts := make([]service.ShippingOption, 0, len(mockCouriers))
	for _, courier := range mockCouriers {
		opts = append(opts, service.ShippingOption{
			CourierCode:    courier,
			CourierService: "REG",
			Cost:           kg * mockRatePerKg,
			ETD:            "2-3 days",
		})
	}
	return opts
}

func (c *Client) realCost(ctx context.Context, originCityID, destCityID int, weightGram int) ([]service.ShippingOption, error) {
	form := url.Values{
		"origin":      {strconv.Itoa(originCityID)},
		"destination": {strconv.Itoa(destCityID)},
		"weight":      {strconv.Itoa(weightGram)},
		"courier":     {strings.Join(mockCouriers, ":")},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("rajaongkir: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rajaongkir: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rajaongkir: unexpected status %d", resp.StatusCode)
	}

	var body struct {
		Data []struct {
			Code    string  `json:"code"`
			Service string  `json:"service"`
			Cost    float64 `json:"cost"`
			ETD     string  `json:"etd"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("rajaongkir: decode response: %w", err)
	}

	opts := make([]service.ShippingOption, 0, len(body.Data))
	for _, d := range body.Data {
		opts = append(opts, service.ShippingOption{CourierCode: d.Code, CourierService: d.Service, Cost: d.Cost, ETD: d.ETD})
	}
	return opts, nil
}

// TrackWaybill is a stub — real tracking sync is out of scope for phase 5
// (FR-5.9 note) beyond the method existing on ShippingGateway.
func (c *Client) TrackWaybill(ctx context.Context, waybill string) (*service.TrackInfo, error) {
	return nil, fmt.Errorf("rajaongkir: TrackWaybill not implemented")
}
