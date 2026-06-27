# Team Pulse

Team Pulse is a Go-based sports team management project built as four services:

- `activity-service`: manages team activities, schedules, and RSVP state
- `payment-service`: manages subscription and billing state
- `api-gateway`: exposes a single API surface for the frontend
- `web-app`: serves the user-facing application

The repository also includes local Docker support, AWS ECS Fargate infrastructure, and a GitHub Actions deployment workflow.

The current demo now covers a broader Team Pulse product surface:

- team and club activity coordination
- recurring event scheduling with time-zone validation
- attendance and RSVP tracking
- invitation queuing for push, email, and SMS workflows
- subscription billing via Stripe
- mobile-oriented API patterns such as pagination, versioning, combined responses, and caching

## Repository Contents

- Go service entrypoints under `cmd/`
- service packages under `internal/`
- static frontend assets under `web/`
- sample activity and subscription data under `data/`
- database migrations under `db/migrations/`
- local Docker Compose support in `docker-compose.yml`
- AWS Terraform infrastructure in `infra/aws/terraform`
- a convenience local startup script at `work/run-microservices.sh`

## Data Files

`data/activity-service/activities.json` contains seed activity records, including:

- training and match entries
- start times and locations
- RSVP state per participant

`data/payment-service/subscriptions.json` contains seed billing data, including:

- team subscription plan
- payment status
- renewal date
- Stripe customer identifier

## Product Understanding

This project is intentionally framed around the core product areas Team Pulse supports:

- team and club management
- event scheduling
- attendance
- messaging and invitations
- payments
- volunteer coordination
- mobile-first experience

The technically hardest feature to build reliably is usually notifications. Payments are constrained but well-scoped. Recurring events and calendar sync are subtle because of time zones and edits to future occurrences. Offline support adds sync complexity. Notifications combine the widest set of hard problems at once: fan-out, retries, deduplication, provider failures, quiet hours, preferences, and delivery observability.

## API And Architecture Notes

The activity service now exposes backward-compatible versioned routes under `/v1` alongside legacy routes:

- `POST /v1/event-series`
- `GET /v1/event-series/{seriesId}`
- `PATCH /v1/event-series/{seriesId}`
- `POST /v1/event-series/{seriesId}/split`
- `GET /v1/events?teamId=team-pulse&from=...&to=...&limit=20&cursor=...&include=dashboard`
- `PATCH /v1/events/{occurrenceId}`
- `GET /v1/activities?limit=20&cursor=...&include=dashboard`
- `POST /v1/activities`
- `POST /v1/activities/{activityId}/rsvps`
- `GET /v1/activities/{activityId}/invitations`
- `POST /v1/activities/{activityId}/invitations`

The OpenAPI contract for the gateway is available at [web/openapi.yaml](/Users/sd/Supun/GO-PET-PROJECTS/test/team-pulse/web/openapi.yaml) and is served by the web app at `/openapi.yaml`.
Scalar API documentation is served as a separate Docker service at `http://localhost:3001` when running the Compose stack.
PostgreSQL migration scripts for the app are organized under [db/migrations](/Users/sd/Supun/GO-PET-PROJECTS/test/team-pulse/db/migrations), with concrete schemas for [activity-service](/Users/sd/Supun/GO-PET-PROJECTS/test/team-pulse/db/migrations/activity-service) and [payment-service](/Users/sd/Supun/GO-PET-PROJECTS/test/team-pulse/db/migrations/payment-service), plus stateless-service markers for the gateway and web app.

Key design points:

- Recurring scheduling is modeled as `event-series` plus materialized `events` (occurrences).
- Team Pulse keeps the recurrence rule on the series and stores user-facing exceptions on individual occurrences.
- Single-occurrence changes use `PATCH /v1/events/{occurrenceId}`.
- "This and future" changes use `POST /v1/event-series/{seriesId}/split`.
- Full-series changes use `PATCH /v1/event-series/{seriesId}`.
- The relational schema mirrors that design with `event_series`, `event_occurrences`, `occurrence_rsvps`, and `occurrence_invitations`.
- Billing persistence is modeled with `subscriptions` and append-only `checkout_events`.
- Time zones are validated using IANA zone identifiers.
- Mobile clients can fetch activities and dashboard data in one request to reduce round trips.
- Activity feeds return `ETag` and `Cache-Control` headers for short-lived caching.
- Invitation creation accepts an `Idempotency-Key` header to prevent duplicate notifications on client retries.

For a production notification platform at larger scale:

- accept notification jobs synchronously, then enqueue work for background delivery
- fan out by channel: push, email, SMS
- use retries with backoff and dead-letter queues
- record idempotency keys or deduplication hashes before delivery
- scale horizontally with stateless API nodes and worker pools
- shard hot tenants or channels when throughput grows into the millions

## Local Configuration

`.env.docker` defines Stripe-related variables:

```env
STRIPE_SECRET_KEY=
STRIPE_PRICE_STARTER=
STRIPE_PRICE_CLUB=
STRIPE_PRICE_PRO=
```

Leave `STRIPE_SECRET_KEY` empty to use a mock Stripe client locally, as noted in the file.

## Local Run

The local startup script is:

```sh
./work/run-microservices.sh
```

It sets `GOCACHE` to `work/.gocache` by default and then attempts to start all four Go services with `go run`.

You can also run the containerized stack with:

```sh
docker compose up --build
```

This starts:

- the Team Pulse UI at `http://localhost:3000`
- the API gateway at `http://localhost:8080`
- the Scalar API docs at `http://localhost:3001`

## Deployment

AWS deployment assets live in [infra/aws/README.md](/Users/sd/Supun/GO-PET-PROJECTS/test/team-pulse/infra/aws/README.md). The GitHub Actions workflow at `.github/workflows/deploy.yml` builds and pushes the service images, then applies Terraform against AWS using a remote S3 backend.
