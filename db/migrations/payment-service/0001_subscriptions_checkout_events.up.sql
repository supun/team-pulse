BEGIN;

CREATE TABLE subscriptions (
    team_id TEXT PRIMARY KEY,
    team_name TEXT NOT NULL,
    plan TEXT NOT NULL CHECK (plan IN ('starter', 'club', 'pro')),
    status TEXT NOT NULL CHECK (status IN ('trial', 'active', 'past_due')),
    price_monthly_nok INTEGER NOT NULL CHECK (price_monthly_nok >= 0),
    renewal_date TIMESTAMPTZ NOT NULL,
    last_payment_status TEXT NOT NULL,
    last_payment_at TIMESTAMPTZ,
    stripe_customer_id TEXT,
    stripe_session_id TEXT,
    stripe_session_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE checkout_events (
    id BIGSERIAL PRIMARY KEY,
    team_id TEXT NOT NULL REFERENCES subscriptions(team_id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN ('checkout_session_created', 'checkout_session_completed')),
    stripe_session_id TEXT,
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX subscriptions_status_idx
    ON subscriptions (status);

CREATE INDEX subscriptions_plan_idx
    ON subscriptions (plan);

CREATE INDEX checkout_events_team_id_created_at_idx
    ON checkout_events (team_id, created_at DESC);

CREATE INDEX checkout_events_stripe_session_id_idx
    ON checkout_events (stripe_session_id)
    WHERE stripe_session_id IS NOT NULL;

COMMIT;
