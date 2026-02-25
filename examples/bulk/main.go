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

	resp, err := client.BulkDomains(
		context.Background(),
		[]string{"google.com", "github.com", "example.com"},
		rdapapi.WithFollow(),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total: %d, Success: %d, Failed: %d\n",
		resp.Summary.Total, resp.Summary.Successful, resp.Summary.Failed)

	for _, r := range resp.Results {
		if r.Status == "success" && r.Data != nil {
			registrar := "unknown"
			if r.Data.Registrar.Name != nil {
				registrar = *r.Data.Registrar.Name
			}
			fmt.Printf("  %s — %s\n", r.Domain, registrar)
		} else {
			msg := "unknown error"
			if r.Message != nil {
				msg = *r.Message
			}
			fmt.Printf("  %s — error: %s\n", r.Domain, msg)
		}
	}
}
