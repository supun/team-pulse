package activity

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateListAndRSVPFlow(t *testing.T) {
	handler := NewHandler(NewService(""))

	createPayload := ActivityPayload{
		Title:           "Volunteer Cleanup",
		Kind:            KindVolunteer,
		StartsAt:        time.Now().UTC().Add(72 * time.Hour),
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
