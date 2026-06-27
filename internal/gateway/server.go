package gateway

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"teampulse/internal/platform/httpjson"
)

type Server struct {
	activityServiceURL string
	paymentServiceURL  string
	client             *http.Client
}

func NewServer(activityServiceURL, paymentServiceURL string) *Server {
	return &Server{
		activityServiceURL: strings.TrimRight(activityServiceURL, "/"),
		paymentServiceURL:  strings.TrimRight(paymentServiceURL, "/"),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/activities", s.proxyActivities)
	mux.HandleFunc("/api/activities/", s.proxyActivitySubroutes)
	mux.HandleFunc("/api/dashboard", s.proxyDashboard)
	mux.HandleFunc("/api/subscriptions", s.proxySubscriptions)
	mux.HandleFunc("/api/subscriptions/", s.proxySubscriptionByTeam)
	mux.HandleFunc("/api/checkout-sessions", s.proxyCheckoutSessions)
	mux.HandleFunc("/api/webhooks/stripe", s.proxyStripeWebhooks)
	return httpjson.WithCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) proxyActivities(w http.ResponseWriter, r *http.Request) {
	s.forwardJSON(w, r, s.activityServiceURL+"/activities")
}

func (s *Server) proxyActivitySubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	s.forwardJSON(w, r, s.activityServiceURL+path)
}

func (s *Server) proxyDashboard(w http.ResponseWriter, r *http.Request) {
	s.forwardJSON(w, r, s.activityServiceURL+"/dashboard")
}

func (s *Server) proxySubscriptions(w http.ResponseWriter, r *http.Request) {
	s.forwardJSON(w, r, s.paymentServiceURL+"/subscriptions")
}

func (s *Server) proxySubscriptionByTeam(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	s.forwardJSON(w, r, s.paymentServiceURL+path)
}

func (s *Server) proxyCheckoutSessions(w http.ResponseWriter, r *http.Request) {
	s.forwardJSON(w, r, s.paymentServiceURL+"/checkout-sessions")
}

func (s *Server) proxyStripeWebhooks(w http.ResponseWriter, r *http.Request) {
	s.forwardJSON(w, r, s.paymentServiceURL+"/webhooks/stripe")
}

func (s *Server) forwardJSON(w http.ResponseWriter, r *http.Request, targetURL string) {
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "failed to build downstream request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		httpjson.WriteError(w, http.StatusBadGateway, "downstream service unavailable")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
