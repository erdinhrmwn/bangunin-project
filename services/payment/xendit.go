package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const xenditInvoiceURL = "https://api.xendit.co/v2/invoices"

// xenditClient wraps the Xendit Invoice API. In mock mode it returns a
// deterministic fake invoice without calling out — used for local dev/CI
// and when Xendit isn't reachable (AC-5.e: monolith must degrade, not crash).
type xenditClient struct {
	secretKey string
	mock      bool
	http      *http.Client
}

func newXenditClient(secretKey string, mock bool) *xenditClient {
	return &xenditClient{secretKey: secretKey, mock: mock, http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *xenditClient) CreateInvoice(ctx context.Context, externalID string, amount float64, description string) (invoiceID, invoiceURL string, err error) {
	if c.mock {
		id := "mock-inv-" + uuid.NewString()
		return id, "https://mock.xendit.co/invoices/" + id, nil
	}

	body, _ := json.Marshal(map[string]any{
		"external_id": externalID,
		"amount":      amount,
		"description": description,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, xenditInvoiceURL, strings.NewReader(string(body)))
	if err != nil {
		return "", "", fmt.Errorf("xendit: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.secretKey+":")))

	resp, err := c.http.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("xendit: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("xendit: unexpected status %d", resp.StatusCode)
	}

	var out struct {
		ID         string `json:"id"`
		InvoiceURL string `json:"invoice_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", fmt.Errorf("xendit: decode response: %w", err)
	}
	return out.ID, out.InvoiceURL, nil
}

const xenditDisbursementURL = "https://api.xendit.co/disbursements"

func (c *xenditClient) Disburse(ctx context.Context, externalID string, amount float64, bankCode, accountNumber, accountName string) (disbursementID string, err error) {
	if c.mock {
		return "mock-disb-" + uuid.NewString(), nil
	}

	body, _ := json.Marshal(map[string]any{
		"external_id":         externalID,
		"amount":              amount,
		"bank_code":           bankCode,
		"account_holder_name": accountName,
		"account_number":      accountNumber,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, xenditDisbursementURL, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("xendit: build disbursement request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.secretKey+":")))

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("xendit: disbursement request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("xendit: unexpected disbursement status %d", resp.StatusCode)
	}

	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("xendit: decode disbursement response: %w", err)
	}
	return out.ID, nil
}
