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

const mailjetSendURL = "https://api.mailjet.com/v3.1/send"

// mailjetClient wraps the Mailjet Send API. In mock mode it returns a
// deterministic fake message id without calling out — used for local dev/CI
// and when Mailjet isn't reachable.
type mailjetClient struct {
	apiKey   string
	secret   string
	fromAddr string
	fromName string
	mock     bool
	http     *http.Client
}

func newMailjetClient(apiKey, secret, fromAddr, fromName string, mock bool) *mailjetClient {
	return &mailjetClient{
		apiKey: apiKey, secret: secret, fromAddr: fromAddr, fromName: fromName,
		mock: mock, http: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *mailjetClient) SendEmail(ctx context.Context, to, subject, body string) (messageID string, err error) {
	if c.mock {
		return "mock-msg-" + uuid.NewString(), nil
	}

	payload, _ := json.Marshal(map[string]any{
		"Messages": []map[string]any{
			{
				"From":     map[string]string{"Email": c.fromAddr, "Name": c.fromName},
				"To":       []map[string]string{{"Email": to}},
				"Subject":  subject,
				"TextPart": body,
			},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mailjetSendURL, strings.NewReader(string(payload)))
	if err != nil {
		return "", fmt.Errorf("mailjet: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.apiKey+":"+c.secret)))

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("mailjet: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mailjet: unexpected status %d", resp.StatusCode)
	}

	var out struct {
		Messages []struct {
			To []struct {
				MessageID string `json:"MessageID"`
			} `json:"To"`
		} `json:"Messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("mailjet: decode response: %w", err)
	}
	if len(out.Messages) == 0 || len(out.Messages[0].To) == 0 {
		return "", fmt.Errorf("mailjet: empty response")
	}
	return fmt.Sprint(out.Messages[0].To[0].MessageID), nil
}
