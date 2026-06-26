package payment

import "time"

type SubscriptionPlan string

const (
	PlanStarter SubscriptionPlan = "starter"
	PlanClub    SubscriptionPlan = "club"
	PlanPro     SubscriptionPlan = "pro"
)

type SubscriptionStatus string

const (
	StatusTrial   SubscriptionStatus = "trial"
	StatusActive  SubscriptionStatus = "active"
	StatusPastDue SubscriptionStatus = "past_due"
)

type Subscription struct {
	TeamID            string             `json:"teamId"`
	TeamName          string             `json:"teamName"`
	Plan              SubscriptionPlan   `json:"plan"`
	Status            SubscriptionStatus `json:"status"`
	PriceMonthlyNOK   int                `json:"priceMonthlyNOK"`
	RenewalDate       time.Time          `json:"renewalDate"`
	LastPaymentStatus string             `json:"lastPaymentStatus"`
	LastPaymentAt     *time.Time         `json:"lastPaymentAt"`
	StripeCustomerID  string             `json:"stripeCustomerId,omitempty"`
	StripeSessionID   string             `json:"stripeSessionId,omitempty"`
	StripeSessionURL  string             `json:"stripeSessionUrl,omitempty"`
}

type CheckoutSessionRequest struct {
	TeamID     string           `json:"teamId"`
	TeamName   string           `json:"teamName"`
	Plan       SubscriptionPlan `json:"plan"`
	SuccessURL string           `json:"successUrl"`
	CancelURL  string           `json:"cancelUrl"`
}

type CheckoutSessionResponse struct {
	Subscription Subscription `json:"subscription"`
	SessionID    string       `json:"sessionId"`
	CheckoutURL  string       `json:"checkoutUrl"`
	Message      string       `json:"message"`
}

type StripeWebhookPayload struct {
	Type string            `json:"type"`
	Data StripeWebhookData `json:"data"`
}

type StripeWebhookData struct {
	Object StripeCheckoutSession `json:"object"`
}

type StripeCheckoutSession struct {
	ID                string `json:"id"`
	URL               string `json:"url,omitempty"`
	Customer          string `json:"customer,omitempty"`
	SubscriptionID    string `json:"subscription,omitempty"`
	ClientReferenceID string `json:"client_reference_id,omitempty"`
}
