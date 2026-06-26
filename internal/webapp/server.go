package webapp

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	webRoot    string
	apiBaseURL string
}

func NewServer(webRoot, apiBaseURL string) *Server {
	return &Server{
		webRoot:    webRoot,
		apiBaseURL: strings.TrimRight(apiBaseURL, "/"),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/app-config.js", s.handleConfig)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleFrontend)
	return mux
}

func (s *Server) handleConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	_, _ = fmt.Fprintf(w, "window.__APP_CONFIG__ = { API_BASE_URL: %q };\n", s.apiBaseURL)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleFrontend(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.ServeFile(w, r, filepath.Join(s.webRoot, "index.html"))
		return
	}

	clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	target := filepath.Join(s.webRoot, clean)
	if _, err := os.Stat(target); err == nil {
		http.ServeFile(w, r, target)
		return
	}

	http.NotFound(w, r)
}
