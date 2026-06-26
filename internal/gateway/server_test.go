package gateway

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGatewayProxiesChargeEndpoint(t *testing.T) {
	server := NewServer("http://activity.local", "http://payment.local")
	server.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "http://payment.local/checkout-sessions" {
				t.Fatalf("unexpected url %s", r.URL.String())
			}
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method %s", r.Method)
			}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"receiptId":"rcpt-1001"}`)),
			}, nil
		}),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/checkout-sessions", bytes.NewBufferString(`{"teamId":"team-pulse","teamName":"Pulse United","plan":"pro","successUrl":"http://localhost:3000/success","cancelUrl":"http://localhost:3000/cancel"}`))
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"receiptId":"rcpt-1001"`) {
		t.Fatalf("unexpected response body %s", res.Body.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
