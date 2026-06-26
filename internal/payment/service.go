package payment

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidPlan         = errors.New("subscription plan must be one of: starter, club, pro")
	ErrTeamIDMissing       = errors.New("teamId is required")
	ErrSubscriptionMissing = errors.New("subscription not found")
	ErrSuccessURLMissing   = errors.New("successUrl is required")
	ErrCancelURLMissing    = errors.New("cancelUrl is required")
)

type persistedState struct {
	Subscriptions []Subscription `json:"subscriptions"`
}

type Service struct {
	mu            sync.RWMutex
	subscriptions map[string]Subscription
	stripe        StripeClient
	filePath      string
}

func NewService(stripe StripeClient, filePath string) *Service {
	svc := &Service{
		subscriptions: make(map[string]Subscription),
		stripe:        stripe,
		filePath:      filePath,
	}

	if err := svc.load(); err == nil && len(svc.subscriptions) > 0 {
		return svc
	}

	now := time.Now().UTC()
	lastPaid := now.Add(-7 * 24 * time.Hour)
	svc.subscriptions["team-demo"] = Subscription{
		TeamID:            "team-demo",
		TeamName:          "TeamPulse FC",
		Plan:              PlanClub,
		Status:            StatusActive,
		PriceMonthlyNOK:   priceForPlan(PlanClub),
		RenewalDate:       now.Add(23 * 24 * time.Hour),
		LastPaymentStatus: "paid",
		LastPaymentAt:     &lastPaid,
		StripeCustomerID:  "cus_team-demo",
	}
	_ = svc.persist()

	return svc
}

func (s *Service) ListSubscriptions() []Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Subscription, 0, len(s.subscriptions))
	for _, item := range s.subscriptions {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].TeamID < items[j].TeamID
	})
	return items
}

func (s *Service) CreateCheckoutSession(req CheckoutSessionRequest) (*CheckoutSessionResponse, error) {
	if strings.TrimSpace(req.TeamID) == "" {
		return nil, ErrTeamIDMissing
	}
	if !validPlan(req.Plan) {
		return nil, ErrInvalidPlan
	}
	if strings.TrimSpace(req.SuccessURL) == "" {
		return nil, ErrSuccessURLMissing
	}
	if strings.TrimSpace(req.CancelURL) == "" {
		return nil, ErrCancelURLMissing
	}

	session, err := s.stripe.CreateCheckoutSession(StripeCheckoutRequest{
		TeamID:     strings.TrimSpace(req.TeamID),
		TeamName:   strings.TrimSpace(req.TeamName),
		Plan:       req.Plan,
		SuccessURL: strings.TrimSpace(req.SuccessURL),
		CancelURL:  strings.TrimSpace(req.CancelURL),
	})
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	subscription := Subscription{
		TeamID:            strings.TrimSpace(req.TeamID),
		TeamName:          strings.TrimSpace(req.TeamName),
		Plan:              req.Plan,
		Status:            StatusTrial,
		PriceMonthlyNOK:   priceForPlan(req.Plan),
		RenewalDate:       time.Now().UTC().Add(30 * 24 * time.Hour),
		LastPaymentStatus: "checkout_pending",
		StripeCustomerID:  session.CustomerID,
		StripeSessionID:   session.SessionID,
		StripeSessionURL:  session.CheckoutURL,
	}
	if subscription.TeamName == "" {
		subscription.TeamName = "Unknown Team"
	}

	s.subscriptions[subscription.TeamID] = subscription
	if err := s.persist(); err != nil {
		return nil, err
	}

	return &CheckoutSessionResponse{
		Subscription: subscription,
		SessionID:    session.SessionID,
		CheckoutURL:  session.CheckoutURL,
		Message:      "Stripe checkout session created successfully.",
	}, nil
}

func (s *Service) GetSubscription(teamID string) (*Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.subscriptions[teamID]
	if !ok {
		return nil, ErrSubscriptionMissing
	}
	return &item, nil
}

func (s *Service) MarkCheckoutCompleted(event StripeCheckoutSession) (*Subscription, error) {
	teamID := strings.TrimSpace(event.ClientReferenceID)
	if teamID == "" {
		return nil, ErrTeamIDMissing
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.subscriptions[teamID]
	if !ok {
		return nil, ErrSubscriptionMissing
	}

	now := time.Now().UTC()
	item.Status = StatusActive
	item.LastPaymentStatus = "paid"
	item.LastPaymentAt = &now
	if strings.TrimSpace(event.Customer) != "" {
		item.StripeCustomerID = strings.TrimSpace(event.Customer)
	}
	if strings.TrimSpace(event.ID) != "" {
		item.StripeSessionID = strings.TrimSpace(event.ID)
	}
	item.StripeSessionURL = ""
	s.subscriptions[teamID] = item
	if err := s.persist(); err != nil {
		return nil, err
	}
	return &item, nil
}

func validPlan(plan SubscriptionPlan) bool {
	return plan == PlanStarter || plan == PlanClub || plan == PlanPro
}

func priceForPlan(plan SubscriptionPlan) int {
	switch plan {
	case PlanStarter:
		return 99
	case PlanClub:
		return 249
	case PlanPro:
		return 499
	default:
		return 0
	}
}

func (s *Service) load() error {
	if s.filePath == "" {
		return nil
	}
	body, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var state persistedState
	if err := json.Unmarshal(body, &state); err != nil {
		return err
	}
	for _, item := range state.Subscriptions {
		s.subscriptions[item.TeamID] = item
	}
	return nil
}

func (s *Service) persist() error {
	if s.filePath == "" {
		return nil
	}

	items := make([]Subscription, 0, len(s.subscriptions))
	for _, item := range s.subscriptions {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].TeamID < items[j].TeamID
	})

	state := persistedState{Subscriptions: items}
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.filePath, body, 0o644)
}
