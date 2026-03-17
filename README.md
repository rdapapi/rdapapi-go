# rdapapi-go

Official Go SDK for the [RDAP API](https://rdapapi.io) — look up domains, IP addresses, ASNs, nameservers, and entities via the RDAP protocol.

[![Go Reference](https://pkg.go.dev/badge/github.com/rdapapi/rdapapi-go.svg)](https://pkg.go.dev/github.com/rdapapi/rdapapi-go)
[![CI](https://github.com/rdapapi/rdapapi-go/actions/workflows/ci.yml/badge.svg)](https://github.com/rdapapi/rdapapi-go/actions/workflows/ci.yml)

## Installation

```bash
go get github.com/rdapapi/rdapapi-go
```

Requires Go 1.22 or later. Zero external dependencies.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    rdapapi "github.com/rdapapi/rdapapi-go"
)

func main() {
    client := rdapapi.NewClient("your-api-key")

    domain, err := client.Domain(context.Background(), "google.com")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(domain.Domain)
    if domain.Registrar != nil && domain.Registrar.Name != nil {
        fmt.Println(*domain.Registrar.Name)
    }
    if domain.Dates != nil && domain.Dates.Registered != nil {
        fmt.Println(*domain.Dates.Registered)
    }
}
```

## Usage

### Client Options

```go
// Custom timeout
client := rdapapi.NewClient("key", rdapapi.WithTimeout(10*time.Second))

// Custom base URL
client := rdapapi.NewClient("key", rdapapi.WithBaseURL("https://custom.api.com/v1"))

// Custom HTTP client
client := rdapapi.NewClient("key", rdapapi.WithHTTPClient(myHTTPClient))
```

### Domain Lookup

```go
domain, err := client.Domain(ctx, "example.com")

// With registrar follow-through (thin registries)
domain, err := client.Domain(ctx, "example.com", rdapapi.WithFollow())
```

### IP Address Lookup

```go
ip, err := client.IP(ctx, "8.8.8.8")
fmt.Println(*ip.Name)        // "LVLT-GOGL-8-8-8"
fmt.Println(*ip.Country)     // "US"
fmt.Println(ip.CIDR)         // ["8.8.8.0/24"]
```

### ASN Lookup

```go
asn, err := client.ASN(ctx, "15169")    // or "AS15169"
fmt.Println(*asn.Name)                   // "GOOGLE"
```

### Nameserver Lookup

```go
ns, err := client.Nameserver(ctx, "ns1.google.com")
fmt.Println(ns.IPAddresses.V4)  // ["216.239.32.10"]
```

### Entity Lookup

```go
entity, err := client.Entity(ctx, "GOGL")
fmt.Println(*entity.Organization)  // "Google LLC"
fmt.Println(entity.Networks)       // IP blocks owned by entity
```

### Bulk Domain Lookup

Requires a Pro or Business plan. Up to 10 domains per call.

```go
resp, err := client.BulkDomains(ctx, []string{"google.com", "github.com"}, rdapapi.WithFollow())
for _, r := range resp.Results {
    if r.Status == "success" {
        fmt.Printf("%s — %s\n", r.Domain, *r.Data.Registrar.Name)
    }
}
```

## Error Handling

All API errors are returned as typed errors that can be checked with `errors.As`:

```go
domain, err := client.Domain(ctx, "example.com")
if err != nil {
    var notFound *rdapapi.NotFoundError
    if errors.As(err, &notFound) {
        fmt.Println("Not found:", notFound.Message)
    }

    var rateLimited *rdapapi.RateLimitError
    if errors.As(err, &rateLimited) {
        fmt.Printf("Retry after %d seconds\n", rateLimited.RetryAfter)
    }

    var authErr *rdapapi.AuthenticationError
    if errors.As(err, &authErr) {
        fmt.Println("Invalid API key")
    }
}
```

| Error Type | HTTP Status | Description |
|---|---|---|
| `ValidationError` | 400 | Invalid input |
| `AuthenticationError` | 401 | Invalid or missing API key |
| `SubscriptionRequiredError` | 403 | No active subscription |
| `NotFoundError` | 404 | No RDAP data found |
| `RateLimitError` | 429 | Rate limit or quota exceeded |
| `UpstreamError` | 502 | Upstream RDAP server failure |
| `TemporarilyUnavailableError` | 503 | Domain data temporarily unavailable |

All typed errors embed `*APIError` which provides `StatusCode`, `Code`, `Message`, and `RetryAfter` fields.

## Nullable Fields

Fields that may be absent in API responses use Go pointer types (`*string`, `*int`, `*bool`). Always check for `nil` before dereferencing:

```go
if domain.Dates.Expires != nil {
    fmt.Println("Expires:", *domain.Dates.Expires)
}
```

## Development

Set up pre-commit hooks (runs lint + tests before each commit):

```bash
git config core.hooksPath .githooks
```

## License

MIT — see [LICENSE](LICENSE).
