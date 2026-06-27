package main

import (
	"log"
	"net/http"
	"os"

	"teampulse/internal/payment"
)

func main() {
	port := envOrDefault("PORT", "8082")
	stripeClient := buildStripeClient()
	handler := payment.NewHandler(payment.NewService(stripeClient, envOrDefault("PAYMENT_DATA_FILE", "data/payment-service/subscriptions.json")))

	log.Printf("payment-service listening on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, handler.Routes()); err != nil {
		log.Fatal(err)
	}
}

func buildStripeClient() payment.StripeClient {
	secretKey := os.Getenv("STRIPE_SECRET_KEY")
	priceIDs := map[payment.SubscriptionPlan]string{
		payment.PlanStarter: os.Getenv("STRIPE_PRICE_STARTER"),
		payment.PlanClub:    os.Getenv("STRIPE_PRICE_CLUB"),
		payment.PlanPro:     os.Getenv("STRIPE_PRICE_PRO"),
	}

	if secretKey == "" {
		log.Print("payment-service using mock Stripe client; set STRIPE_SECRET_KEY and STRIPE_PRICE_* env vars for real Stripe Checkout")
		return &payment.MockStripeClient{}
	}

	return payment.NewStripeAPIClient(secretKey, priceIDs)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
