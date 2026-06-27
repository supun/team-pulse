package main

import (
	"log"
	"net/http"
	"os"

	"teampulse/internal/gateway"
)

func main() {
	port := envOrDefault("PORT", "8080")
	activityURL := envOrDefault("ACTIVITY_SERVICE_URL", "http://localhost:8081")
	paymentURL := envOrDefault("PAYMENT_SERVICE_URL", "http://localhost:8082")

	server := gateway.NewServer(activityURL, paymentURL)

	log.Printf("api-gateway listening on http://localhost:%s", port)
	log.Printf("activity-service upstream: %s", activityURL)
	log.Printf("payment-service upstream: %s", paymentURL)

	if err := http.ListenAndServe(":"+port, server.Routes()); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
