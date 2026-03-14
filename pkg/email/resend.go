package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// resendAPI is a var so tests can override it with a local httptest.Server URL.
var resendAPI = "https://api.resend.com/emails"

// ResendSender sends emails via the Resend REST API (https://resend.com).
type ResendSender struct {
	apiKey string
	from   string
	client *http.Client
}

// NewResendSender creates a ResendSender. apiKey is the RESEND_API_KEY env var.
func NewResendSender(apiKey, from string) *ResendSender {
	return &ResendSender{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{},
	}
}

func (r *ResendSender) Send(msg Message) error {
	payload := map[string]any{
		"from":    r.from,
		"to":      []string{msg.To},
		"subject": msg.Subject,
		"html":    msg.HTML,
	}
	if msg.Text != "" {
		payload["text"] = msg.Text
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("resend: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, resendAPI, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("resend: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("resend: unexpected status %d", resp.StatusCode)
	}
	return nil
}
