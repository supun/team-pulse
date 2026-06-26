# TeamPulse

TeamPulse is a small full-stack pet project for sports team coordination. It now uses a microservice-oriented layout so the core team activity domain and subscription billing domain are isolated the way a production system often would be.

## Why this project fits the JD

- **Go backend services:** each backend service is written in Go using the standard library.
- **API and JSON design:** the gateway and services communicate with JSON over HTTP.
- **Frontend work:** the UI is built with React modules in the browser.
- **Scalable architecture:** activity management and payments are split into separate services behind an API gateway.
- **Quality focus:** each service has HTTP-level tests and input validation.

## Service layout

- `cmd/activity-service`: owns matches, training, volunteer work, social events, RSVP flow, and dashboard aggregation
- `cmd/payment-service`: owns subscription plans, Stripe checkout creation, webhook processing, and billing state
- `cmd/api-gateway`: exposes a backend-for-frontend API and proxies client requests to backend services
- `cmd/web-app`: serves the standalone frontend and reads backend location from runtime config
- `internal/activity` and `internal/payment`: service-owned models and logic with no shared cross-service DTO package
- `infra/aws/terraform`: AWS ECS Fargate deployment scaffold

Default ports:

- Web app: `3000`
- API gateway: `8080`
- Activity service: `8081`
- Payment service: `8082`

## Features

- Create team activities for matches, training, volunteer work, or social events
- Capture RSVP responses per activity
- Show a dashboard summary with upcoming events and attendance
- Start team subscriptions through Stripe Checkout in a dedicated payment service
- Run the frontend independently from the backend services

## API

### API gateway endpoints

### `GET /api/health`

Returns a simple health response.

### `GET /api/activities`

Returns all activities sorted by start time.

### `POST /api/activities`

Example body:

```json
{
  "title": "Saturday Match",
  "kind": "match",
  "startsAt": "2026-06-27T14:00:00Z",
  "location": "River Park",
  "maxParticipants": 16,
  "notes": "Meet 45 minutes early."
}
```

### `POST /api/activities/:id/rsvps`

Example body:

```json
{
  "memberName": "Mia",
  "status": "going"
}
```

### `GET /api/dashboard`

Returns summary cards for the frontend.

### `GET /api/subscriptions`

Returns the current subscription list from the payment service.

### `POST /api/checkout-sessions`

Example body:

```json
{
  "teamId": "team-pulse",
  "teamName": "Pulse United",
  "plan": "pro",
  "successUrl": "http://localhost:3000/billing/success",
  "cancelUrl": "http://localhost:3000/billing/cancel"
}
```

This creates a Stripe subscription checkout session and returns the hosted checkout URL.

### `POST /api/webhooks/stripe`

Accepts Stripe webhook events. In this demo, `checkout.session.completed` activates the subscription record.

## Run locally

From `/Users/sd/Documents/Codex/2026-06-25/u`:

```bash
go run ./cmd/activity-service
```

In a second terminal:

```bash
go run ./cmd/payment-service
```

Optional Stripe env vars for real Checkout:

```bash
STRIPE_SECRET_KEY=sk_test_...
STRIPE_PRICE_STARTER=price_...
STRIPE_PRICE_CLUB=price_...
STRIPE_PRICE_PRO=price_...
go run ./cmd/payment-service
```

In a third terminal:

```bash
go run ./cmd/api-gateway
```

In a fourth terminal:

```bash
go run ./cmd/web-app
```

Then open `http://localhost:3000`.

There is also a helper script at `/Users/sd/Documents/Codex/2026-06-25/u/work/run-microservices.sh`.

## Run locally with Docker Compose

Create a local Docker env file:

```bash
cp .env.docker.example .env.docker
```

If you want real Stripe Checkout, fill in the `STRIPE_*` values in `.env.docker`. If not, leave `STRIPE_SECRET_KEY` empty and the payment service will use the mock Stripe client.

Start the full stack:

```bash
docker compose up --build
```

Run in the background:

```bash
docker compose up --build -d
```

Stop everything:

```bash
docker compose down
```

If `docker compose up --build` fails with `error getting credentials` and mentions
`docker-credential-desktop`, your Docker client is configured to use a credential
helper binary that is not installed. For public base images in this repo, remove
`"credsStore": "desktop"` from `~/.docker/config.json`, or run with a temporary
clean Docker config:

```bash
mkdir -p /tmp/docker-config
printf '{ "auths": {} }\n' > /tmp/docker-config/config.json
DOCKER_CONFIG=/tmp/docker-config docker compose up --build
```

The compose stack exposes:

- `http://localhost:3000` for the web app
- `http://localhost:8080` for the API gateway
- `http://localhost:8081` for the activity service
- `http://localhost:8082` for the payment service

Local service data is persisted in mounted folders:

- `data/activity-service`
- `data/payment-service`

The frontend reads its backend URL from `/app-config.js`. By default it points to `http://localhost:8080`, but you can override it with `API_BASE_URL` when starting `cmd/web-app`.

If Stripe env vars are not set, the payment service falls back to a mock Stripe client so the project remains runnable without external credentials.

Each backend service also owns its own persisted JSON data file by default:

- activity service: `data/activity-service/activities.json`
- payment service: `data/payment-service/subscriptions.json`

## Test

```bash
go test ./...
```

## AWS deployment

The repo now includes an AWS deployment path in [infra/aws/README.md](/Users/sd/Documents/Codex/2026-06-25/u/infra/aws/README.md:1).

The target shape is:

- ECS Fargate for `web-app`, `api-gateway`, `activity-service`, and `payment-service`
- one public ALB routing `/*` to the web app and `/api/*` to the API gateway
- Cloud Map service discovery for internal gateway-to-service calls
- ECR-backed container images built from the Dockerfiles in `/Users/sd/Documents/Codex/2026-06-25/u/build`
- Stripe secret injection through AWS Secrets Manager for the payment service

Terraform lives in `/Users/sd/Documents/Codex/2026-06-25/u/infra/aws/terraform`.

## Suggested next steps

- Replace the in-memory stores with PostgreSQL per service
- Add an asynchronous event flow between services for subscription lifecycle changes
- Add authentication and tenant-aware team membership roles
- Add CI/CD to build images, push to ECR, and apply Terraform automatically
