// Package rdapapi provides a Go client for the RDAP API (https://rdapapi.io).
//
// It supports looking up domains, IP addresses, ASNs, nameservers, and entities
// via the RDAP protocol.
//
// Basic usage:
//
//	client := rdapapi.NewClient("your-api-key")
//
//	domain, err := client.Domain(ctx, "google.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(domain.Registrar.Name)
package rdapapi
