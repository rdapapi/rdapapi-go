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
	ctx := context.Background()

	// Full catalog. Does not count against your monthly quota.
	tlds, err := client.TLDs(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%d TLDs supported, coverage %.0f%%\n", tlds.Meta.Count, tlds.Meta.Coverage*100)

	for i, tld := range tlds.Data {
		if i >= 5 {
			break
		}
		if tld.FieldAvailability == nil {
			fmt.Printf(".%s via %s (not enough data yet)\n", tld.TLD, tld.RDAPServerHost)
			continue
		}
		fmt.Printf(
			".%s via %s: registrar=%s, expires_at=%s\n",
			tld.TLD, tld.RDAPServerHost,
			tld.FieldAvailability.Registrar,
			tld.FieldAvailability.ExpiresAt,
		)
	}

	// Skip the transfer when nothing has changed.
	later, err := client.TLDs(ctx, rdapapi.WithIfNoneMatch(tlds.ETag))
	if err != nil {
		log.Fatal(err)
	}
	if later == nil {
		fmt.Println("No change since last poll")
	}

	// Single-TLD lookup.
	com, err := client.TLD(ctx, "com")
	if err != nil {
		log.Fatal(err)
	}
	if com != nil {
		fmt.Printf(".com supported since %s\n", com.Data.SupportedSince)
	}
}
