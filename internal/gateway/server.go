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
	mux.HandleFunc("/api/v1/activities", s.proxyActivities)
	mux.HandleFunc("/api/v1/activities/", s.proxyActivitySubroutes)
	mux.HandleFunc("/api/v1/events", s.proxyEvents)
	mux.HandleFunc("/api/v1/events/", s.proxyEventSubroutes)
	mux.HandleFunc("/api/v1/event-series", s.proxyEventSeries)
	mux.HandleFunc("/api/v1/event-series/", s.proxyEventSeriesSubroutes)
	mux.HandleFunc("/api/v1/dashboard", s.proxyDashboard)
	mux.HandleFunc("/api/v1/subscriptions", s.proxySubscriptions)
	mux.HandleFunc("/api/v1/subscriptions/", s.proxySubscriptionByTeam)
	mux.HandleFunc("/api/v1/checkout-sessions", s.proxyCheckoutSessions)
	mux.HandleFunc("/api/v1/webhooks/stripe", s.proxyStripeWebhooks)
	mux.HandleFunc("/api/activities", s.proxyActivities)
	mux.HandleFunc("/api/activities/", s.proxyActivitySubroutes)
	mux.HandleFunc("/api/events", s.proxyEvents)
	mux.HandleFunc("/api/events/", s.proxyEventSubroutes)
	mux.HandleFunc("/api/event-series", s.proxyEventSeries)
	mux.HandleFunc("/api/event-series/", s.proxyEventSeriesSubroutes)
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
	path := "/activities"
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		path = "/v1/activities"
	}
	s.forwardJSON(w, r, s.activityServiceURL+path)
}

func (s *Server) proxyActivitySubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	s.forwardJSON(w, r, s.activityServiceURL+path)
}

func (s *Server) proxyEvents(w http.ResponseWriter, r *http.Request) {
	path := "/events"
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		path = "/v1/events"
	}
	s.forwardJSON(w, r, s.activityServiceURL+path)
}

func (s *Server) proxyEventSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	s.forwardJSON(w, r, s.activityServiceURL+path)
}

func (s *Server) proxyEventSeries(w http.ResponseWriter, r *http.Request) {
	path := "/event-series"
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		path = "/v1/event-series"
	}
	s.forwardJSON(w, r, s.activityServiceURL+path)
}

func (s *Server) proxyEventSeriesSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	s.forwardJSON(w, r, s.activityServiceURL+path)
}

func (s *Server) proxyDashboard(w http.ResponseWriter, r *http.Request) {
	path := "/dashboard"
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		path = "/v1/dashboard"
	}
	s.forwardJSON(w, r, s.activityServiceURL+path)
}

func (s *Server) proxySubscriptions(w http.ResponseWriter, r *http.Request) {
	path := "/subscriptions"
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		path = "/v1/subscriptions"
	}
	s.forwardJSON(w, r, s.paymentServiceURL+path)
}

func (s *Server) proxySubscriptionByTeam(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	s.forwardJSON(w, r, s.paymentServiceURL+path)
}

func (s *Server) proxyCheckoutSessions(w http.ResponseWriter, r *http.Request) {
	path := "/checkout-sessions"
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		path = "/v1/checkout-sessions"
	}
	s.forwardJSON(w, r, s.paymentServiceURL+path)
}

func (s *Server) proxyStripeWebhooks(w http.ResponseWriter, r *http.Request) {
	path := "/webhooks/stripe"
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		path = "/v1/webhooks/stripe"
	}
	s.forwardJSON(w, r, s.paymentServiceURL+path)
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
	req.URL.RawQuery = r.URL.RawQuery
	req.Header.Set("Content-Type", "application/json")
	copyHeader(r.Header, req.Header, "If-None-Match")
	copyHeader(r.Header, req.Header, "Idempotency-Key")
	copyHeader(r.Header, req.Header, "Accept")

	resp, err := s.client.Do(req)
	if err != nil {
		httpjson.WriteError(w, http.StatusBadGateway, "downstream service unavailable")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	copyHeader(resp.Header, w.Header(), "ETag")
	copyHeader(resp.Header, w.Header(), "Cache-Control")
	copyHeader(resp.Header, w.Header(), "X-API-Version")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func copyHeader(src http.Header, dst http.Header, key string) {
	for name, values := range src {
		if strings.EqualFold(name, key) && len(values) > 0 {
			dst.Set(key, values[0])
			return
		}
	}
}
