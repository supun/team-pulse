BEGIN;

CREATE TABLE event_series (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    title TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('match', 'training', 'volunteer', 'social')),
    starts_at_utc TIMESTAMPTZ NOT NULL,
    time_zone TEXT NOT NULL,
    location TEXT NOT NULL,
    max_participants INTEGER NOT NULL CHECK (max_participants > 0),
    notes TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    horizon_days INTEGER NOT NULL DEFAULT 180 CHECK (horizon_days > 0),
    recurrence_frequency TEXT CHECK (recurrence_frequency IN ('daily', 'weekly')),
    recurrence_interval INTEGER CHECK (recurrence_interval IS NULL OR recurrence_interval > 0),
    recurrence_count INTEGER CHECK (recurrence_count IS NULL OR recurrence_count > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (recurrence_frequency IS NULL AND recurrence_interval IS NULL AND recurrence_count IS NULL)
        OR
        (recurrence_frequency IS NOT NULL AND recurrence_interval IS NOT NULL AND recurrence_count IS NOT NULL)
    )
);

CREATE TABLE event_occurrences (
    id TEXT PRIMARY KEY,
    series_id TEXT NOT NULL REFERENCES event_series(id) ON DELETE CASCADE,
    team_id TEXT NOT NULL,
    occurrence_index INTEGER NOT NULL CHECK (occurrence_index > 0),
    title TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('match', 'training', 'volunteer', 'social')),
    starts_at_utc TIMESTAMPTZ NOT NULL,
    time_zone TEXT NOT NULL,
    location TEXT NOT NULL,
    max_participants INTEGER NOT NULL CHECK (max_participants > 0),
    notes TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'scheduled',
    is_exception BOOLEAN NOT NULL DEFAULT FALSE,
    recurrence_frequency TEXT CHECK (recurrence_frequency IN ('daily', 'weekly')),
    recurrence_interval INTEGER CHECK (recurrence_interval IS NULL OR recurrence_interval > 0),
    recurrence_count INTEGER CHECK (recurrence_count IS NULL OR recurrence_count > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (series_id, occurrence_index)
);

CREATE TABLE occurrence_rsvps (
    occurrence_id TEXT NOT NULL REFERENCES event_occurrences(id) ON DELETE CASCADE,
    member_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('going', 'maybe', 'declined')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (occurrence_id, member_name)
);

CREATE TABLE occurrence_invitations (
    id TEXT PRIMARY KEY,
    occurrence_id TEXT NOT NULL REFERENCES event_occurrences(id) ON DELETE CASCADE,
    channel TEXT NOT NULL CHECK (channel IN ('push', 'email', 'sms')),
    recipients JSONB NOT NULL,
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    idempotency_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (jsonb_typeof(recipients) = 'array')
);

CREATE UNIQUE INDEX occurrence_invitations_occurrence_idempotency_key_idx
    ON occurrence_invitations (occurrence_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX event_series_team_id_idx
    ON event_series (team_id);

CREATE INDEX event_occurrences_team_id_starts_at_idx
    ON event_occurrences (team_id, starts_at_utc);

CREATE INDEX event_occurrences_series_id_starts_at_idx
    ON event_occurrences (series_id, starts_at_utc);

CREATE INDEX occurrence_invitations_occurrence_id_created_at_idx
    ON occurrence_invitations (occurrence_id, created_at DESC);

COMMIT;
