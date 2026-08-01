package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
)

// callbackHandler receives Xendit's invoice webhook, verifies the callback
// token, then relays the event to the monolith's internal endpoint
// (FR-5.5: payment-service must not touch the monolith's DB directly).
type callbackHandler struct {
	cfg config
	log zerolog.Logger
}

func (h *callbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Mock && r.Header.Get("x-callback-token") != h.cfg.XenditCallbackToken {
		http.Error(w, "invalid callback token", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	var event struct {
		ExternalID string `json:"external_id"`
		Status     string `json:"status"`
		ID         string `json:"id"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if err := h.relay(r.Context(), event.ExternalID, event.ID, event.Status); err != nil {
		h.log.Error().Err(err).Str("external_id", event.ExternalID).Msg("relay to monolith failed")
		http.Error(w, "relay failed", http.StatusBadGateway)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *callbackHandler) relay(ctx context.Context, checkoutGroupID, invoiceID, status string) error {
	payload, _ := json.Marshal(map[string]string{
		"checkout_group_id": checkoutGroupID,
		"xendit_invoice_id": invoiceID,
		"status":            status,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.cfg.MonolithCallbackURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", h.cfg.InternalSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("monolith callback returned status %d", resp.StatusCode)
	}
	return nil
}
