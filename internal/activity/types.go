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

type ActivityPayload struct {
	Title           string       `json:"title"`
	Kind            ActivityKind `json:"kind"`
	StartsAt        time.Time    `json:"startsAt"`
	Location        string       `json:"location"`
	MaxParticipants int          `json:"maxParticipants"`
	Notes           string       `json:"notes"`
}

type RSVPRequest struct {
	MemberName string     `json:"memberName"`
	Status     RSVPStatus `json:"status"`
}

type ActivityResponse struct {
	ID              string       `json:"id"`
	Title           string       `json:"title"`
	Kind            ActivityKind `json:"kind"`
	StartsAt        time.Time    `json:"startsAt"`
	Location        string       `json:"location"`
	MaxParticipants int          `json:"maxParticipants"`
	Notes           string       `json:"notes"`
	GoingCount      int          `json:"goingCount"`
	MaybeCount      int          `json:"maybeCount"`
	DeclinedCount   int          `json:"declinedCount"`
	ConfirmedRatio  float64      `json:"confirmedRatio"`
}

type ActivityListResponse struct {
	Items []ActivityResponse `json:"items"`
}

type DashboardResponse struct {
	UpcomingActivities int                  `json:"upcomingActivities"`
	TotalGoing         int                  `json:"totalGoing"`
	ByKind             map[ActivityKind]int `json:"byKind"`
	NextActivity       *ActivityResponse    `json:"nextActivity"`
}
