package activity

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
	mux.HandleFunc("/activities", h.handleActivities)
	mux.HandleFunc("/activities/", h.handleActivitySubroutes)
	mux.HandleFunc("/dashboard", h.handleDashboard)
	return httpjson.WithCORS(mux)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleActivities(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		httpjson.WriteJSON(w, http.StatusOK, ActivityListResponse{Items: h.service.ListActivities()})
	case http.MethodPost:
		var payload ActivityPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		item, err := h.service.CreateActivity(payload)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		httpjson.WriteJSON(w, http.StatusCreated, item)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleActivitySubroutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/activities/")
	if !strings.HasSuffix(path, "/rsvps") {
		http.NotFound(w, r)
		return
	}

	activityID := strings.TrimSuffix(strings.TrimSuffix(path, "/rsvps"), "/")
	if activityID == "" {
		http.NotFound(w, r)
		return
	}

	var request RSVPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	item, err := h.service.RecordRSVP(activityID, request)
	if err != nil {
		switch {
		case errors.Is(err, ErrActivityNotFound):
			httpjson.WriteError(w, http.StatusNotFound, err.Error())
		default:
			httpjson.WriteError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) handleDashboard(w http.ResponseWriter, _ *http.Request) {
	httpjson.WriteJSON(w, http.StatusOK, h.service.Dashboard())
}
