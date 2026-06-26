package webapp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConfigEndpoint(t *testing.T) {
	server := NewServer("../../web", "http://localhost:8080")

	req := httptest.NewRequest(http.MethodGet, "/app-config.js", nil)
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `API_BASE_URL: "http://localhost:8080"`) {
		t.Fatalf("unexpected config body %s", res.Body.String())
	}
}
