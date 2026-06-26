# Team Pulse

Team Pulse is a Go-based sports team management project intended to run as a small set of local microservices plus a web app.

## Current Repository State

This repository currently includes:

- sample activity data in `data/activity-service/activities.json`
- sample subscription data in `data/payment-service/subscriptions.json`
- local Stripe-related environment defaults in `.env.docker`
- a convenience startup script in `work/run-microservices.sh`

The service entrypoints referenced by the startup script are not present in the current tree yet:

- `./cmd/activity-service`
- `./cmd/payment-service`
- `./cmd/api-gateway`
- `./cmd/web-app`

That means the script documents the intended runtime shape, but it will not start successfully until those Go commands are added.

## Intended Services

- `activity-service`: manages team activities, schedules, and RSVP state
- `payment-service`: manages subscription and billing state
- `api-gateway`: exposes a single API surface for the frontend
- `web-app`: serves the user-facing application

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

## Local Configuration

`.env.docker` defines Stripe-related variables:

```env
STRIPE_SECRET_KEY=
STRIPE_PRICE_STARTER=
STRIPE_PRICE_CLUB=
STRIPE_PRICE_PRO=
```

Leave `STRIPE_SECRET_KEY` empty to use a mock Stripe client locally, as noted in the file.

## Run Script

The local startup script is:

```sh
./work/run-microservices.sh
```

It sets `GOCACHE` to `work/.gocache` by default and then attempts to start all four Go services with `go run`.

## Next Steps

1. Add the missing Go service entrypoints under `cmd/`.
2. Add a `go.mod` file if the project is intended to build as a Go module.
3. Update this README with actual setup, API, and development instructions once the service code exists.
