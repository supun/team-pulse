package activity

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultTeamID      = "team-pulse"
	defaultHorizonDays = 180
)

var (
	ErrActivityNotFound   = errors.New("activity not found")
	ErrOccurrenceNotFound = errors.New("occurrence not found")
	ErrSeriesNotFound     = errors.New("event series not found")
	ErrInvalidRSVP        = errors.New("invalid rsvp status")
	ErrInvalidCursor      = errors.New("invalid cursor")
)

type seriesRecord struct {
	ID              string       `json:"id"`
	TeamID          string       `json:"teamId"`
	Title           string       `json:"title"`
	Kind            ActivityKind `json:"kind"`
	StartsAt        time.Time    `json:"startsAt"`
	TimeZone        string       `json:"timeZone"`
	Location        string       `json:"location"`
	MaxParticipants int          `json:"maxParticipants"`
	Notes           string       `json:"notes"`
	Status          string       `json:"status"`
	HorizonDays     int          `json:"horizonDays"`
	Recurrence      *Recurrence  `json:"recurrence,omitempty"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
}

type occurrenceRecord struct {
	ID              string                `json:"id"`
	SeriesID        string                `json:"seriesId,omitempty"`
	TeamID          string                `json:"teamId"`
	OccurrenceIndex int                   `json:"occurrenceIndex,omitempty"`
	Title           string                `json:"title"`
	Kind            ActivityKind          `json:"kind"`
	StartsAt        time.Time             `json:"startsAt"`
	TimeZone        string                `json:"timeZone"`
	Location        string                `json:"location"`
	MaxParticipants int                   `json:"maxParticipants"`
	Notes           string                `json:"notes"`
	Status          string                `json:"status"`
	IsException     bool                  `json:"isException,omitempty"`
	RSVPs           map[string]RSVPStatus `json:"rsvps"`
	Recurrence      *Recurrence           `json:"recurrence,omitempty"`
	UpdatedAt       time.Time             `json:"updatedAt"`
}

type invitationRecord struct {
	ID             string    `json:"id"`
	ActivityID     string    `json:"activityId"`
	Channel        string    `json:"channel"`
	Recipients     []string  `json:"recipients"`
	Message        string    `json:"message"`
	Status         string    `json:"status"`
	IdempotencyKey string    `json:"idempotencyKey,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type persistedState struct {
	NextSeriesID     int                 `json:"nextSeriesId"`
	NextOccurrenceID int                 `json:"nextOccurrenceId"`
	NextInvitationID int                 `json:"nextInvitationId"`
	Series           []*seriesRecord     `json:"series,omitempty"`
	Activities       []*occurrenceRecord `json:"activities"`
	Invitations      []*invitationRecord `json:"invitations,omitempty"`
	IdempotencyKeys  map[string]string   `json:"idempotencyKeys,omitempty"`
}

type Service struct {
	mu               sync.RWMutex
	series           map[string]*seriesRecord
	occurrences      map[string]*occurrenceRecord
	invitations      map[string]*invitationRecord
	invitationIndex  map[string][]string
	idempotencyKeys  map[string]string
	nextSeriesID     int
	nextOccurrenceID int
	nextInvitationID int
	filePath         string
}

func NewService(filePath string) *Service {
	svc := &Service{
		series:          make(map[string]*seriesRecord),
		occurrences:     make(map[string]*occurrenceRecord),
		invitations:     make(map[string]*invitationRecord),
		invitationIndex: make(map[string][]string),
		idempotencyKeys: make(map[string]string),
		filePath:        filePath,
	}

	if err := svc.load(); err == nil && len(svc.occurrences) > 0 {
		return svc
	}

	now := time.Now().UTC()
	seeds := []EventSeriesPayload{
		{
			TeamID:          defaultTeamID,
			Title:           "Wednesday Recovery Session",
			Kind:            KindTraining,
			StartsAt:        now.Add(24 * time.Hour),
			TimeZone:        "Europe/Oslo",
			Location:        "North Arena",
			MaxParticipants: 18,
			Notes:           "Low-intensity mobility and ball retention work.",
			Recurrence:      &Recurrence{Frequency: "weekly", Interval: 1, Count: 6},
		},
		{
			TeamID:          defaultTeamID,
			Title:           "Away Match vs. Fjordvik",
			Kind:            KindMatch,
			StartsAt:        now.Add(48 * time.Hour),
			TimeZone:        "Europe/Oslo",
			Location:        "Fjordvik Stadium",
			MaxParticipants: 16,
			Notes:           "Bring white kit and meet 45 minutes before kickoff.",
		},
	}

	for _, seed := range seeds {
		series, _ := svc.CreateSeries(seed)
		if series.FirstOccurrenceID != "" {
			_, _ = svc.RecordRSVP(series.FirstOccurrenceID, RSVPRequest{MemberName: "Alex", Status: RSVPGoing})
		}
	}
	_ = svc.persist()

	return svc
}

func (s *Service) CreateActivity(payload ActivityPayload) (*ActivityResponse, error) {
	series, err := s.CreateSeries(EventSeriesPayload{
		TeamID:          payload.TeamID,
		Title:           payload.Title,
		Kind:            payload.Kind,
		StartsAt:        payload.StartsAt,
		TimeZone:        payload.TimeZone,
		Location:        payload.Location,
		MaxParticipants: payload.MaxParticipants,
		Notes:           payload.Notes,
		Recurrence:      payload.Recurrence,
	})
	if err != nil {
		return nil, err
	}
	if len(series.Occurrences) == 0 {
		return nil, ErrActivityNotFound
	}
	first := series.Occurrences[0]
	return &first, nil
}

func (s *Service) CreateSeries(payload EventSeriesPayload) (*EventSeriesResponse, error) {
	if err := validateSeriesPayload(payload); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	series, occurrences := s.createSeriesLocked(payload)
	s.series[series.ID] = series
	for _, occurrence := range occurrences {
		s.occurrences[occurrence.ID] = occurrence
	}
	if err := s.persist(); err != nil {
		return nil, err
	}

	resp := s.toSeriesResponseLocked(series.ID, true)
	return &resp, nil
}

func (s *Service) ListSeries() []EventSeriesResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]EventSeriesResponse, 0, len(s.series))
	for id := range s.series {
		out = append(out, s.toSeriesResponseLocked(id, false))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartsAt.Before(out[j].StartsAt)
	})
	return out
}

func (s *Service) GetSeries(seriesID string) (*EventSeriesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.series[seriesID]; !ok {
		return nil, ErrSeriesNotFound
	}
	resp := s.toSeriesResponseLocked(seriesID, true)
	return &resp, nil
}

func (s *Service) UpdateSeries(seriesID string, patch EventSeriesPatchRequest) (*EventSeriesResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	series, ok := s.series[seriesID]
	if !ok {
		return nil, ErrSeriesNotFound
	}
	if err := applySeriesPatch(series, patch); err != nil {
		return nil, err
	}

	s.syncSeriesOccurrencesLocked(series.ID)
	if err := s.persist(); err != nil {
		return nil, err
	}

	resp := s.toSeriesResponseLocked(seriesID, true)
	return &resp, nil
}

func (s *Service) SplitSeries(seriesID string, request EventSeriesSplitRequest) (*EventSeriesResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	series, ok := s.series[seriesID]
	if !ok {
		return nil, ErrSeriesNotFound
	}
	if strings.TrimSpace(request.OccurrenceID) == "" {
		return nil, errors.New("occurrenceId is required")
	}
	if request.Patch == nil {
		return nil, errors.New("patch is required")
	}

	seriesOccurrences := s.seriesOccurrencesLocked(seriesID)
	splitPos := -1
	for idx, occurrence := range seriesOccurrences {
		if occurrence.ID == request.OccurrenceID {
			splitPos = idx
			break
		}
	}
	if splitPos == -1 {
		return nil, ErrOccurrenceNotFound
	}

	splitOccurrence := seriesOccurrences[splitPos]
	newPayload := EventSeriesPayload{
		TeamID:          series.TeamID,
		Title:           series.Title,
		Kind:            series.Kind,
		StartsAt:        splitOccurrence.StartsAt,
		TimeZone:        series.TimeZone,
		Location:        series.Location,
		MaxParticipants: series.MaxParticipants,
		Notes:           series.Notes,
		Recurrence:      cloneRecurrence(series.Recurrence),
		HorizonDays:     series.HorizonDays,
	}

	if newPayload.Recurrence != nil && newPayload.Recurrence.Count > 0 {
		newPayload.Recurrence.Count = len(seriesOccurrences) - splitPos
	}
	if err := applySeriesPatchToPayload(&newPayload, *request.Patch); err != nil {
		return nil, err
	}
	if err := validateSeriesPayload(newPayload); err != nil {
		return nil, err
	}

	for _, occurrence := range seriesOccurrences[splitPos:] {
		delete(s.occurrences, occurrence.ID)
	}

	if series.Recurrence != nil {
		remainingCount := splitPos
		if remainingCount <= 0 {
			series.Status = "archived"
			series.Recurrence.Count = 0
		} else {
			series.Recurrence.Count = remainingCount
		}
	}
	series.UpdatedAt = time.Now().UTC()
	s.syncSeriesOccurrencesLocked(series.ID)

	newSeries, occurrences := s.createSeriesLocked(newPayload)
	s.series[newSeries.ID] = newSeries
	for _, occurrence := range occurrences {
		s.occurrences[occurrence.ID] = occurrence
	}

	if err := s.persist(); err != nil {
		return nil, err
	}

	resp := s.toSeriesResponseLocked(newSeries.ID, true)
	return &resp, nil
}

func (s *Service) UpdateOccurrence(occurrenceID string, patch EventOccurrencePatchRequest) (*ActivityResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	occurrence, ok := s.occurrences[occurrenceID]
	if !ok {
		return nil, ErrOccurrenceNotFound
	}
	if err := applyOccurrencePatch(occurrence, patch); err != nil {
		return nil, err
	}
	occurrence.IsException = true
	occurrence.UpdatedAt = time.Now().UTC()
	if err := s.persist(); err != nil {
		return nil, err
	}

	resp := toActivityResponse(occurrence)
	return &resp, nil
}

func (s *Service) ListActivities() []ActivityResponse {
	items, _, _ := s.ListActivitiesPage(0, "", "", nil, nil)
	return items
}

func (s *Service) ListActivitiesPage(limit int, cursor, teamID string, from, to *time.Time) ([]ActivityResponse, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]ActivityResponse, 0, len(s.occurrences))
	for _, occurrence := range s.occurrences {
		if teamID != "" && occurrence.TeamID != teamID {
			continue
		}
		if from != nil && occurrence.StartsAt.Before(*from) {
			continue
		}
		if to != nil && occurrence.StartsAt.After(*to) {
			continue
		}
		items = append(items, toActivityResponse(occurrence))
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].StartsAt.Equal(items[j].StartsAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].StartsAt.Before(items[j].StartsAt)
	})

	offset := 0
	if cursor != "" {
		parsed, err := strconv.Atoi(cursor)
		if err != nil || parsed < 0 {
			return nil, "", ErrInvalidCursor
		}
		offset = parsed
	}
	if offset > len(items) {
		return nil, "", ErrInvalidCursor
	}

	if limit <= 0 {
		limit = len(items)
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}

	nextCursor := ""
	if end < len(items) {
		nextCursor = strconv.Itoa(end)
	}
	return items[offset:end], nextCursor, nil
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

	item, ok := s.occurrences[activityID]
	if !ok {
		return nil, ErrActivityNotFound
	}

	item.RSVPs[strings.TrimSpace(request.MemberName)] = request.Status
	item.UpdatedAt = time.Now().UTC()
	if err := s.persist(); err != nil {
		return nil, err
	}
	resp := toActivityResponse(item)
	return &resp, nil
}

func (s *Service) CreateInvitation(activityID string, payload InvitationPayload, idempotencyKey string) (*InvitationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.occurrences[activityID]; !ok {
		return nil, ErrActivityNotFound
	}
	if err := validateInvitationPayload(payload); err != nil {
		return nil, err
	}

	key := strings.TrimSpace(idempotencyKey)
	if key != "" {
		scopedKey := activityID + ":" + key
		if invitationID, ok := s.idempotencyKeys[scopedKey]; ok {
			existing := s.invitations[invitationID]
			resp := toInvitationResponse(existing)
			return &resp, nil
		}
	}

	s.nextInvitationID++
	recipients := make([]string, 0, len(payload.Recipients))
	for _, recipient := range payload.Recipients {
		recipients = append(recipients, strings.TrimSpace(recipient))
	}

	item := &invitationRecord{
		ID:             fmt.Sprintf("inv-%03d", s.nextInvitationID),
		ActivityID:     activityID,
		Channel:        strings.TrimSpace(payload.Channel),
		Recipients:     recipients,
		Message:        strings.TrimSpace(payload.Message),
		Status:         "queued",
		IdempotencyKey: key,
		CreatedAt:      time.Now().UTC(),
	}
	s.invitations[item.ID] = item
	s.invitationIndex[activityID] = append(s.invitationIndex[activityID], item.ID)
	if key != "" {
		s.idempotencyKeys[activityID+":"+key] = item.ID
	}

	if err := s.persist(); err != nil {
		return nil, err
	}

	resp := toInvitationResponse(item)
	return &resp, nil
}

func (s *Service) ListInvitations(activityID string, limit int, cursor string) ([]InvitationResponse, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.occurrences[activityID]; !ok {
		return nil, "", ErrActivityNotFound
	}

	ids := append([]string(nil), s.invitationIndex[activityID]...)
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	offset := 0
	if cursor != "" {
		parsed, err := strconv.Atoi(cursor)
		if err != nil || parsed < 0 {
			return nil, "", ErrInvalidCursor
		}
		offset = parsed
	}
	if offset > len(ids) {
		return nil, "", ErrInvalidCursor
	}

	if limit <= 0 {
		limit = len(ids)
	}
	end := offset + limit
	if end > len(ids) {
		end = len(ids)
	}

	out := make([]InvitationResponse, 0, end-offset)
	for _, id := range ids[offset:end] {
		out = append(out, toInvitationResponse(s.invitations[id]))
	}

	nextCursor := ""
	if end < len(ids) {
		nextCursor = strconv.Itoa(end)
	}
	return out, nextCursor, nil
}

func (s *Service) ActivitiesETag(limit int, cursor string, includeDashboard bool, teamID string, from, to *time.Time) (string, error) {
	items, nextCursor, err := s.ListActivitiesPage(limit, cursor, teamID, from, to)
	if err != nil {
		return "", err
	}

	hash := sha1.New()
	_, _ = hash.Write([]byte(teamID))
	if from != nil {
		_, _ = hash.Write([]byte(from.UTC().Format(time.RFC3339Nano)))
	}
	if to != nil {
		_, _ = hash.Write([]byte(to.UTC().Format(time.RFC3339Nano)))
	}
	for _, item := range items {
		_, _ = hash.Write([]byte(item.ID))
		_, _ = hash.Write([]byte(item.StartsAt.UTC().Format(time.RFC3339Nano)))
		_, _ = hash.Write([]byte(fmt.Sprintf("%d%d%d", item.GoingCount, item.MaybeCount, item.DeclinedCount)))
	}
	_, _ = hash.Write([]byte(nextCursor))
	if includeDashboard {
		dashboard := s.Dashboard()
		_, _ = hash.Write([]byte(fmt.Sprintf("%d%d", dashboard.UpcomingActivities, dashboard.TotalGoing)))
		if dashboard.NextActivity != nil {
			_, _ = hash.Write([]byte(dashboard.NextActivity.ID))
		}
	}

	return `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`, nil
}

func (s *Service) Dashboard() DashboardResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	out := DashboardResponse{ByKind: make(map[ActivityKind]int)}
	for _, item := range s.occurrences {
		if item.StartsAt.After(now) {
			out.UpcomingActivities++
		}

		resp := toActivityResponse(item)
		out.ByKind[item.Kind]++
		out.TotalGoing += resp.GoingCount
		if out.NextActivity == nil || resp.StartsAt.Before(out.NextActivity.StartsAt) {
			copy := resp
			out.NextActivity = &copy
		}
	}
	return out
}

func validateSeriesPayload(payload EventSeriesPayload) error {
	timeZone := strings.TrimSpace(payload.TimeZone)
	if timeZone == "" {
		timeZone = "UTC"
	}
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
	case !isValidTimeZone(timeZone):
		return errors.New("timeZone must be a valid IANA time zone")
	case payload.Recurrence != nil && payload.Recurrence.Count < 1:
		return errors.New("recurrence.count must be greater than zero")
	case payload.Recurrence != nil && payload.Recurrence.Count > 52:
		return errors.New("recurrence.count must be 52 or less")
	case payload.Recurrence != nil && payload.Recurrence.Interval < 1:
		return errors.New("recurrence.interval must be greater than zero")
	case payload.Recurrence != nil && payload.Recurrence.Frequency != "daily" && payload.Recurrence.Frequency != "weekly":
		return errors.New("recurrence.frequency must be daily or weekly")
	default:
		return nil
	}
}

func applySeriesPatch(series *seriesRecord, patch EventSeriesPatchRequest) error {
	if patch.TeamID != nil {
		series.TeamID = strings.TrimSpace(*patch.TeamID)
	}
	if patch.Title != nil {
		series.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Kind != nil {
		series.Kind = *patch.Kind
	}
	if patch.StartsAt != nil {
		series.StartsAt = patch.StartsAt.UTC()
	}
	if patch.TimeZone != nil {
		series.TimeZone = strings.TrimSpace(*patch.TimeZone)
	}
	if patch.Location != nil {
		series.Location = strings.TrimSpace(*patch.Location)
	}
	if patch.MaxParticipants != nil {
		series.MaxParticipants = *patch.MaxParticipants
	}
	if patch.Notes != nil {
		series.Notes = strings.TrimSpace(*patch.Notes)
	}
	if patch.Recurrence != nil {
		series.Recurrence = cloneRecurrence(patch.Recurrence)
	}
	if patch.HorizonDays != nil {
		series.HorizonDays = *patch.HorizonDays
	}
	series.UpdatedAt = time.Now().UTC()
	return validateSeriesPayload(seriesToPayload(series))
}

func applySeriesPatchToPayload(payload *EventSeriesPayload, patch EventSeriesPatchRequest) error {
	if patch.TeamID != nil {
		payload.TeamID = strings.TrimSpace(*patch.TeamID)
	}
	if patch.Title != nil {
		payload.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Kind != nil {
		payload.Kind = *patch.Kind
	}
	if patch.StartsAt != nil {
		payload.StartsAt = patch.StartsAt.UTC()
	}
	if patch.TimeZone != nil {
		payload.TimeZone = strings.TrimSpace(*patch.TimeZone)
	}
	if patch.Location != nil {
		payload.Location = strings.TrimSpace(*patch.Location)
	}
	if patch.MaxParticipants != nil {
		payload.MaxParticipants = *patch.MaxParticipants
	}
	if patch.Notes != nil {
		payload.Notes = strings.TrimSpace(*patch.Notes)
	}
	if patch.Recurrence != nil {
		payload.Recurrence = cloneRecurrence(patch.Recurrence)
	}
	if patch.HorizonDays != nil {
		payload.HorizonDays = *patch.HorizonDays
	}
	return nil
}

func applyOccurrencePatch(occurrence *occurrenceRecord, patch EventOccurrencePatchRequest) error {
	if patch.Title != nil {
		occurrence.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Kind != nil {
		occurrence.Kind = *patch.Kind
	}
	if patch.StartsAt != nil {
		occurrence.StartsAt = patch.StartsAt.UTC()
	}
	if patch.TimeZone != nil {
		occurrence.TimeZone = strings.TrimSpace(*patch.TimeZone)
	}
	if patch.Location != nil {
		occurrence.Location = strings.TrimSpace(*patch.Location)
	}
	if patch.MaxParticipants != nil {
		occurrence.MaxParticipants = *patch.MaxParticipants
	}
	if patch.Notes != nil {
		occurrence.Notes = strings.TrimSpace(*patch.Notes)
	}
	if patch.Status != nil {
		occurrence.Status = strings.TrimSpace(*patch.Status)
	}
	return validateSeriesPayload(EventSeriesPayload{
		TeamID:          occurrence.TeamID,
		Title:           occurrence.Title,
		Kind:            occurrence.Kind,
		StartsAt:        occurrence.StartsAt,
		TimeZone:        occurrence.TimeZone,
		Location:        occurrence.Location,
		MaxParticipants: occurrence.MaxParticipants,
		Notes:           occurrence.Notes,
		Recurrence:      occurrence.Recurrence,
	})
}

func validateInvitationPayload(payload InvitationPayload) error {
	channel := strings.TrimSpace(payload.Channel)
	switch {
	case channel != "push" && channel != "email" && channel != "sms":
		return errors.New("channel must be one of: push, email, sms")
	case len(payload.Recipients) == 0:
		return errors.New("at least one recipient is required")
	default:
		for _, recipient := range payload.Recipients {
			if strings.TrimSpace(recipient) == "" {
				return errors.New("recipients cannot contain empty values")
			}
		}
		return nil
	}
}

func cloneRecurrence(in *Recurrence) *Recurrence {
	if in == nil {
		return nil
	}
	copy := *in
	return &copy
}

func toInvitationResponse(item *invitationRecord) InvitationResponse {
	recipients := append([]string(nil), item.Recipients...)
	return InvitationResponse{
		ID:             item.ID,
		ActivityID:     item.ActivityID,
		Channel:        item.Channel,
		Recipients:     recipients,
		Message:        item.Message,
		Status:         item.Status,
		IdempotencyKey: item.IdempotencyKey,
		CreatedAt:      item.CreatedAt,
	}
}

func toActivityResponse(item *occurrenceRecord) ActivityResponse {
	out := ActivityResponse{
		ID:              item.ID,
		SeriesID:        item.SeriesID,
		TeamID:          item.TeamID,
		OccurrenceIndex: item.OccurrenceIndex,
		Title:           item.Title,
		Kind:            item.Kind,
		StartsAt:        item.StartsAt,
		TimeZone:        item.TimeZone,
		Location:        item.Location,
		MaxParticipants: item.MaxParticipants,
		Notes:           item.Notes,
		Status:          item.Status,
		IsException:     item.IsException,
		Recurrence:      cloneRecurrence(item.Recurrence),
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

func isValidTimeZone(name string) bool {
	_, err := time.LoadLocation(name)
	return err == nil
}

func (s *Service) createSeriesLocked(payload EventSeriesPayload) (*seriesRecord, []*occurrenceRecord) {
	normalized := normalizeSeriesPayload(payload)
	s.nextSeriesID++
	now := time.Now().UTC()
	series := &seriesRecord{
		ID:              fmt.Sprintf("series-%03d", s.nextSeriesID),
		TeamID:          normalized.TeamID,
		Title:           normalized.Title,
		Kind:            normalized.Kind,
		StartsAt:        normalized.StartsAt.UTC(),
		TimeZone:        normalized.TimeZone,
		Location:        normalized.Location,
		MaxParticipants: normalized.MaxParticipants,
		Notes:           normalized.Notes,
		Status:          "active",
		HorizonDays:     normalized.HorizonDays,
		Recurrence:      cloneRecurrence(normalized.Recurrence),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	return series, s.generateOccurrencesLocked(series)
}

func normalizeSeriesPayload(payload EventSeriesPayload) EventSeriesPayload {
	if strings.TrimSpace(payload.TeamID) == "" {
		payload.TeamID = defaultTeamID
	}
	if strings.TrimSpace(payload.TimeZone) == "" {
		payload.TimeZone = "UTC"
	}
	if payload.HorizonDays <= 0 {
		payload.HorizonDays = defaultHorizonDays
	}
	payload.Title = strings.TrimSpace(payload.Title)
	payload.Location = strings.TrimSpace(payload.Location)
	payload.Notes = strings.TrimSpace(payload.Notes)
	payload.TeamID = strings.TrimSpace(payload.TeamID)
	payload.TimeZone = strings.TrimSpace(payload.TimeZone)
	return payload
}

func (s *Service) generateOccurrencesLocked(series *seriesRecord) []*occurrenceRecord {
	count := 1
	if series.Recurrence != nil && series.Recurrence.Count > 0 {
		count = series.Recurrence.Count
	}
	occurrences := make([]*occurrenceRecord, 0, count)
	for i := 0; i < count; i++ {
		s.nextOccurrenceID++
		occurrences = append(occurrences, &occurrenceRecord{
			ID:              fmt.Sprintf("act-%03d", s.nextOccurrenceID),
			SeriesID:        series.ID,
			TeamID:          series.TeamID,
			OccurrenceIndex: i + 1,
			Title:           series.Title,
			Kind:            series.Kind,
			StartsAt:        occurrenceStartAt(series, i),
			TimeZone:        series.TimeZone,
			Location:        series.Location,
			MaxParticipants: series.MaxParticipants,
			Notes:           series.Notes,
			Status:          "scheduled",
			RSVPs:           make(map[string]RSVPStatus),
			Recurrence:      cloneRecurrence(series.Recurrence),
			UpdatedAt:       time.Now().UTC(),
		})
	}
	return occurrences
}

func occurrenceStartAt(series *seriesRecord, index int) time.Time {
	startsAt := series.StartsAt.UTC()
	if series.Recurrence == nil {
		return startsAt
	}
	switch series.Recurrence.Frequency {
	case "daily":
		return startsAt.Add(time.Duration(index*series.Recurrence.Interval) * 24 * time.Hour)
	case "weekly":
		return startsAt.Add(time.Duration(index*series.Recurrence.Interval*7) * 24 * time.Hour)
	default:
		return startsAt
	}
}

func (s *Service) seriesOccurrencesLocked(seriesID string) []*occurrenceRecord {
	out := make([]*occurrenceRecord, 0)
	for _, occurrence := range s.occurrences {
		if occurrence.SeriesID == seriesID {
			out = append(out, occurrence)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartsAt.Equal(out[j].StartsAt) {
			return out[i].OccurrenceIndex < out[j].OccurrenceIndex
		}
		return out[i].StartsAt.Before(out[j].StartsAt)
	})
	return out
}

func (s *Service) syncSeriesOccurrencesLocked(seriesID string) {
	series, ok := s.series[seriesID]
	if !ok {
		return
	}

	targetCount := 1
	if series.Status == "archived" {
		targetCount = 0
	} else if series.Recurrence != nil && series.Recurrence.Count > 0 {
		targetCount = series.Recurrence.Count
	}

	current := s.seriesOccurrencesLocked(seriesID)
	for idx := 0; idx < targetCount && idx < len(current); idx++ {
		item := current[idx]
		if item.IsException {
			item.OccurrenceIndex = idx + 1
			continue
		}
		item.TeamID = series.TeamID
		item.OccurrenceIndex = idx + 1
		item.Title = series.Title
		item.Kind = series.Kind
		item.StartsAt = occurrenceStartAt(series, idx)
		item.TimeZone = series.TimeZone
		item.Location = series.Location
		item.MaxParticipants = series.MaxParticipants
		item.Notes = series.Notes
		item.Status = "scheduled"
		item.Recurrence = cloneRecurrence(series.Recurrence)
		item.UpdatedAt = time.Now().UTC()
	}

	if targetCount > len(current) {
		for idx := len(current); idx < targetCount; idx++ {
			s.nextOccurrenceID++
			s.occurrences[fmt.Sprintf("act-%03d", s.nextOccurrenceID)] = &occurrenceRecord{
				ID:              fmt.Sprintf("act-%03d", s.nextOccurrenceID),
				SeriesID:        series.ID,
				TeamID:          series.TeamID,
				OccurrenceIndex: idx + 1,
				Title:           series.Title,
				Kind:            series.Kind,
				StartsAt:        occurrenceStartAt(series, idx),
				TimeZone:        series.TimeZone,
				Location:        series.Location,
				MaxParticipants: series.MaxParticipants,
				Notes:           series.Notes,
				Status:          "scheduled",
				RSVPs:           make(map[string]RSVPStatus),
				Recurrence:      cloneRecurrence(series.Recurrence),
				UpdatedAt:       time.Now().UTC(),
			}
		}
	}

	if targetCount < len(current) {
		for _, item := range current[targetCount:] {
			delete(s.occurrences, item.ID)
		}
	}
}

func (s *Service) toSeriesResponseLocked(seriesID string, includeOccurrences bool) EventSeriesResponse {
	series := s.series[seriesID]
	occurrences := s.seriesOccurrencesLocked(seriesID)
	out := EventSeriesResponse{
		ID:              series.ID,
		TeamID:          series.TeamID,
		Title:           series.Title,
		Kind:            series.Kind,
		StartsAt:        series.StartsAt,
		TimeZone:        series.TimeZone,
		Location:        series.Location,
		MaxParticipants: series.MaxParticipants,
		Notes:           series.Notes,
		Status:          series.Status,
		HorizonDays:     series.HorizonDays,
		Recurrence:      cloneRecurrence(series.Recurrence),
		OccurrenceCount: len(occurrences),
	}
	now := time.Now().UTC()
	for idx, occurrence := range occurrences {
		if idx == 0 {
			out.FirstOccurrenceID = occurrence.ID
		}
		out.LastOccurrenceID = occurrence.ID
		if occurrence.StartsAt.After(now) {
			out.UpcomingCount++
		}
		if includeOccurrences {
			out.Occurrences = append(out.Occurrences, toActivityResponse(occurrence))
		}
	}
	return out
}

func seriesToPayload(series *seriesRecord) EventSeriesPayload {
	return EventSeriesPayload{
		TeamID:          series.TeamID,
		Title:           series.Title,
		Kind:            series.Kind,
		StartsAt:        series.StartsAt,
		TimeZone:        series.TimeZone,
		Location:        series.Location,
		MaxParticipants: series.MaxParticipants,
		Notes:           series.Notes,
		Recurrence:      cloneRecurrence(series.Recurrence),
		HorizonDays:     series.HorizonDays,
	}
}

func (s *Service) migrateLegacySeriesLocked() {
	if len(s.series) > 0 {
		return
	}

	grouped := make(map[string][]*occurrenceRecord)
	for _, occurrence := range s.occurrences {
		key := occurrence.SeriesID
		if key == "" {
			key = occurrence.ID
		}
		grouped[key] = append(grouped[key], occurrence)
	}

	for _, occurrences := range grouped {
		sort.Slice(occurrences, func(i, j int) bool {
			return occurrences[i].StartsAt.Before(occurrences[j].StartsAt)
		})
		first := occurrences[0]
		s.nextSeriesID++
		seriesID := first.SeriesID
		if seriesID == "" {
			seriesID = fmt.Sprintf("series-%03d", s.nextSeriesID)
		}
		recurrence := cloneRecurrence(first.Recurrence)
		if recurrence == nil && len(occurrences) > 1 {
			recurrence = &Recurrence{Frequency: "weekly", Interval: 1, Count: len(occurrences)}
		}
		s.series[seriesID] = &seriesRecord{
			ID:              seriesID,
			TeamID:          first.TeamID,
			Title:           first.Title,
			Kind:            first.Kind,
			StartsAt:        first.StartsAt,
			TimeZone:        first.TimeZone,
			Location:        first.Location,
			MaxParticipants: first.MaxParticipants,
			Notes:           first.Notes,
			Status:          "active",
			HorizonDays:     defaultHorizonDays,
			Recurrence:      recurrence,
			CreatedAt:       first.UpdatedAt,
			UpdatedAt:       first.UpdatedAt,
		}
		for idx, occurrence := range occurrences {
			occurrence.SeriesID = seriesID
			occurrence.OccurrenceIndex = idx + 1
			if occurrence.TeamID == "" {
				occurrence.TeamID = defaultTeamID
			}
		}
	}
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

	s.nextSeriesID = state.NextSeriesID
	s.nextOccurrenceID = state.NextOccurrenceID
	s.nextInvitationID = state.NextInvitationID
	for _, series := range state.Series {
		if series.HorizonDays <= 0 {
			series.HorizonDays = defaultHorizonDays
		}
		if series.TeamID == "" {
			series.TeamID = defaultTeamID
		}
		s.series[series.ID] = series
	}
	for _, occurrence := range state.Activities {
		if occurrence.RSVPs == nil {
			occurrence.RSVPs = make(map[string]RSVPStatus)
		}
		if occurrence.TimeZone == "" {
			occurrence.TimeZone = "UTC"
		}
		if occurrence.TeamID == "" {
			occurrence.TeamID = defaultTeamID
		}
		if occurrence.Status == "" {
			occurrence.Status = "scheduled"
		}
		if occurrence.UpdatedAt.IsZero() {
			occurrence.UpdatedAt = occurrence.StartsAt
		}
		s.occurrences[occurrence.ID] = occurrence
	}
	for _, invitation := range state.Invitations {
		s.invitations[invitation.ID] = invitation
		s.invitationIndex[invitation.ActivityID] = append(s.invitationIndex[invitation.ActivityID], invitation.ID)
	}
	if state.IdempotencyKeys != nil {
		s.idempotencyKeys = state.IdempotencyKeys
	}

	s.migrateLegacySeriesLocked()
	return nil
}

func (s *Service) persist() error {
	if s.filePath == "" {
		return nil
	}

	seriesItems := make([]*seriesRecord, 0, len(s.series))
	for _, item := range s.series {
		seriesItems = append(seriesItems, item)
	}
	sort.Slice(seriesItems, func(i, j int) bool {
		return seriesItems[i].ID < seriesItems[j].ID
	})

	occurrenceItems := make([]*occurrenceRecord, 0, len(s.occurrences))
	for _, item := range s.occurrences {
		occurrenceItems = append(occurrenceItems, item)
	}
	sort.Slice(occurrenceItems, func(i, j int) bool {
		return occurrenceItems[i].ID < occurrenceItems[j].ID
	})

	invitations := make([]*invitationRecord, 0, len(s.invitations))
	for _, item := range s.invitations {
		invitations = append(invitations, item)
	}
	sort.Slice(invitations, func(i, j int) bool {
		return invitations[i].ID < invitations[j].ID
	})

	state := persistedState{
		NextSeriesID:     s.nextSeriesID,
		NextOccurrenceID: s.nextOccurrenceID,
		NextInvitationID: s.nextInvitationID,
		Series:           seriesItems,
		Activities:       occurrenceItems,
		Invitations:      invitations,
		IdempotencyKeys:  s.idempotencyKeys,
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
