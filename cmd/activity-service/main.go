package main

import (
	"log"
	"net/http"
	"os"

	"teampulse/internal/activity"
)

func main() {
	port := envOrDefault("PORT", "8081")
	handler := activity.NewHandler(activity.NewService(envOrDefault("ACTIVITY_DATA_FILE", "data/activity-service/activities.json")))

	log.Printf("activity-service listening on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, handler.Routes()); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
