#!/bin/sh
set -eu

GOCACHE="${GOCACHE:-$(pwd)/work/.gocache}"

go run ./cmd/activity-service &
ACTIVITY_PID=$!

go run ./cmd/payment-service &
PAYMENT_PID=$!

go run ./cmd/api-gateway &
GATEWAY_PID=$!

go run ./cmd/web-app &
WEB_PID=$!

cleanup() {
  kill "$ACTIVITY_PID" "$PAYMENT_PID" "$GATEWAY_PID" "$WEB_PID" 2>/dev/null || true
}

trap cleanup INT TERM EXIT
wait
