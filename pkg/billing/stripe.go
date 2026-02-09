package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/organization"
	"github.com/jordanlanch/industrydb/ent/subscription"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/leads"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/stripe/stripe-go/v76"
	billingportalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	stripesubscription "github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"
)

// EmailSender abstracts email sending for billing notifications.
type EmailSender interface {
	SendEmail(toEmail, toName, subject, htmlBody, plainTextBody string) error
}

// AuditLogger abstracts audit logging for billing events.
type AuditLogger interface {
	LogPaymentFailed(userID int, subscriptionID string, metadata map[string]interface{}) error
}

// OrgMembershipChecker abstracts organization membership lookups.
type OrgMembershipChecker interface {
	CheckMembership(ctx context.Context, orgID int, userID int) (isMember bool, role string, err error)
}

// Service handles Stripe billing operations
type Service struct {
	db          *ent.Client
	leadService *leads.Service
	config      *StripeConfig
	email       EmailSender
	audit       AuditLogger
	orgChecker  OrgMembershipChecker
}

// StripeConfig holds Stripe configuration
type StripeConfig struct {
	SecretKey       string
	WebhookSecret   string
	PriceStarter    string
	PricePro        string
	PriceBusiness   string
	SuccessURL      string
	CancelURL       string
	BaseURL         string
}

// NewService creates a new billing service
func NewService(db *ent.Client, leadService *leads.Service, config *StripeConfig) *Service {
	// Set Stripe API key
	stripe.Key = config.SecretKey

	return &Service{
		db:          db,
		leadService: leadService,
		config:      config,
	}
}

// SetEmailSender sets the email sender for billing notifications.
func (s *Service) SetEmailSender(e EmailSender) {
	s.email = e
}

// SetAuditLogger sets the audit logger for billing events.
func (s *Service) SetAuditLogger(a AuditLogger) {
	s.audit = a
}

// SetOrgMembershipChecker sets the organization membership checker.
func (s *Service) SetOrgMembershipChecker(c OrgMembershipChecker) {
	s.orgChecker = c
}

// checkOrgBillingAccess verifies a user has owner or admin role for billing management.
func (s *Service) checkOrgBillingAccess(orgID int, userID int, role string) error {
	if role == "owner" || role == "admin" {
		return nil
	}
	return fmt.Errorf("only organization owner or admin can manage subscriptions")
}

// CreateCheckoutSession creates a Stripe checkout session
// If organizationID is provided, creates subscription for organization instead of user
func (s *Service) CreateCheckoutSession(ctx context.Context, userID int, tier string, organizationID *int) (*models.CheckoutResponse, error) {
	// Get price ID for tier
	priceID, err := s.getPriceIDForTier(tier)
	if err != nil {
		return nil, err
	}

	// Determine if creating subscription for organization or user
	var customerID string
	var email string
	metadata := map[string]string{
		"user_id": fmt.Sprintf("%d", userID),
		"tier":    tier,
	}

	if organizationID != nil {
		// Organization subscription
		org, err := s.db.Organization.Get(ctx, *organizationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get organization: %w", err)
		}

		// Verify user is owner or admin of organization
		if org.OwnerID == userID {
			// Owner always has access
		} else if s.orgChecker != nil {
			isMember, role, err := s.orgChecker.CheckMembership(ctx, *organizationID, userID)
			if err != nil {
				return nil, fmt.Errorf("failed to check membership: %w", err)
			}
			if !isMember {
				return nil, fmt.Errorf("user is not a member of this organization")
			}
			if err := s.checkOrgBillingAccess(*organizationID, userID, role); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("only organization owner or admin can manage subscriptions")
		}

		// Use organization's stripe customer ID or create new
		if org.StripeCustomerID != nil && *org.StripeCustomerID != "" {
			customerID = *org.StripeCustomerID
		} else {
			// Use billing email or owner's email
			if org.BillingEmail != nil && *org.BillingEmail != "" {
				email = *org.BillingEmail
			} else {
				owner, err := org.QueryOwner().Only(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to get owner: %w", err)
				}
				email = owner.Email
			}

			// Create new Stripe customer for organization
			params := &stripe.CustomerParams{
				Email: stripe.String(email),
				Name:  stripe.String(org.Name),
				Metadata: map[string]string{
					"organization_id": fmt.Sprintf("%d", *organizationID),
					"user_id":         fmt.Sprintf("%d", userID),
				},
			}
			cust, err := customer.New(params)
			if err != nil {
				return nil, fmt.Errorf("failed to create customer: %w", err)
			}
			customerID = cust.ID

			// Save customer ID to organization
			_, err = s.db.Organization.UpdateOneID(*organizationID).
				SetStripeCustomerID(customerID).
				Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to save customer ID: %w", err)
			}
		}

		// Add organization_id to metadata for webhook
		metadata["organization_id"] = fmt.Sprintf("%d", *organizationID)
	} else {
		// User subscription
		u, err := s.db.User.Get(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		// Create or get Stripe customer
		if u.StripeCustomerID != nil && *u.StripeCustomerID != "" {
			customerID = *u.StripeCustomerID
		} else {
			// Create new customer
			params := &stripe.CustomerParams{
				Email: stripe.String(u.Email),
				Metadata: map[string]string{
					"user_id": fmt.Sprintf("%d", userID),
				},
			}
			cust, err := customer.New(params)
			if err != nil {
				return nil, fmt.Errorf("failed to create customer: %w", err)
			}
			customerID = cust.ID

			// Save customer ID to user
			_, err = s.db.User.UpdateOneID(userID).
				SetStripeCustomerID(customerID).
				Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to save customer ID: %w", err)
			}
		}
	}

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.config.SuccessURL),
		CancelURL:  stripe.String(s.config.CancelURL),
		Metadata:   metadata,
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return &models.CheckoutResponse{
		SessionID: sess.ID,
		URL:       sess.URL,
		ExpiresAt: sess.ExpiresAt,
	}, nil
}

// CreateCustomerPortalSession creates a Stripe customer portal session
func (s *Service) CreateCustomerPortalSession(ctx context.Context, userID int, returnURL string) (*models.CustomerPortalResponse, error) {
	// Get user
	u, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if u.StripeCustomerID == nil || *u.StripeCustomerID == "" {
		return nil, fmt.Errorf("user has no Stripe customer ID")
	}

	// Create portal session
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(*u.StripeCustomerID),
		ReturnURL: stripe.String(returnURL),
	}

	sess, err := billingportalsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create portal session: %w", err)
	}

	return &models.CustomerPortalResponse{
		URL: sess.URL,
	}, nil
}

// HandleWebhook processes Stripe webhook events
func (s *Service) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	// Verify webhook signature
	event, err := webhook.ConstructEvent(payload, signature, s.config.WebhookSecret)
	if err != nil {
		return fmt.Errorf("webhook signature verification failed: %w", err)
	}

	log.Printf("📨 Stripe webhook received: %s", event.Type)

	// Handle different event types
	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(ctx, event)
	case "customer.subscription.created":
		return s.handleSubscriptionCreated(ctx, event)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(ctx, event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, event)
	case "invoice.paid":
		return s.handleInvoicePaid(ctx, event)
	case "invoice.payment_failed":
		return s.handleInvoicePaymentFailed(ctx, event)
	default:
		log.Printf("⚠️  Unhandled webhook event type: %s", event.Type)
	}

	return nil
}

// handleCheckoutCompleted handles checkout.session.completed event
func (s *Service) handleCheckoutCompleted(ctx context.Context, event stripe.Event) error {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Get user ID from metadata
	userIDStr, ok := sess.Metadata["user_id"]
	if !ok {
		return fmt.Errorf("user_id not found in metadata")
	}

	var userID int
	fmt.Sscanf(userIDStr, "%d", &userID)

	tier := sess.Metadata["tier"]

	// Check if this is an organization subscription
	if orgIDStr, hasOrg := sess.Metadata["organization_id"]; hasOrg {
		var orgID int
		fmt.Sscanf(orgIDStr, "%d", &orgID)

		log.Printf("✅ Organization checkout completed: org_id=%d, user_id=%d, tier=%s, subscription=%s", orgID, userID, tier, sess.Subscription.ID)

		// Update organization subscription tier
		org, err := s.db.Organization.UpdateOneID(orgID).
			SetSubscriptionTier(organization.SubscriptionTier(tier)).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update organization tier: %w", err)
		}

		// Update organization usage limit based on tier
		limit := s.getUsageLimitForTier(tier)
		_, err = s.db.Organization.UpdateOneID(orgID).
			SetUsageLimit(limit).
			Save(ctx)
		if err != nil {
			log.Printf("⚠️  Failed to update organization usage limit: %v", err)
		}

		log.Printf("✅ Organization %s upgraded to %s tier with %d leads/month", org.Name, tier, limit)

		// Create subscription record associated with organization
		// Note: Subscription schema may need organization_id field
		_, err = s.db.Subscription.Create().
			SetUserID(userID). // Keep user ID for tracking
			SetTier(subscription.Tier(tier)).
			SetStatus(subscription.StatusActive).
			SetStripeSubscriptionID(sess.Subscription.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create organization subscription: %w", err)
		}
	} else {
		// User subscription (original behavior)
		log.Printf("✅ User checkout completed: user_id=%d, tier=%s, subscription=%s", userID, tier, sess.Subscription.ID)

		// Update user subscription tier
		_, err := s.db.User.UpdateOneID(userID).
			SetSubscriptionTier(user.SubscriptionTier(tier)).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update user tier: %w", err)
		}

		// Update usage limit based on tier
		if err := s.leadService.UpdateUsageLimitFromTier(ctx, userID); err != nil {
			log.Printf("⚠️  Failed to update usage limit: %v", err)
		}

		// Create subscription record
		_, err = s.db.Subscription.Create().
			SetUserID(userID).
			SetTier(subscription.Tier(tier)).
			SetStatus(subscription.StatusActive).
			SetStripeSubscriptionID(sess.Subscription.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create subscription: %w", err)
		}
	}

	return nil
}

// handleSubscriptionCreated handles customer.subscription.created event
func (s *Service) handleSubscriptionCreated(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to unmarshal subscription: %w", err)
	}

	log.Printf("📝 Subscription created: %s", sub.ID)
	return nil
}

// handleSubscriptionUpdated handles customer.subscription.updated event
func (s *Service) handleSubscriptionUpdated(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to unmarshal subscription: %w", err)
	}

	log.Printf("🔄 Subscription updated: %s, status=%s", sub.ID, sub.Status)

	// Find subscription by Stripe ID
	entSub, err := s.db.Subscription.Query().
		Where(subscription.StripeSubscriptionIDEQ(sub.ID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Printf("⚠️  Subscription not found in DB: %s", sub.ID)
			return nil
		}
		return fmt.Errorf("failed to find subscription: %w", err)
	}

	// Update subscription status
	status := subscription.StatusActive
	switch sub.Status {
	case stripe.SubscriptionStatusActive:
		status = subscription.StatusActive
	case stripe.SubscriptionStatusCanceled:
		status = subscription.StatusCanceled
	case stripe.SubscriptionStatusPastDue:
		status = subscription.StatusPastDue
	case stripe.SubscriptionStatusUnpaid:
		status = subscription.StatusUnpaid
	}

	_, err = s.db.Subscription.UpdateOne(entSub).
		SetStatus(status).
		SetCurrentPeriodStart(time.Unix(sub.CurrentPeriodStart, 0)).
		SetCurrentPeriodEnd(time.Unix(sub.CurrentPeriodEnd, 0)).
		SetCancelAtPeriodEnd(sub.CancelAtPeriodEnd).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// Send email notification based on status change
	if s.email != nil {
		u, err := s.db.User.Get(ctx, entSub.UserID)
		if err != nil {
			log.Printf("⚠️  Failed to get user for email notification: %v", err)
			return nil
		}

		switch sub.Status {
		case stripe.SubscriptionStatusActive:
			subject, html, plain := buildSubscriptionActivatedEmail(u.Name, string(entSub.Tier), s.config.BaseURL)
			if err := s.email.SendEmail(u.Email, u.Name, subject, html, plain); err != nil {
				log.Printf("⚠️  Failed to send activation email to %s: %v", u.Email, err)
			}
		case stripe.SubscriptionStatusPastDue:
			subject, html, plain := buildPaymentFailedEmail(u.Name, s.config.BaseURL)
			if err := s.email.SendEmail(u.Email, u.Name, subject, html, plain); err != nil {
				log.Printf("⚠️  Failed to send payment failed email to %s: %v", u.Email, err)
			}
		}
	}

	return nil
}

// handleSubscriptionDeleted handles customer.subscription.deleted event
func (s *Service) handleSubscriptionDeleted(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to unmarshal subscription: %w", err)
	}

	log.Printf("❌ Subscription deleted: %s", sub.ID)

	// Find subscription
	entSub, err := s.db.Subscription.Query().
		Where(subscription.StripeSubscriptionIDEQ(sub.ID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to find subscription: %w", err)
	}

	// Update subscription status
	_, err = s.db.Subscription.UpdateOne(entSub).
		SetStatus(subscription.StatusCanceled).
		SetCanceledAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// Downgrade user to free tier
	u, err := s.db.User.UpdateOneID(entSub.UserID).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageLimit(50).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to downgrade user: %w", err)
	}

	// Send cancellation email notification
	if s.email != nil {
		subject, html, plain := buildSubscriptionCancelledEmail(u.Name, s.config.BaseURL)
		if err := s.email.SendEmail(u.Email, u.Name, subject, html, plain); err != nil {
			log.Printf("⚠️  Failed to send cancellation email to %s: %v", u.Email, err)
		}
	}

	return nil
}

// handleInvoicePaid handles invoice.paid event
func (s *Service) handleInvoicePaid(ctx context.Context, event stripe.Event) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return fmt.Errorf("failed to unmarshal invoice: %w", err)
	}

	log.Printf("💰 Invoice paid: %s, amount=%d", invoice.ID, invoice.AmountPaid)

	// Send renewal email notification for recurring invoices (not the first one)
	if s.email != nil && invoice.Subscription != nil && invoice.Subscription.ID != "" && invoice.BillingReason == stripe.InvoiceBillingReasonSubscriptionCycle {
		entSub, err := s.db.Subscription.Query().
			Where(subscription.StripeSubscriptionIDEQ(invoice.Subscription.ID)).
			Only(ctx)
		if err == nil {
			u, err := s.db.User.Get(ctx, entSub.UserID)
			if err == nil {
				nextBilling := time.Unix(invoice.PeriodEnd, 0).Format("2006-01-02")
				subject, html, plain := buildSubscriptionRenewedEmail(u.Name, string(entSub.Tier), nextBilling, s.config.BaseURL)
				if err := s.email.SendEmail(u.Email, u.Name, subject, html, plain); err != nil {
					log.Printf("⚠️  Failed to send renewal email to %s: %v", u.Email, err)
				}
			}
		}
	}

	return nil
}

// handleInvoicePaymentFailed handles invoice.payment_failed event
func (s *Service) handleInvoicePaymentFailed(ctx context.Context, event stripe.Event) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return fmt.Errorf("failed to unmarshal invoice: %w", err)
	}

	log.Printf("⚠️  Invoice payment failed: %s", invoice.ID)

	// Find subscription by Stripe subscription ID from the invoice
	if invoice.Subscription == nil || invoice.Subscription.ID == "" {
		log.Printf("⚠️  No subscription ID in failed invoice %s", invoice.ID)
		return nil
	}

	entSub, err := s.db.Subscription.Query().
		Where(subscription.StripeSubscriptionIDEQ(invoice.Subscription.ID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Printf("⚠️  Subscription not found for failed invoice: %s", invoice.Subscription.ID)
			return nil
		}
		return fmt.Errorf("failed to find subscription: %w", err)
	}

	// Update subscription status to past_due
	_, err = s.db.Subscription.UpdateOne(entSub).
		SetStatus(subscription.StatusPastDue).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update subscription to past_due: %w", err)
	}

	log.Printf("🔄 Subscription %s set to past_due due to payment failure", invoice.Subscription.ID)

	// Get user for email notification
	u, err := s.db.User.Get(ctx, entSub.UserID)
	if err != nil {
		log.Printf("⚠️  Failed to get user %d for payment failed notification: %v", entSub.UserID, err)
		return nil
	}

	// Send payment failed email notification
	if s.email != nil {
		subject, html, plain := buildPaymentFailedEmail(u.Name, s.config.BaseURL)
		if err := s.email.SendEmail(u.Email, u.Name, subject, html, plain); err != nil {
			log.Printf("⚠️  Failed to send payment failed email to %s: %v", u.Email, err)
		}
	}

	// Log audit event
	if s.audit != nil {
		metadata := map[string]interface{}{
			"invoice_id":      invoice.ID,
			"subscription_id": invoice.Subscription.ID,
			"amount_due":      invoice.AmountDue,
		}
		if err := s.audit.LogPaymentFailed(entSub.UserID, invoice.Subscription.ID, metadata); err != nil {
			log.Printf("⚠️  Failed to log payment failed audit: %v", err)
		}
	}

	return nil
}

// getPriceIDForTier returns the Stripe price ID for a tier
func (s *Service) getPriceIDForTier(tier string) (string, error) {
	switch tier {
	case "starter":
		return s.config.PriceStarter, nil
	case "pro":
		return s.config.PricePro, nil
	case "business":
		return s.config.PriceBusiness, nil
	default:
		return "", fmt.Errorf("invalid tier: %s", tier)
	}
}

// getUsageLimitForTier returns the usage limit for a subscription tier
func (s *Service) getUsageLimitForTier(tier string) int {
	switch tier {
	case "free":
		return 50
	case "starter":
		return 500
	case "pro":
		return 2000
	case "business":
		return 10000
	default:
		return 50 // Default to free tier limit
	}
}

// GetPricing returns pricing information for all tiers
func (s *Service) GetPricing() *models.PricingResponse {
	return &models.PricingResponse{
		Tiers: []models.PricingTier{
			{
				Name:        "free",
				Price:       0,
				LeadsLimit:  50,
				Description: "Perfect for trying out the platform",
				Features: []string{
					"50 leads per month",
					"Basic data fields",
					"CSV export",
				},
			},
			{
				Name:        "starter",
				Price:       49,
				LeadsLimit:  500,
				Description: "Great for small businesses",
				Features: []string{
					"500 leads per month",
					"Phone & Address included",
					"CSV & Excel export",
					"Email support",
				},
			},
			{
				Name:        "pro",
				Price:       149,
				LeadsLimit:  2000,
				Description: "For growing businesses",
				Features: []string{
					"2,000 leads per month",
					"Email & Social media included",
					"Priority export",
					"Priority support",
				},
			},
			{
				Name:        "business",
				Price:       349,
				LeadsLimit:  10000,
				Description: "For large organizations",
				Features: []string{
					"10,000 leads per month",
					"Full data access",
					"API access",
					"Dedicated support",
					"Custom integrations",
				},
			},
		},
	}
}

// CancelUserSubscriptions cancels all active Stripe subscriptions for a user
// This is called when a user deletes their account (GDPR compliance)
func (s *Service) CancelUserSubscriptions(ctx context.Context, userID int) error {
	// Find all active subscriptions for this user in database
	subscriptions, err := s.db.Subscription.Query().
		Where(
			subscription.UserIDEQ(userID),
			subscription.StatusIn(
				subscription.StatusActive,
				subscription.StatusTrialing,
			),
		).
		All(ctx)

	if err != nil {
		return fmt.Errorf("failed to query subscriptions: %w", err)
	}

	if len(subscriptions) == 0 {
		return nil // No active subscriptions to cancel
	}

	// Cancel each subscription in Stripe
	for _, sub := range subscriptions {
		if sub.StripeSubscriptionID == "" {
			continue // No Stripe subscription ID
		}

		// Cancel subscription in Stripe immediately using subscription package
		params := &stripe.SubscriptionCancelParams{
			InvoiceNow: stripe.Bool(false), // Don't create final invoice
			Prorate:    stripe.Bool(false), // No prorating on cancellation
		}

		_, err := stripesubscription.Cancel(sub.StripeSubscriptionID, params)
		if err != nil {
			log.Printf("❌ Failed to cancel Stripe subscription %s: %v", sub.StripeSubscriptionID, err)
			// Continue canceling other subscriptions even if one fails
			continue
		}

		// Update subscription status in database
		_, err = sub.Update().
			SetStatus(subscription.StatusCanceled).
			SetCanceledAt(time.Now()).
			Save(ctx)

		if err != nil {
			log.Printf("⚠️  Failed to update subscription status in database: %v", err)
		}

		log.Printf("✅ Canceled Stripe subscription %s for deleted account", sub.StripeSubscriptionID)
	}

	return nil
}
