package payment

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrStripeConfig = errors.New("stripe is not configured for this plan")

type StripeClient interface {
	CreateCheckoutSession(req StripeCheckoutRequest) (*StripeCheckoutResult, error)
}

type StripeCheckoutRequest struct {
	TeamID     string
	TeamName   string
	Plan       SubscriptionPlan
	SuccessURL string
	CancelURL  string
}

type StripeCheckoutResult struct {
	SessionID   string
	CheckoutURL string
	CustomerID  string
}

type StripeAPIClient struct {
	secretKey string
	priceIDs  map[SubscriptionPlan]string
	baseURL   string
	client    *http.Client
}

func NewStripeAPIClient(secretKey string, priceIDs map[SubscriptionPlan]string) *StripeAPIClient {
	return &StripeAPIClient{
		secretKey: strings.TrimSpace(secretKey),
		priceIDs:  priceIDs,
		baseURL:   "https://api.stripe.com/v1",
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (c *StripeAPIClient) CreateCheckoutSession(req StripeCheckoutRequest) (*StripeCheckoutResult, error) {
	priceID := strings.TrimSpace(c.priceIDs[req.Plan])
	if c.secretKey == "" || priceID == "" {
		return nil, ErrStripeConfig
	}

	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("success_url", req.SuccessURL)
	form.Set("cancel_url", req.CancelURL)
	form.Set("client_reference_id", req.TeamID)
	form.Set("metadata[team_id]", req.TeamID)
	form.Set("metadata[team_name]", req.TeamName)
	form.Set("metadata[plan]", string(req.Plan))
	form.Set("line_items[0][price]", priceID)
	form.Set("line_items[0][quantity]", "1")

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq.SetBasicAuth(c.secretKey, "")
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stripe checkout session failed: %s", strings.TrimSpace(string(body)))
	}

	var payload struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Customer string `json:"customer"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	return &StripeCheckoutResult{
		SessionID:   payload.ID,
		CheckoutURL: payload.URL,
		CustomerID:  payload.Customer,
	}, nil
}

type MockStripeClient struct{}

func (m *MockStripeClient) CreateCheckoutSession(req StripeCheckoutRequest) (*StripeCheckoutResult, error) {
	plan := string(req.Plan)
	return &StripeCheckoutResult{
		SessionID:   "cs_test_" + req.TeamID,
		CheckoutURL: fmt.Sprintf("https://checkout.stripe.local/session/%s/%s", req.TeamID, plan),
		CustomerID:  "cus_" + req.TeamID,
	}, nil
}

func BuildStripeWebhookEvent(teamID, customerID, sessionID string) []byte {
	payload := StripeWebhookPayload{
		Type: "checkout.session.completed",
		Data: StripeWebhookData{
			Object: StripeCheckoutSession{
				ID:                sessionID,
				Customer:          customerID,
				ClientReferenceID: teamID,
				SubscriptionID:    "sub_" + teamID,
			},
		},
	}
	body, _ := json.Marshal(payload)
	return body
}
