package activity

import "time"

type ActivityKind string

const (
	KindMatch     ActivityKind = "match"
	KindTraining  ActivityKind = "training"
	KindVolunteer ActivityKind = "volunteer"
	KindSocial    ActivityKind = "social"
)

type RSVPStatus string

const (
	RSVPGoing    RSVPStatus = "going"
	RSVPMaybe    RSVPStatus = "maybe"
	RSVPDeclined RSVPStatus = "declined"
)

type UpdateScope string

const (
	ScopeSingleOccurrence UpdateScope = "single_occurrence"
	ScopeThisAndFuture    UpdateScope = "this_and_future"
	ScopeEntireSeries     UpdateScope = "entire_series"
)

type ActivityPayload struct {
	TeamID          string       `json:"teamId"`
	Title           string       `json:"title"`
	Kind            ActivityKind `json:"kind"`
	StartsAt        time.Time    `json:"startsAt"`
	TimeZone        string       `json:"timeZone"`
	Location        string       `json:"location"`
	MaxParticipants int          `json:"maxParticipants"`
	Notes           string       `json:"notes"`
	Recurrence      *Recurrence  `json:"recurrence,omitempty"`
}

type EventSeriesPayload struct {
	TeamID          string       `json:"teamId"`
	Title           string       `json:"title"`
	Kind            ActivityKind `json:"kind"`
	StartsAt        time.Time    `json:"startsAt"`
	TimeZone        string       `json:"timeZone"`
	Location        string       `json:"location"`
	MaxParticipants int          `json:"maxParticipants"`
	Notes           string       `json:"notes"`
	Recurrence      *Recurrence  `json:"recurrence,omitempty"`
	HorizonDays     int          `json:"horizonDays,omitempty"`
}

type EventSeriesPatchRequest struct {
	TeamID          *string       `json:"teamId,omitempty"`
	Title           *string       `json:"title,omitempty"`
	Kind            *ActivityKind `json:"kind,omitempty"`
	StartsAt        *time.Time    `json:"startsAt,omitempty"`
	TimeZone        *string       `json:"timeZone,omitempty"`
	Location        *string       `json:"location,omitempty"`
	MaxParticipants *int          `json:"maxParticipants,omitempty"`
	Notes           *string       `json:"notes,omitempty"`
	Recurrence      *Recurrence   `json:"recurrence,omitempty"`
	HorizonDays     *int          `json:"horizonDays,omitempty"`
}

type EventSeriesSplitRequest struct {
	OccurrenceID string                   `json:"occurrenceId"`
	Patch        *EventSeriesPatchRequest `json:"patch"`
}

type EventOccurrencePatchRequest struct {
	Title           *string       `json:"title,omitempty"`
	Kind            *ActivityKind `json:"kind,omitempty"`
	StartsAt        *time.Time    `json:"startsAt,omitempty"`
	TimeZone        *string       `json:"timeZone,omitempty"`
	Location        *string       `json:"location,omitempty"`
	MaxParticipants *int          `json:"maxParticipants,omitempty"`
	Notes           *string       `json:"notes,omitempty"`
	Status          *string       `json:"status,omitempty"`
}

type RSVPRequest struct {
	MemberName string     `json:"memberName"`
	Status     RSVPStatus `json:"status"`
}

type Recurrence struct {
	Frequency string `json:"frequency"`
	Interval  int    `json:"interval"`
	Count     int    `json:"count"`
}

type ActivityResponse struct {
	ID              string       `json:"id"`
	SeriesID        string       `json:"seriesId,omitempty"`
	TeamID          string       `json:"teamId,omitempty"`
	OccurrenceIndex int          `json:"occurrenceIndex,omitempty"`
	Title           string       `json:"title"`
	Kind            ActivityKind `json:"kind"`
	StartsAt        time.Time    `json:"startsAt"`
	TimeZone        string       `json:"timeZone"`
	Location        string       `json:"location"`
	MaxParticipants int          `json:"maxParticipants"`
	Notes           string       `json:"notes"`
	Status          string       `json:"status,omitempty"`
	IsException     bool         `json:"isException,omitempty"`
	GoingCount      int          `json:"goingCount"`
	MaybeCount      int          `json:"maybeCount"`
	DeclinedCount   int          `json:"declinedCount"`
	ConfirmedRatio  float64      `json:"confirmedRatio"`
	Recurrence      *Recurrence  `json:"recurrence,omitempty"`
}

type ActivityListResponse struct {
	Items      []ActivityResponse `json:"items"`
	NextCursor string             `json:"nextCursor,omitempty"`
	Dashboard  *DashboardResponse `json:"dashboard,omitempty"`
}

type DashboardResponse struct {
	UpcomingActivities int                  `json:"upcomingActivities"`
	TotalGoing         int                  `json:"totalGoing"`
	ByKind             map[ActivityKind]int `json:"byKind"`
	NextActivity       *ActivityResponse    `json:"nextActivity"`
}

type EventSeriesResponse struct {
	ID                string             `json:"id"`
	TeamID            string             `json:"teamId"`
	Title             string             `json:"title"`
	Kind              ActivityKind       `json:"kind"`
	StartsAt          time.Time          `json:"startsAt"`
	TimeZone          string             `json:"timeZone"`
	Location          string             `json:"location"`
	MaxParticipants   int                `json:"maxParticipants"`
	Notes             string             `json:"notes"`
	Status            string             `json:"status"`
	HorizonDays       int                `json:"horizonDays"`
	Recurrence        *Recurrence        `json:"recurrence,omitempty"`
	OccurrenceCount   int                `json:"occurrenceCount"`
	UpcomingCount     int                `json:"upcomingCount"`
	FirstOccurrenceID string             `json:"firstOccurrenceId,omitempty"`
	LastOccurrenceID  string             `json:"lastOccurrenceId,omitempty"`
	Occurrences       []ActivityResponse `json:"occurrences,omitempty"`
}

type EventSeriesListResponse struct {
	Items []EventSeriesResponse `json:"items"`
}

type InvitationPayload struct {
	Channel    string   `json:"channel"`
	Recipients []string `json:"recipients"`
	Message    string   `json:"message"`
}

type InvitationResponse struct {
	ID             string    `json:"id"`
	ActivityID     string    `json:"activityId"`
	Channel        string    `json:"channel"`
	Recipients     []string  `json:"recipients"`
	Message        string    `json:"message"`
	Status         string    `json:"status"`
	IdempotencyKey string    `json:"idempotencyKey,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type InvitationListResponse struct {
	Items      []InvitationResponse `json:"items"`
	NextCursor string               `json:"nextCursor,omitempty"`
}
