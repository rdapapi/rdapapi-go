// Package rdapapi provides a Go client for the RDAP API (https://rdapapi.io).
//
// It supports looking up domains, IP addresses, ASNs, nameservers, and entities
// via the RDAP protocol. Domains in the ccTLDs that run no RDAP server are read
// over WHOIS instead and returned in the same shape; Meta.Source says which
// protocol answered.
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
