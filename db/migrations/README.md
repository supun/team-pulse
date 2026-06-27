# Team Pulse Migrations

This directory contains SQL migrations for services that move from file-backed
demo persistence to a relational database.

## Activity Service

The `activity-service` migration set models recurring scheduling as:

- `event_series`: shared recurrence definition and team-level metadata
- `event_occurrences`: materialized occurrences generated from a series
- `occurrence_rsvps`: RSVP state keyed by occurrence and member name
- `occurrence_invitations`: queued invitation attempts with idempotency support

The schema matches the Team Pulse API contract:

- `POST /api/v1/event-series` writes to `event_series` and precomputes rows in `event_occurrences`
- `PATCH /api/v1/events/{occurrenceId}` updates a single row in `event_occurrences` and marks it as an exception
- `POST /api/v1/event-series/{seriesId}/split` archives or shortens the original series and creates a new series with future occurrences
- `POST /api/v1/activities/{activityId}/invitations` persists an invitation row with a unique `(occurrence_id, idempotency_key)` constraint

The SQL is written for PostgreSQL.

## Payment Service

The `payment-service` migration set models:

- `subscriptions`: current billing state per team
- `checkout_events`: append-only checkout and webhook lifecycle events

This matches the Team Pulse billing API:

- `POST /api/v1/checkout-sessions` upserts the subscription trial state and records a `checkout_session_created` event
- `POST /api/v1/webhooks/stripe` updates the subscription state and records a `checkout_session_completed` event
- `GET /api/v1/subscriptions` and `GET /api/v1/subscriptions/{teamId}` read from `subscriptions`

## Stateless Services

`api-gateway` and `web-app` do not currently own relational state. Their
directories contain README markers instead of SQL files so the migration layout
still covers the whole app.
