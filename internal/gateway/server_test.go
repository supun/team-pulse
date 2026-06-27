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

func TestGatewayProxiesVersionedActivityRequestsWithHeaders(t *testing.T) {
	server := NewServer("http://activity.local", "http://payment.local")
	server.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "http://activity.local/v1/events?limit=5" {
				t.Fatalf("unexpected url %s", r.URL.String())
			}
			if r.Header.Get("If-None-Match") != `W/"abc"` {
				t.Fatalf("expected If-None-Match header to be forwarded")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type":  []string{"application/json"},
					"ETag":          []string{`W/"abc"`},
					"X-API-Version": []string{"1"},
					"Cache-Control": []string{"private, max-age=30"},
				},
				Body: io.NopCloser(strings.NewReader(`{"items":[]}`)),
			}, nil
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?limit=5", nil)
	req.Header.Set("If-None-Match", `W/"abc"`)
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if res.Header().Get("ETag") != `W/"abc"` {
		t.Fatalf("expected ETag response header, got %q", res.Header().Get("ETag"))
	}
	if res.Header().Get("X-API-Version") != "1" {
		t.Fatalf("expected version response header, got %q", res.Header().Get("X-API-Version"))
	}
}

func TestGatewayProxiesSeriesSplitRequests(t *testing.T) {
	server := NewServer("http://activity.local", "http://payment.local")
	server.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "http://activity.local/v1/event-series/series-101/split" {
				t.Fatalf("unexpected url %s", r.URL.String())
			}
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method %s", r.Method)
			}
			if r.Header.Get("Idempotency-Key") != "split-001" {
				t.Fatalf("expected Idempotency-Key to be forwarded")
			}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Content-Type": []string{"application/json"}, "X-API-Version": []string{"1"}},
				Body:       io.NopCloser(strings.NewReader(`{"id":"series-202"}`)),
			}, nil
		}),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/event-series/series-101/split", bytes.NewBufferString(`{"occurrenceId":"act-003"}`))
	req.Header.Set("Idempotency-Key", "split-001")
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"id":"series-202"`) {
		t.Fatalf("unexpected response body %s", res.Body.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
