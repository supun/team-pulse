package activity

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrActivityNotFound = errors.New("activity not found")
	ErrInvalidRSVP      = errors.New("invalid rsvp status")
)

type record struct {
	ID              string                `json:"id"`
	Title           string                `json:"title"`
	Kind            ActivityKind          `json:"kind"`
	StartsAt        time.Time             `json:"startsAt"`
	Location        string                `json:"location"`
	MaxParticipants int                   `json:"maxParticipants"`
	Notes           string                `json:"notes"`
	RSVPs           map[string]RSVPStatus `json:"rsvps"`
}

type persistedState struct {
	NextID     int       `json:"nextId"`
	Activities []*record `json:"activities"`
}

type Service struct {
	mu         sync.RWMutex
	activities map[string]*record
	nextID     int
	filePath   string
}

func NewService(filePath string) *Service {
	svc := &Service{
		activities: make(map[string]*record),
		filePath:   filePath,
	}

	if err := svc.load(); err == nil && len(svc.activities) > 0 {
		return svc
	}

	now := time.Now().UTC()
	seeds := []ActivityPayload{
		{
			Title:           "Wednesday Recovery Session",
			Kind:            KindTraining,
			StartsAt:        now.Add(24 * time.Hour),
			Location:        "North Arena",
			MaxParticipants: 18,
			Notes:           "Low-intensity mobility and ball retention work.",
		},
		{
			Title:           "Away Match vs. Fjordvik",
			Kind:            KindMatch,
			StartsAt:        now.Add(48 * time.Hour),
			Location:        "Fjordvik Stadium",
			MaxParticipants: 16,
			Notes:           "Bring white kit and meet 45 minutes before kickoff.",
		},
	}

	for _, seed := range seeds {
		item, _ := svc.CreateActivity(seed)
		_, _ = svc.RecordRSVP(item.ID, RSVPRequest{MemberName: "Alex", Status: RSVPGoing})
	}
	_ = svc.persist()

	return svc
}

func (s *Service) CreateActivity(payload ActivityPayload) (*ActivityResponse, error) {
	if err := validatePayload(payload); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	item := &record{
		ID:              fmt.Sprintf("act-%03d", s.nextID),
		Title:           strings.TrimSpace(payload.Title),
		Kind:            payload.Kind,
		StartsAt:        payload.StartsAt.UTC(),
		Location:        strings.TrimSpace(payload.Location),
		MaxParticipants: payload.MaxParticipants,
		Notes:           strings.TrimSpace(payload.Notes),
		RSVPs:           make(map[string]RSVPStatus),
	}
	s.activities[item.ID] = item
	if err := s.persist(); err != nil {
		return nil, err
	}

	resp := toResponse(item)
	return &resp, nil
}

func (s *Service) ListActivities() []ActivityResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]ActivityResponse, 0, len(s.activities))
	for _, activity := range s.activities {
		items = append(items, toResponse(activity))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].StartsAt.Before(items[j].StartsAt)
	})

	return items
}

func (s *Service) RecordRSVP(activityID string, request RSVPRequest) (*ActivityResponse, error) {
	if strings.TrimSpace(request.MemberName) == "" {
		return nil, errors.New("member name is required")
	}
	if request.Status != RSVPGoing && request.Status != RSVPMaybe && request.Status != RSVPDeclined {
		return nil, ErrInvalidRSVP
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.activities[activityID]
	if !ok {
		return nil, ErrActivityNotFound
	}

	item.RSVPs[strings.TrimSpace(request.MemberName)] = request.Status
	if err := s.persist(); err != nil {
		return nil, err
	}
	resp := toResponse(item)
	return &resp, nil
}

func (s *Service) Dashboard() DashboardResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	out := DashboardResponse{
		ByKind: make(map[ActivityKind]int),
	}

	for _, item := range s.activities {
		if item.StartsAt.After(now) {
			out.UpcomingActivities++
		}

		resp := toResponse(item)
		out.ByKind[item.Kind]++
		out.TotalGoing += resp.GoingCount
		if out.NextActivity == nil || resp.StartsAt.Before(out.NextActivity.StartsAt) {
			copy := resp
			out.NextActivity = &copy
		}
	}

	return out
}

func validatePayload(payload ActivityPayload) error {
	switch {
	case strings.TrimSpace(payload.Title) == "":
		return errors.New("title is required")
	case strings.TrimSpace(payload.Location) == "":
		return errors.New("location is required")
	case payload.Kind != KindMatch && payload.Kind != KindTraining && payload.Kind != KindVolunteer && payload.Kind != KindSocial:
		return errors.New("kind must be one of: match, training, volunteer, social")
	case payload.StartsAt.IsZero():
		return errors.New("startsAt is required")
	case payload.MaxParticipants <= 0:
		return errors.New("maxParticipants must be greater than zero")
	default:
		return nil
	}
}

func toResponse(item *record) ActivityResponse {
	out := ActivityResponse{
		ID:              item.ID,
		Title:           item.Title,
		Kind:            item.Kind,
		StartsAt:        item.StartsAt,
		Location:        item.Location,
		MaxParticipants: item.MaxParticipants,
		Notes:           item.Notes,
	}

	for _, status := range item.RSVPs {
		switch status {
		case RSVPGoing:
			out.GoingCount++
		case RSVPMaybe:
			out.MaybeCount++
		case RSVPDeclined:
			out.DeclinedCount++
		}
	}

	if item.MaxParticipants > 0 {
		out.ConfirmedRatio = float64(out.GoingCount) / float64(item.MaxParticipants)
	}

	return out
}

func (s *Service) load() error {
	if s.filePath == "" {
		return nil
	}

	body, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var state persistedState
	if err := json.Unmarshal(body, &state); err != nil {
		return err
	}

	s.nextID = state.NextID
	for _, item := range state.Activities {
		if item.RSVPs == nil {
			item.RSVPs = make(map[string]RSVPStatus)
		}
		s.activities[item.ID] = item
	}
	return nil
}

func (s *Service) persist() error {
	if s.filePath == "" {
		return nil
	}

	items := make([]*record, 0, len(s.activities))
	for _, item := range s.activities {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	state := persistedState{
		NextID:     s.nextID,
		Activities: items,
	}

	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.filePath, body, 0o644)
}
