package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

const brevoSMTPEndpoint = "https://api.brevo.com/v3/smtp/email"

type BrevoAPISender struct {
	apiKey string
	from   mail.Address
	client *http.Client
}

func NewBrevoAPISender(apiKey, from string) *BrevoAPISender {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		addr = &mail.Address{Name: "UNESA Rent", Address: strings.TrimSpace(from)}
	}
	return &BrevoAPISender{
		apiKey: apiKey,
		from:   *addr,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *BrevoAPISender) Send(ctx context.Context, to, subject, text string) error {
	payload := map[string]any{
		"sender":      map[string]string{"name": s.from.Name, "email": s.from.Address},
		"to":          []map[string]string{{"email": to}},
		"subject":     subject,
		"textContent": text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("menyiapkan email Brevo: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, brevoSMTPEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("membuat request Brevo: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("menghubungi Brevo API: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Brevo API status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}
