package sms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/ent/smscampaign"
	"github.com/jordanlanch/industrydb/ent/smsmessage"
)

var (
	// ErrCampaignNotFound is returned when campaign doesn't exist
	ErrCampaignNotFound = errors.New("campaign not found")
	// ErrCampaignAlreadySent is returned when trying to send an already sent campaign
	ErrCampaignAlreadySent = errors.New("campaign already sent")
	// ErrInvalidPhoneNumber is returned when phone number format is invalid
	ErrInvalidPhoneNumber = errors.New("invalid phone number format")
)

// SMSProvider defines the interface for SMS delivery providers (Twilio, etc.)
type SMSProvider interface {
	SendSMS(ctx context.Context, to, from, body string) (*SMSResult, error)
	GetMessageStatus(ctx context.Context, sid string) (*MessageStatus, error)
}

// SMSResult holds the result of sending an SMS
type SMSResult struct {
	SID         string
	Status      string
	Cost        float64
	DateCreated time.Time
}

// MessageStatus holds the delivery status of an SMS
type MessageStatus struct {
	SID       string
	Status    string
	ErrorCode int
	ErrorMsg  string
}

// CampaignFilters holds filters for targeting leads
type CampaignFilters struct {
	Industry string
	Country  string
	City     string
}

// CampaignStats holds statistics for a campaign
type CampaignStats struct {
	CampaignID      int     `json:"campaign_id"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	TotalRecipients int     `json:"total_recipients"`
	SentCount       int     `json:"sent_count"`
	DeliveredCount  int     `json:"delivered_count"`
	FailedCount     int     `json:"failed_count"`
	EstimatedCost   float64 `json:"estimated_cost"`
	ActualCost      float64 `json:"actual_cost"`
	DeliveryRate    float64 `json:"delivery_rate"`
}

// Service handles SMS operations
type Service struct {
	db       *ent.Client
	provider SMSProvider
	fromNumber string
}

// NewService creates a new SMS service
func NewService(db *ent.Client, provider SMSProvider, fromNumber string) *Service {
	return &Service{
		db:         db,
		provider:   provider,
		fromNumber: fromNumber,
	}
}

// CreateCampaign creates a new SMS campaign
func (s *Service) CreateCampaign(ctx context.Context, userID int, name, messageTemplate string, filters CampaignFilters) (*ent.SMSCampaign, error) {
	// Convert filters to JSON
	filterMap := map[string]interface{}{}
	if filters.Industry != "" {
		filterMap["industry"] = filters.Industry
	}
	if filters.Country != "" {
		filterMap["country"] = filters.Country
	}
	if filters.City != "" {
		filterMap["city"] = filters.City
	}

	// Count target leads
	query := s.db.Lead.Query().Where(lead.PhoneNEQ(""))
	if filters.Industry != "" {
		query = query.Where(lead.IndustryEQ(lead.Industry(filters.Industry)))
	}
	if filters.Country != "" {
		query = query.Where(lead.CountryEQ(filters.Country))
	}
	if filters.City != "" {
		query = query.Where(lead.CityEQ(filters.City))
	}

	totalRecipients, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count recipients: %w", err)
	}

	// Estimate cost ($0.0075 per SMS - Twilio average)
	estimatedCost := float64(totalRecipients) * 0.0075

	campaign, err := s.db.SMSCampaign.
		Create().
		SetUserID(userID).
		SetName(name).
		SetMessageTemplate(messageTemplate).
		SetTargetFilters(filterMap).
		SetTotalRecipients(totalRecipients).
		SetEstimatedCost(estimatedCost).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create campaign: %w", err)
	}

	return campaign, nil
}

// SendCampaign sends an SMS campaign to all recipients
func (s *Service) SendCampaign(ctx context.Context, campaignID int) error {
	// Get campaign
	campaign, err := s.db.SMSCampaign.Get(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("failed to get campaign: %w", err)
	}

	// Check if already sent
	if campaign.Status == smscampaign.StatusSent {
		return ErrCampaignAlreadySent
	}

	// Update status to sending
	campaign, err = s.db.SMSCampaign.
		UpdateOneID(campaignID).
		SetStatus(smscampaign.StatusSending).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update campaign status: %w", err)
	}

	// Get target leads based on filters
	filters := campaign.TargetFilters
	query := s.db.Lead.Query().Where(lead.PhoneNEQ(""))

	if industry, ok := filters["industry"].(string); ok && industry != "" {
		query = query.Where(lead.IndustryEQ(lead.Industry(industry)))
	}
	if country, ok := filters["country"].(string); ok && country != "" {
		query = query.Where(lead.CountryEQ(country))
	}
	if city, ok := filters["city"].(string); ok && city != "" {
		query = query.Where(lead.CityEQ(city))
	}

	leads, err := query.All(ctx)
	if err != nil {
		return fmt.Errorf("failed to get leads: %w", err)
	}

	// Send to each lead
	var totalCost float64
	var sentCount, failedCount int

	for _, l := range leads {
		// Personalize message (replace {name} with lead name)
		message := strings.ReplaceAll(campaign.MessageTemplate, "{name}", l.Name)

		// Create message record
		msg, err := s.db.SMSMessage.
			Create().
			SetCampaignID(campaignID).
			SetLeadID(l.ID).
			SetPhoneNumber(l.Phone).
			SetMessageBody(message).
			Save(ctx)

		if err != nil {
			failedCount++
			continue
		}

		// Send via provider
		result, err := s.provider.SendSMS(ctx, l.Phone, s.fromNumber, message)
		if err != nil {
			// Mark as failed
			s.db.SMSMessage.
				UpdateOneID(msg.ID).
				SetStatus(smsmessage.StatusFailed).
				SetFailedAt(time.Now()).
				SetErrorMessage(err.Error()).
				Save(ctx)
			failedCount++
			continue
		}

		// Update message with result
		s.db.SMSMessage.
			UpdateOneID(msg.ID).
			SetTwilioSid(result.SID).
			SetStatus(smsmessage.StatusSent).
			SetSentAt(time.Now()).
			SetCost(result.Cost).
			Save(ctx)

		totalCost += result.Cost
		sentCount++
	}

	// Update campaign stats
	_, err = s.db.SMSCampaign.
		UpdateOneID(campaignID).
		SetStatus(smscampaign.StatusSent).
		SetSentAt(time.Now()).
		SetSentCount(sentCount).
		SetFailedCount(failedCount).
		SetActualCost(totalCost).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to update campaign stats: %w", err)
	}

	return nil
}

// GetCampaignStats retrieves statistics for a campaign
func (s *Service) GetCampaignStats(ctx context.Context, campaignID int) (*CampaignStats, error) {
	campaign, err := s.db.SMSCampaign.Get(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}

	// Count delivered messages
	deliveredCount, err := s.db.SMSMessage.
		Query().
		Where(
			smsmessage.CampaignIDEQ(campaignID),
			smsmessage.StatusEQ(smsmessage.StatusDelivered),
		).
		Count(ctx)
	if err != nil {
		deliveredCount = 0
	}

	// Calculate delivery rate
	deliveryRate := 0.0
	if campaign.SentCount > 0 {
		deliveryRate = (float64(deliveredCount) / float64(campaign.SentCount)) * 100
	}

	return &CampaignStats{
		CampaignID:      campaign.ID,
		Name:            campaign.Name,
		Status:          string(campaign.Status),
		TotalRecipients: campaign.TotalRecipients,
		SentCount:       campaign.SentCount,
		DeliveredCount:  deliveredCount,
		FailedCount:     campaign.FailedCount,
		EstimatedCost:   campaign.EstimatedCost,
		ActualCost:      campaign.ActualCost,
		DeliveryRate:    deliveryRate,
	}, nil
}

// SendSMS sends a single SMS message (for testing or manual sends)
func (s *Service) SendSMS(ctx context.Context, userID int, to, message string) (*ent.SMSMessage, error) {
	// Validate phone number format (E.164)
	if !strings.HasPrefix(to, "+") {
		return nil, ErrInvalidPhoneNumber
	}

	// Create message record
	msg, err := s.db.SMSMessage.
		Create().
		SetPhoneNumber(to).
		SetMessageBody(message).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Send via provider
	result, err := s.provider.SendSMS(ctx, to, s.fromNumber, message)
	if err != nil {
		// Mark as failed
		msg, _ = s.db.SMSMessage.
			UpdateOneID(msg.ID).
			SetStatus(smsmessage.StatusFailed).
			SetFailedAt(time.Now()).
			SetErrorMessage(err.Error()).
			Save(ctx)
		return msg, fmt.Errorf("failed to send SMS: %w", err)
	}

	// Update message with result
	msg, err = s.db.SMSMessage.
		UpdateOneID(msg.ID).
		SetTwilioSid(result.SID).
		SetStatus(smsmessage.StatusSent).
		SetSentAt(time.Now()).
		SetCost(result.Cost).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	return msg, nil
}

// UpdateMessageStatus updates message status from Twilio callback
func (s *Service) UpdateMessageStatus(ctx context.Context, sid, status string, errorCode int, errorMsg string) error {
	// Find message by SID
	msg, err := s.db.SMSMessage.
		Query().
		Where(smsmessage.TwilioSidEQ(sid)).
		Only(ctx)

	if err != nil {
		return fmt.Errorf("failed to find message: %w", err)
	}

	// Map Twilio status to our status
	var msgStatus smsmessage.Status
	switch status {
	case "delivered":
		msgStatus = smsmessage.StatusDelivered
	case "failed", "undelivered":
		msgStatus = smsmessage.StatusFailed
	default:
		msgStatus = smsmessage.StatusSent
	}

	// Update message
	update := s.db.SMSMessage.
		UpdateOneID(msg.ID).
		SetStatus(msgStatus)

	if msgStatus == smsmessage.StatusDelivered {
		update = update.SetDeliveredAt(time.Now())

		// Update campaign delivered count
		if msg.CampaignID != nil {
			campaign, err := s.db.SMSCampaign.Get(ctx, *msg.CampaignID)
			if err == nil {
				s.db.SMSCampaign.
					UpdateOneID(*msg.CampaignID).
					SetDeliveredCount(campaign.DeliveredCount + 1).
					Save(ctx)
			}
		}
	} else if msgStatus == smsmessage.StatusFailed {
		update = update.
			SetFailedAt(time.Now()).
			SetErrorCode(errorCode).
			SetErrorMessage(errorMsg)
	}

	_, err = update.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update message status: %w", err)
	}

	return nil
}
