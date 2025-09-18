// Package email provides Resend email provider implementation
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ResendEmail represents the request payload for Resend API
type ResendEmail struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text,omitempty"`
	ReplyTo string   `json:"reply_to,omitempty"`
}

// ResendResponse represents the response from Resend API
type ResendResponse struct {
	ID    string `json:"id"`
	Error *struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ResendProvider implements EmailProvider interface using Resend API
// Strictly aligned with trigger.dev's Resend client usage
type ResendProvider struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewResendProvider creates a new Resend email provider
func NewResendProvider(apiKey string) EmailProvider {
	return &ResendProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.resend.com",
	}
}

// SendEmail sends an email through Resend API
// Aligned with trigger.dev's #sendEmail method
func (r *ResendProvider) SendEmail(ctx context.Context, email Email) error {
	// Validate API key
	if r.apiKey == "" {
		return fmt.Errorf("resend API key is required")
	}

	// Convert email to Resend format
	resendEmail := ResendEmail{
		From:    email.From,
		To:      []string{email.To},
		Subject: email.Subject,
		HTML:    email.HTML,
		Text:    email.Text,
		ReplyTo: email.ReplyTo,
	}

	// Marshal request payload
	payload, err := json.Marshal(resendEmail)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", r.baseURL+"/emails", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	// Send request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var resendResp ResendResponse
	if err := json.Unmarshal(body, &resendResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if resp.StatusCode >= 400 {
		if resendResp.Error != nil {
			return fmt.Errorf("resend API error: %s - %s", resendResp.Error.Name, resendResp.Error.Message)
		}
		return fmt.Errorf("resend API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
