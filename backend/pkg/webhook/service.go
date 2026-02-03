package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/webhook"
)

// Service handles webhook operations
type Service struct {
	client     *ent.Client
	httpClient *http.Client
}

// NewService creates a new webhook service
func NewService(client *ent.Client) *Service {
	return &Service{
		client: client,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Event types
const (
	EventLeadCreated     = "lead.created"
	EventExportCompleted = "export.completed"
	EventExportFailed    = "export.failed"
	EventUserRegistered  = "user.registered"
)

// Payload represents a webhook payload
type Payload struct {
	Event     string                 `json:"event"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

// CreateWebhook creates a new webhook for a user
func (s *Service) CreateWebhook(ctx context.Context, userID int, url string, events []string, description string) (*ent.Webhook, error) {
	// Generate secret for HMAC signature
	secret, err := generateSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	webhook, err := s.client.Webhook.Create().
		SetUserID(userID).
		SetURL(url).
		SetEvents(events).
		SetSecret(secret).
		SetDescription(description).
		SetActive(true).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	return webhook, nil
}

// ListWebhooks lists all webhooks for a user
func (s *Service) ListWebhooks(ctx context.Context, userID int) ([]*ent.Webhook, error) {
	webhooks, err := s.client.Webhook.Query().
		Where(webhook.HasUserWith(func(q *ent.UserQuery) {
			q.Where(func(s *ent.UserQuery) {
				s.Where(func(s *ent.UserQuery) {
					s.IDEQ(userID)
				})
			})
		})).
		Order(ent.Desc(webhook.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	return webhooks, nil
}

// GetWebhook retrieves a webhook by ID
func (s *Service) GetWebhook(ctx context.Context, webhookID int, userID int) (*ent.Webhook, error) {
	wh, err := s.client.Webhook.Query().
		Where(
			webhook.ID(webhookID),
			webhook.HasUserWith(func(q *ent.UserQuery) {
				q.IDEQ(userID)
			}),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}

	return wh, nil
}

// UpdateWebhook updates a webhook
func (s *Service) UpdateWebhook(ctx context.Context, webhookID int, userID int, url *string, events []string, active *bool) (*ent.Webhook, error) {
	update := s.client.Webhook.UpdateOneID(webhookID).
		Where(webhook.HasUserWith(func(q *ent.UserQuery) {
			q.IDEQ(userID)
		}))

	if url != nil {
		update.SetURL(*url)
	}
	if events != nil {
		update.SetEvents(events)
	}
	if active != nil {
		update.SetActive(*active)
	}

	wh, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update webhook: %w", err)
	}

	return wh, nil
}

// DeleteWebhook deletes a webhook
func (s *Service) DeleteWebhook(ctx context.Context, webhookID int, userID int) error {
	_, err := s.client.Webhook.Delete().
		Where(
			webhook.ID(webhookID),
			webhook.HasUserWith(func(q *ent.UserQuery) {
				q.IDEQ(userID)
			}),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	return nil
}

// TriggerWebhooks triggers all active webhooks for a specific event
func (s *Service) TriggerWebhooks(ctx context.Context, userID int, event string, data map[string]interface{}) {
	// Query active webhooks that subscribe to this event
	webhooks, err := s.client.Webhook.Query().
		Where(
			webhook.HasUserWith(func(q *ent.UserQuery) {
				q.IDEQ(userID)
			}),
			webhook.Active(true),
		).
		All(ctx)
	if err != nil {
		log.Printf("⚠️  Failed to query webhooks for event %s: %v", event, err)
		return
	}

	// Filter webhooks that subscribe to this event
	for _, wh := range webhooks {
		if containsEvent(wh.Events, event) {
			// Trigger webhook asynchronously
			go s.deliverWebhook(wh, event, data)
		}
	}
}

// deliverWebhook delivers a webhook with retries
func (s *Service) deliverWebhook(wh *ent.Webhook, event string, data map[string]interface{}) {
	ctx := context.Background()

	payload := Payload{
		Event:     event,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	// Marshal payload
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("⚠️  Failed to marshal webhook payload: %v", err)
		s.incrementFailureCount(ctx, wh.ID)
		return
	}

	// Generate HMAC signature
	signature := generateSignature(body, wh.Secret)

	// Attempt delivery with retries
	maxRetries := wh.RetryCount
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2^attempt seconds
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoff)
		}

		// Create HTTP request
		req, err := http.NewRequest("POST", wh.URL, bytes.NewReader(body))
		if err != nil {
			log.Printf("⚠️  Failed to create webhook request: %v", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Signature", signature)
		req.Header.Set("X-Webhook-Event", event)

		// Send request
		resp, err := s.httpClient.Do(req)
		if err != nil {
			log.Printf("⚠️  Webhook delivery failed (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
			continue
		}

		// Check response
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("✅ Webhook delivered successfully: %s (event: %s)", wh.URL, event)
			s.incrementSuccessCount(ctx, wh.ID)
			resp.Body.Close()
			return
		}

		log.Printf("⚠️  Webhook returned error status %d (attempt %d/%d)", resp.StatusCode, attempt+1, maxRetries+1)
		resp.Body.Close()
	}

	// All retries failed
	log.Printf("❌ Webhook delivery failed after %d attempts: %s (event: %s)", maxRetries+1, wh.URL, event)
	s.incrementFailureCount(ctx, wh.ID)
}

// incrementSuccessCount increments the success count for a webhook
func (s *Service) incrementSuccessCount(ctx context.Context, webhookID int) {
	_, err := s.client.Webhook.UpdateOneID(webhookID).
		AddSuccessCount(1).
		SetLastTriggeredAt(time.Now()).
		Save(ctx)
	if err != nil {
		log.Printf("⚠️  Failed to update webhook success count: %v", err)
	}
}

// incrementFailureCount increments the failure count for a webhook
func (s *Service) incrementFailureCount(ctx context.Context, webhookID int) {
	_, err := s.client.Webhook.UpdateOneID(webhookID).
		AddFailureCount(1).
		SetLastTriggeredAt(time.Now()).
		Save(ctx)
	if err != nil {
		log.Printf("⚠️  Failed to update webhook failure count: %v", err)
	}
}

// generateSecret generates a random secret for HMAC signature
func generateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateSignature generates HMAC-SHA256 signature for webhook payload
func generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies the HMAC signature of a webhook payload
func VerifySignature(payload []byte, signature string, secret string) bool {
	expected := generateSignature(payload, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}

// containsEvent checks if events slice contains a specific event
func containsEvent(events []string, event string) bool {
	for _, e := range events {
		if e == event {
			return true
		}
	}
	return false
}
