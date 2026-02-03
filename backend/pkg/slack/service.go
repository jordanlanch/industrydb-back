package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	// ErrSlackSendFailed is returned when Slack API fails
	ErrSlackSendFailed = errors.New("failed to send Slack notification")
)

// Message represents a Slack message
type Message struct {
	Text string `json:"text"`
}

// SlackClient is an interface for sending Slack notifications
type SlackClient interface {
	SendMessage(ctx context.Context, msg Message) error
}

// WebhookClient implements SlackClient using Slack webhooks
type WebhookClient struct {
	webhookURL string
	httpClient *http.Client
}

// NewWebhookClient creates a new Slack webhook client
func NewWebhookClient(webhookURL string) *WebhookClient {
	return &WebhookClient{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendMessage sends a message to Slack via webhook
func (c *WebhookClient) SendMessage(ctx context.Context, msg Message) error {
	if c.webhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ErrSlackSendFailed
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrSlackSendFailed
	}

	return nil
}

// Service handles Slack notifications
type Service struct {
	client SlackClient
}

// NewService creates a new Slack service
func NewService(client SlackClient) *Service {
	return &Service{
		client: client,
	}
}

// IsEnabled returns true if Slack notifications are enabled
func (s *Service) IsEnabled() bool {
	return s.client != nil
}

// NotifyNewLead sends a notification for a new lead
func (s *Service) NotifyNewLead(ctx context.Context, name, industry, country, city string) error {
	if !s.IsEnabled() {
		return nil // Silently skip if not enabled
	}

	text := fmt.Sprintf("🎯 *New Lead*\n"+
		"• Name: %s\n"+
		"• Industry: %s\n"+
		"• Location: %s, %s",
		name, industry, city, country)

	msg := Message{Text: text}
	return s.client.SendMessage(ctx, msg)
}

// NotifyExportComplete sends a notification when an export is complete
func (s *Service) NotifyExportComplete(ctx context.Context, userEmail, format string, leadCount int) error {
	if !s.IsEnabled() {
		return nil
	}

	text := fmt.Sprintf("📊 *Export Complete*\n"+
		"• User: %s\n"+
		"• Format: %s\n"+
		"• Leads: %d",
		userEmail, format, leadCount)

	msg := Message{Text: text}
	return s.client.SendMessage(ctx, msg)
}

// NotifySubscriptionUpgrade sends a notification when a user upgrades
func (s *Service) NotifySubscriptionUpgrade(ctx context.Context, userEmail, fromTier, toTier string) error {
	if !s.IsEnabled() {
		return nil
	}

	text := fmt.Sprintf("🚀 *Subscription Upgrade*\n"+
		"• User: %s\n"+
		"• From: %s\n"+
		"• To: %s",
		userEmail, fromTier, toTier)

	msg := Message{Text: text}
	return s.client.SendMessage(ctx, msg)
}

// NotifySubscriptionCancellation sends a notification when a subscription is canceled
func (s *Service) NotifySubscriptionCancellation(ctx context.Context, userEmail, tier, reason string) error {
	if !s.IsEnabled() {
		return nil
	}

	text := fmt.Sprintf("⚠️ *Subscription Canceled*\n"+
		"• User: %s\n"+
		"• Tier: %s",
		userEmail, tier)

	if reason != "" {
		text += fmt.Sprintf("\n• Reason: %s", reason)
	}

	msg := Message{Text: text}
	return s.client.SendMessage(ctx, msg)
}

// NotifyNewUser sends a notification when a new user registers
func (s *Service) NotifyNewUser(ctx context.Context, name, email string) error {
	if !s.IsEnabled() {
		return nil
	}

	text := fmt.Sprintf("👤 *New User Registration*\n"+
		"• Name: %s\n"+
		"• Email: %s",
		name, email)

	msg := Message{Text: text}
	return s.client.SendMessage(ctx, msg)
}
