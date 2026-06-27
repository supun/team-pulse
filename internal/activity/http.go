package activity

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	mux.HandleFunc("/events", h.handleEvents)
	mux.HandleFunc("/events/", h.handleEventsSubroutes)
	mux.HandleFunc("/event-series", h.handleSeriesCollection)
	mux.HandleFunc("/event-series/", h.handleSeriesSubroutes)
	mux.HandleFunc("/v1/activities", h.handleActivities)
	mux.HandleFunc("/v1/activities/", h.handleActivitySubroutes)
	mux.HandleFunc("/v1/dashboard", h.handleDashboard)
	mux.HandleFunc("/v1/events", h.handleEvents)
	mux.HandleFunc("/v1/events/", h.handleEventsSubroutes)
	mux.HandleFunc("/v1/event-series", h.handleSeriesCollection)
	mux.HandleFunc("/v1/event-series/", h.handleSeriesSubroutes)
	return httpjson.WithCORS(mux)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleActivities(w http.ResponseWriter, r *http.Request) {
	h.handleEvents(w, r)
}

func (h *Handler) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-API-Version", "1")

	switch r.Method {
	case http.MethodGet:
		limit, cursor, includeDashboard, teamID, from, to, err := parseListOptions(r)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		etag, err := h.service.ActivitiesETag(limit, cursor, includeDashboard, teamID, from, to)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		items, nextCursor, err := h.service.ListActivitiesPage(limit, cursor, teamID, from, to)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		response := ActivityListResponse{Items: items, NextCursor: nextCursor}
		if includeDashboard {
			dashboard := h.service.Dashboard()
			response.Dashboard = &dashboard
		}

		w.Header().Set("Cache-Control", "private, max-age=30")
		w.Header().Set("ETag", etag)
		httpjson.WriteJSON(w, http.StatusOK, response)
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

func (h *Handler) handleSeriesCollection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-API-Version", "1")

	switch r.Method {
	case http.MethodGet:
		httpjson.WriteJSON(w, http.StatusOK, EventSeriesListResponse{Items: h.service.ListSeries()})
	case http.MethodPost:
		var payload EventSeriesPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		item, err := h.service.CreateSeries(payload)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		httpjson.WriteJSON(w, http.StatusCreated, item)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleSeriesSubroutes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-API-Version", "1")

	path := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/v1"), "/event-series/")
	if path == r.URL.Path {
		path = strings.TrimPrefix(r.URL.Path, "/event-series/")
	}

	seriesID, route := splitSeriesSubroute(path)
	if seriesID == "" {
		http.NotFound(w, r)
		return
	}

	switch {
	case route == "" && r.Method == http.MethodGet:
		item, err := h.service.GetSeries(seriesID)
		if err != nil {
			if errors.Is(err, ErrSeriesNotFound) {
				httpjson.WriteError(w, http.StatusNotFound, err.Error())
				return
			}
			httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, item)
	case route == "" && r.Method == http.MethodPatch:
		var patch EventSeriesPatchRequest
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		item, err := h.service.UpdateSeries(seriesID, patch)
		if err != nil {
			if errors.Is(err, ErrSeriesNotFound) {
				httpjson.WriteError(w, http.StatusNotFound, err.Error())
				return
			}
			httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, item)
	case route == "split" && r.Method == http.MethodPost:
		var request EventSeriesSplitRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		item, err := h.service.SplitSeries(seriesID, request)
		if err != nil {
			switch {
			case errors.Is(err, ErrSeriesNotFound), errors.Is(err, ErrOccurrenceNotFound):
				httpjson.WriteError(w, http.StatusNotFound, err.Error())
			default:
				httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		httpjson.WriteJSON(w, http.StatusCreated, item)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleActivitySubroutes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-API-Version", "1")

	path := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/v1"), "/activities/")
	if path == r.URL.Path {
		path = strings.TrimPrefix(r.URL.Path, "/activities/")
	}

	activityID, route := splitActivitySubroute(path)
	if activityID == "" {
		http.NotFound(w, r)
		return
	}

	switch route {
	case "rsvps":
		h.handleRSVP(w, r, activityID)
	case "invitations":
		h.handleInvitations(w, r, activityID)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleEventsSubroutes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-API-Version", "1")

	path := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/v1"), "/events/")
	if path == r.URL.Path {
		path = strings.TrimPrefix(r.URL.Path, "/events/")
	}
	occurrenceID := strings.Trim(path, "/")
	if occurrenceID == "" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPatch {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var patch EventOccurrencePatchRequest
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	item, err := h.service.UpdateOccurrence(occurrenceID, patch)
	if err != nil {
		if errors.Is(err, ErrOccurrenceNotFound) {
			httpjson.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpjson.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) handleDashboard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("X-API-Version", "1")
	httpjson.WriteJSON(w, http.StatusOK, h.service.Dashboard())
}

func (h *Handler) handleRSVP(w http.ResponseWriter, r *http.Request, activityID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
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

func (h *Handler) handleInvitations(w http.ResponseWriter, r *http.Request, activityID string) {
	switch r.Method {
	case http.MethodGet:
		limit, cursor, _, _, _, _, err := parseListOptions(r)
		if err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		items, nextCursor, err := h.service.ListInvitations(activityID, limit, cursor)
		if err != nil {
			switch {
			case errors.Is(err, ErrActivityNotFound):
				httpjson.WriteError(w, http.StatusNotFound, err.Error())
			default:
				httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			}
			return
		}

		httpjson.WriteJSON(w, http.StatusOK, InvitationListResponse{Items: items, NextCursor: nextCursor})
	case http.MethodPost:
		var payload InvitationPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			httpjson.WriteError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		item, err := h.service.CreateInvitation(activityID, payload, r.Header.Get("Idempotency-Key"))
		if err != nil {
			switch {
			case errors.Is(err, ErrActivityNotFound):
				httpjson.WriteError(w, http.StatusNotFound, err.Error())
			default:
				httpjson.WriteError(w, http.StatusBadRequest, err.Error())
			}
			return
		}

		httpjson.WriteJSON(w, http.StatusCreated, item)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func splitActivitySubroute(path string) (string, string) {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func splitSeriesSubroute(path string) (string, string) {
	path = strings.Trim(path, "/")
	if path == "" {
		return "", ""
	}
	parts := strings.Split(path, "/")
	if len(parts) == 1 {
		return parts[0], ""
	}
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func parseListOptions(r *http.Request) (int, string, bool, string, *time.Time, *time.Time, error) {
	limit := 0
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > 100 {
			return 0, "", false, "", nil, nil, errors.New("limit must be between 1 and 100")
		}
		limit = parsed
	}

	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	includeDashboard := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include")), "dashboard")
	teamID := strings.TrimSpace(r.URL.Query().Get("teamId"))

	var from *time.Time
	if rawFrom := strings.TrimSpace(r.URL.Query().Get("from")); rawFrom != "" {
		parsed, err := time.Parse(time.RFC3339, rawFrom)
		if err != nil {
			return 0, "", false, "", nil, nil, errors.New("from must be RFC3339")
		}
		from = &parsed
	}

	var to *time.Time
	if rawTo := strings.TrimSpace(r.URL.Query().Get("to")); rawTo != "" {
		parsed, err := time.Parse(time.RFC3339, rawTo)
		if err != nil {
			return 0, "", false, "", nil, nil, errors.New("to must be RFC3339")
		}
		to = &parsed
	}

	return limit, cursor, includeDashboard, teamID, from, to, nil
}
