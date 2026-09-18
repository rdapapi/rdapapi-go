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

	_, err := client.Domain(context.Background(), "example.nope")
	if err != nil {
		// Check NotSupportedError first: it's a 404 variant for uncovered namespaces.
		var notSupported *rdapapi.NotSupportedError
		if errors.As(err, &notSupported) {
			fmt.Printf("TLD not covered by RDAP: %s\n", notSupported.Message)
			return
		}

		var notFound *rdapapi.NotFoundError
		if errors.As(err, &notFound) {
			fmt.Printf("Domain not registered: %s (code: %s)\n", notFound.Message, notFound.Code)
			return
		}

		// Check QuotaExceededError first: it's a 429 variant that waiting
		// will not clear.
		var quota *rdapapi.QuotaExceededError
		if errors.As(err, &quota) {
			fmt.Println("Monthly quota spent. Upgrade at https://rdapapi.io/pricing")
			return
		}

		var rateLimited *rdapapi.RateLimitError
		if errors.As(err, &rateLimited) {
			fmt.Printf("Rate limited, retry after %d seconds\n", rateLimited.RetryAfter)
			return
		}

		var authErr *rdapapi.AuthenticationError
		if errors.As(err, &authErr) {
			fmt.Println("Invalid API key")
			return
		}

		// Check PlanUpgradeRequiredError first: it's a 403 variant raised on
		// an account that is already subscribed.
		var upgrade *rdapapi.PlanUpgradeRequiredError
		if errors.As(err, &upgrade) {
			fmt.Println("Bulk lookups need a Pro or Business plan")
			return
		}

		// Branch on Code: a 403 from the CDN edge — an IP block, a WAF rule —
		// never reaches the API, so it carries no JSON body and Code is
		// unknown_error on an account whose billing is fine.
		var subErr *rdapapi.SubscriptionRequiredError
		if errors.As(err, &subErr) {
			if subErr.Code == "subscription_required" {
				fmt.Println("No active subscription. Visit https://rdapapi.io/pricing")
			} else {
				fmt.Printf("Forbidden: %s (code: %s)\n", subErr.Message, subErr.Code)
			}
			return
		}

		log.Fatal(err)
	}
}
