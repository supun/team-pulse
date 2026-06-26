package main

import (
	"log"
	"net/http"
	"os"

	"teampulse/internal/webapp"
)

func main() {
	port := envOrDefault("PORT", "3000")
	apiBaseURL := envOrDefault("API_BASE_URL", "http://localhost:8080")

	server := webapp.NewServer("web", apiBaseURL)

	log.Printf("web-app listening on http://localhost:%s", port)
	log.Printf("api-gateway upstream: %s", apiBaseURL)

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
