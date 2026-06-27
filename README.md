# Team Pulse

Team Pulse is a Go-based sports team management project built as four services:

- `activity-service`: manages team activities, schedules, and RSVP state
- `payment-service`: manages subscription and billing state
- `api-gateway`: exposes a single API surface for the frontend
- `web-app`: serves the user-facing application

The repository also includes local Docker support, AWS ECS Fargate infrastructure, and a GitHub Actions deployment workflow.

## Repository Contents

- Go service entrypoints under `cmd/`
- service packages under `internal/`
- static frontend assets under `web/`
- sample activity and subscription data under `data/`
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

## Deployment

AWS deployment assets live in [infra/aws/README.md](/Users/sd/Supun/GO-PET-PROJECTS/test/team-pulse/infra/aws/README.md). The GitHub Actions workflow at `.github/workflows/deploy.yml` builds and pushes the service images, then applies Terraform against AWS using a remote S3 backend.
