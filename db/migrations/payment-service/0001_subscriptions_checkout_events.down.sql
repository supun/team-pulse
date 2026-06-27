BEGIN;

DROP INDEX IF EXISTS checkout_events_stripe_session_id_idx;
DROP INDEX IF EXISTS checkout_events_team_id_created_at_idx;
DROP INDEX IF EXISTS subscriptions_plan_idx;
DROP INDEX IF EXISTS subscriptions_status_idx;

DROP TABLE IF EXISTS checkout_events;
DROP TABLE IF EXISTS subscriptions;

COMMIT;
