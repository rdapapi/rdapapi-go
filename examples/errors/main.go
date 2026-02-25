package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	rdapapi "github.com/rdapapi/rdapapi-go"
)

func main() {
	client := rdapapi.NewClient(os.Getenv("RDAPAPI_KEY"))

	_, err := client.Domain(context.Background(), "this-domain-does-not-exist.example")
	if err != nil {
		var notFound *rdapapi.NotFoundError
		if errors.As(err, &notFound) {
			fmt.Printf("Domain not found: %s (code: %s)\n", notFound.Message, notFound.Code)
			return
		}

		var rateLimited *rdapapi.RateLimitError
		if errors.As(err, &rateLimited) {
			fmt.Printf("Rate limited — retry after %d seconds\n", rateLimited.RetryAfter)
			return
		}

		var authErr *rdapapi.AuthenticationError
		if errors.As(err, &authErr) {
			fmt.Println("Invalid API key")
			return
		}

		var subErr *rdapapi.SubscriptionRequiredError
		if errors.As(err, &subErr) {
			fmt.Println("Subscription required — visit https://rdapapi.io/pricing")
			return
		}

		log.Fatal(err)
	}
}
