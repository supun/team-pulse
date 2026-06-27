package payment

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"teampulse/internal/platform/httpjson"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/subscriptions", h.handleSubscriptions)
	mux.HandleFunc("/subscriptions/", h.handleSubscriptionByTeam)
	mux.HandleFunc("/checkout-sessions", h.handleCheckoutSessions)
	mux.HandleFunc("/webhooks/stripe", h.handleStripeWebhook)
	return httpjson.WithCORS(mux)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"items": h.service.ListSubscriptions()})
}

func (h *Handler) handleSubscriptionByTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	teamID := strings.TrimPrefix(r.URL.Path, "/subscriptions/")
	teamID = strings.TrimSpace(strings.Trim(teamID, "/"))
	if teamID == "" {
		http.NotFound(w, r)
		return
	}

	item, err := h.service.GetSubscription(teamID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionMissing) {
			httpjson.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpjson.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) handleCheckoutSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req CheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	resp, err := h.service.CreateCheckoutSession(req)
	if err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	httpjson.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload StripeWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if payload.Type == "checkout.session.completed" {
		subscription, err := h.service.MarkCheckoutCompleted(payload.Data.Object)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, subscription)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
}
