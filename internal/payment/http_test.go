package payment

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckoutSessionAndWebhookFlow(t *testing.T) {
	handler := NewHandler(NewService(&MockStripeClient{}, ""))

	payload := CheckoutSessionRequest{
		TeamID:     "team-pulse",
		TeamName:   "Pulse United",
		Plan:       PlanPro,
		SuccessURL: "http://localhost:3000/billing/success",
		CancelURL:  "http://localhost:3000/billing/cancel",
	}
	body, _ := json.Marshal(payload)

	chargeReq := httptest.NewRequest(http.MethodPost, "/checkout-sessions", bytes.NewReader(body))
	chargeRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(chargeRes, chargeReq)

	if chargeRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", chargeRes.Code)
	}

	webhookReq := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(BuildStripeWebhookEvent("team-pulse", "cus_team-pulse", "cs_test_team-pulse")))
	webhookRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(webhookRes, webhookReq)

	if webhookRes.Code != http.StatusOK {
		t.Fatalf("expected webhook 200, got %d", webhookRes.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/subscriptions/team-pulse", nil)
	getRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(getRes, getReq)

	if getRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRes.Code)
	}

	var sub Subscription
	if err := json.Unmarshal(getRes.Body.Bytes(), &sub); err != nil {
		t.Fatalf("unmarshal subscription: %v", err)
	}
	if sub.Plan != PlanPro {
		t.Fatalf("expected plan pro, got %s", sub.Plan)
	}
	if sub.Status != StatusActive {
		t.Fatalf("expected active status, got %s", sub.Status)
	}
	if sub.StripeCustomerID != "cus_team-pulse" {
		t.Fatalf("expected stripe customer id, got %s", sub.StripeCustomerID)
	}
}
