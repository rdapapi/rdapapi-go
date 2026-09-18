package main

import (
	"context"
	"fmt"
	"log"
	"os"

	rdapapi "github.com/rdapapi/rdapapi-go"
)

func main() {
	client := rdapapi.NewClient(os.Getenv("RDAPAPI_KEY"))

	// Basic domain lookup.
	domain, err := client.Domain(context.Background(), "google.com")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Domain: %s\n", domain.Domain)
	if domain.Registrar.Name != nil {
		fmt.Printf("Registrar: %s\n", *domain.Registrar.Name)
	}
	if domain.Dates.Registered != nil {
		fmt.Printf("Registered: %s\n", *domain.Dates.Registered)
	}
	if domain.Dates.Expires != nil {
		fmt.Printf("Expires: %s\n", *domain.Dates.Expires)
	}
	fmt.Printf("Status: %v\n", domain.Status)
	fmt.Printf("Nameservers: %v\n", domain.Nameservers)
	if domain.DNSSEC != nil {
		fmt.Printf("DNSSEC: %v\n", *domain.DNSSEC)
	} else {
		fmt.Println("DNSSEC: not published by this registry")
	}
	fmt.Printf("Answered over %s\n", domain.Meta.Source)

	// What the registry declared it withheld, where it declared anything.
	if domain.Redacted != nil {
		for role, fields := range domain.Redacted.Entities {
			for field, method := range fields {
				fmt.Printf("Redacted: %s.%s (%s)\n", role, field, method)
			}
		}
	}

	// With registrar follow-through.
	followed, err := client.Domain(context.Background(), "google.com", rdapapi.WithFollow())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n--- With follow ---\n")
	fmt.Printf("Followed: %v\n", followed.Meta.Followed)
	if followed.Entities.Registrant != nil && followed.Entities.Registrant.Name != nil {
		fmt.Printf("Registrant: %s\n", *followed.Entities.Registrant.Name)
	}
}
