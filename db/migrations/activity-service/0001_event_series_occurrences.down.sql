BEGIN;

DROP INDEX IF EXISTS occurrence_invitations_occurrence_id_created_at_idx;
DROP INDEX IF EXISTS event_occurrences_series_id_starts_at_idx;
DROP INDEX IF EXISTS event_occurrences_team_id_starts_at_idx;
DROP INDEX IF EXISTS event_series_team_id_idx;
DROP INDEX IF EXISTS occurrence_invitations_occurrence_idempotency_key_idx;

DROP TABLE IF EXISTS occurrence_invitations;
DROP TABLE IF EXISTS occurrence_rsvps;
DROP TABLE IF EXISTS event_occurrences;
DROP TABLE IF EXISTS event_series;

COMMIT;
