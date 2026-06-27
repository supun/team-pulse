package activity

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCreateListAndRSVPFlow(t *testing.T) {
	handler := NewHandler(NewService(""))

	createPayload := ActivityPayload{
		TeamID:          "team-pulse",
		Title:           "Volunteer Cleanup",
		Kind:            KindVolunteer,
		StartsAt:        time.Now().UTC().Add(72 * time.Hour),
		TimeZone:        "Europe/Oslo",
		Location:        "North Field",
		MaxParticipants: 12,
		Notes:           "Bring gloves.",
	}
	body, _ := json.Marshal(createPayload)

	createReq := httptest.NewRequest(http.MethodPost, "/activities", bytes.NewReader(body))
	createRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createRes.Code)
	}

	var created ActivityResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if created.SeriesID == "" {
		t.Fatal("expected created activity to belong to a series")
	}

	rsvpBody, _ := json.Marshal(RSVPRequest{MemberName: "Jamie", Status: RSVPGoing})
	rsvpReq := httptest.NewRequest(http.MethodPost, "/activities/"+created.ID+"/rsvps", bytes.NewReader(rsvpBody))
	rsvpRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rsvpRes, rsvpReq)
	if rsvpRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rsvpRes.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/activities", nil)
	listRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(listRes, listReq)

	var list ActivityListResponse
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	found := false
	for _, item := range list.Items {
		if item.ID == created.ID {
			found = true
			if item.GoingCount != 1 {
				t.Fatalf("expected 1 going, got %d", item.GoingCount)
			}
		}
	}
	if !found {
		t.Fatal("created activity missing from list response")
	}
}

func TestVersionedEventsSupportPaginationAndCaching(t *testing.T) {
	handler := NewHandler(NewService(""))

	createPayload := EventSeriesPayload{
		TeamID:          "team-pulse",
		Title:           "Recurring Training",
		Kind:            KindTraining,
		StartsAt:        time.Date(2026, 7, 1, 18, 0, 0, 0, time.UTC),
		TimeZone:        "Europe/Oslo",
		Location:        "Central Arena",
		MaxParticipants: 18,
		Notes:           "Weekly block.",
		Recurrence: &Recurrence{
			Frequency: "weekly",
			Interval:  1,
			Count:     3,
		},
	}
	body, _ := json.Marshal(createPayload)

	createReq := httptest.NewRequest(http.MethodPost, "/v1/event-series", bytes.NewReader(body))
	createRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createRes.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/events?limit=2&include=dashboard&teamId=team-pulse&from=2026-07-01T00:00:00Z", nil)
	listRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRes.Code)
	}
	if listRes.Header().Get("ETag") == "" {
		t.Fatal("expected ETag header")
	}
	if listRes.Header().Get("X-API-Version") != "1" {
		t.Fatalf("expected version header, got %q", listRes.Header().Get("X-API-Version"))
	}

	var list ActivityListResponse
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list.Items))
	}
	if list.NextCursor == "" {
		t.Fatal("expected next cursor for paginated response")
	}
	if list.Dashboard == nil {
		t.Fatal("expected dashboard in combined response")
	}

	cachedReq := httptest.NewRequest(http.MethodGet, "/v1/events?limit=2&include=dashboard&teamId=team-pulse&from=2026-07-01T00:00:00Z", nil)
	cachedReq.Header.Set("If-None-Match", listRes.Header().Get("ETag"))
	cachedRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(cachedRes, cachedReq)
	if cachedRes.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", cachedRes.Code)
	}

	differentFilterReq := httptest.NewRequest(http.MethodGet, "/v1/events?limit=2&include=dashboard&teamId=team-pulse&from=2026-07-08T00:00:00Z", nil)
	differentFilterReq.Header.Set("If-None-Match", listRes.Header().Get("ETag"))
	differentFilterRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(differentFilterRes, differentFilterReq)
	if differentFilterRes.Code == http.StatusNotModified {
		t.Fatal("expected different filters to produce a different cache key")
	}
}

func TestOccurrencePatchAndSeriesSplitFlow(t *testing.T) {
	handler := NewHandler(NewService(""))

	createPayload := EventSeriesPayload{
		TeamID:          "team-pulse",
		Title:           "Academy Training",
		Kind:            KindTraining,
		StartsAt:        time.Date(2026, 7, 1, 18, 0, 0, 0, time.UTC),
		TimeZone:        "Europe/Oslo",
		Location:        "Field One",
		MaxParticipants: 20,
		Notes:           "Core session.",
		Recurrence:      &Recurrence{Frequency: "weekly", Interval: 1, Count: 4},
	}
	body, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/v1/event-series", bytes.NewReader(body))
	createRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createRes.Code)
	}

	var created EventSeriesResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal series response: %v", err)
	}
	if len(created.Occurrences) != 4 {
		t.Fatalf("expected 4 occurrences, got %d", len(created.Occurrences))
	}

	patchBody := `{"location":"Indoor Hall","status":"rescheduled"}`
	patchReq := httptest.NewRequest(http.MethodPatch, "/v1/events/"+created.Occurrences[1].ID, strings.NewReader(patchBody))
	patchRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(patchRes, patchReq)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("expected patch 200, got %d", patchRes.Code)
	}

	var patched ActivityResponse
	if err := json.Unmarshal(patchRes.Body.Bytes(), &patched); err != nil {
		t.Fatalf("unmarshal patched occurrence: %v", err)
	}
	if !patched.IsException || patched.Location != "Indoor Hall" {
		t.Fatalf("expected exception occurrence with updated location, got %+v", patched)
	}

	splitBody := `{"occurrenceId":"` + created.Occurrences[2].ID + `","patch":{"title":"Academy Summer Block","location":"South Campus"}}`
	splitReq := httptest.NewRequest(http.MethodPost, "/v1/event-series/"+created.ID+"/split", strings.NewReader(splitBody))
	splitRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(splitRes, splitReq)
	if splitRes.Code != http.StatusCreated {
		t.Fatalf("expected split 201, got %d", splitRes.Code)
	}

	var splitSeries EventSeriesResponse
	if err := json.Unmarshal(splitRes.Body.Bytes(), &splitSeries); err != nil {
		t.Fatalf("unmarshal split series: %v", err)
	}
	if splitSeries.Title != "Academy Summer Block" {
		t.Fatalf("expected new title after split, got %q", splitSeries.Title)
	}
	if splitSeries.OccurrenceCount != 2 {
		t.Fatalf("expected 2 occurrences in new split series, got %d", splitSeries.OccurrenceCount)
	}

	invalidSplitReq := httptest.NewRequest(http.MethodPost, "/v1/event-series/"+created.ID+"/split", strings.NewReader(`{"occurrenceId":"`+created.Occurrences[2].ID+`"}`))
	invalidSplitRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(invalidSplitRes, invalidSplitReq)
	if invalidSplitRes.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when patch is missing, got %d", invalidSplitRes.Code)
	}
}

func TestInvitationEndpointUsesIdempotencyKey(t *testing.T) {
	handler := NewHandler(NewService(""))

	createPayload := ActivityPayload{
		TeamID:          "team-pulse",
		Title:           "Invite Test",
		Kind:            KindMatch,
		StartsAt:        time.Now().UTC().Add(24 * time.Hour),
		TimeZone:        "UTC",
		Location:        "North Arena",
		MaxParticipants: 16,
		Notes:           "Invite flow.",
	}
	body, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/activities", bytes.NewReader(body))
	createRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(createRes, createReq)

	var created ActivityResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}

	invitationBody := `{"channel":"push","recipients":["mia@example.com"],"message":"Match tomorrow"}`
	firstReq := httptest.NewRequest(http.MethodPost, "/activities/"+created.ID+"/invitations", strings.NewReader(invitationBody))
	firstReq.Header.Set("Idempotency-Key", "invite-001")
	firstRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(firstRes, firstReq)
	if firstRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", firstRes.Code)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/activities/"+created.ID+"/invitations", strings.NewReader(invitationBody))
	secondReq.Header.Set("Idempotency-Key", "invite-001")
	secondRes := httptest.NewRecorder()
	handler.Routes().ServeHTTP(secondRes, secondReq)
	if secondRes.Code != http.StatusCreated {
		t.Fatalf("expected 201 on duplicate idempotent request, got %d", secondRes.Code)
	}

	var first InvitationResponse
	if err := json.Unmarshal(firstRes.Body.Bytes(), &first); err != nil {
		t.Fatalf("unmarshal first invitation: %v", err)
	}
	var second InvitationResponse
	if err := json.Unmarshal(secondRes.Body.Bytes(), &second); err != nil {
		t.Fatalf("unmarshal second invitation: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected duplicate request to return same invitation, got %s and %s", first.ID, second.ID)
	}
}
