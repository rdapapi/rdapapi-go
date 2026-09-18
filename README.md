# rdapapi-go

Official Go SDK for the [RDAP API](https://rdapapi.io) — look up domains, IP addresses, ASNs, nameservers, and entities via the RDAP protocol, with a WHOIS fallback for the ccTLDs that run no RDAP server.

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
    if domain.Registrar.Name != nil {
        fmt.Println(*domain.Registrar.Name)
    }
    if domain.Dates.Registered != nil {
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

`DNSSEC` is `*bool`: `nil` means the registry publishes no DNSSEC status (`.tr`, `.gg`
and `.nc` do not), which is not the same as an unsigned delegation.

```go
if domain.DNSSEC != nil && *domain.DNSSEC {
    fmt.Println("Signed delegation")
}
```

### WHOIS Fallback

Some ccTLDs (`.it`, `.eu`, `.tr` …) have no RDAP server. They are read from the
registry's WHOIS server and come back in the same shape, with `Meta.Source` set to
`rdapapi.ProtocolWHOIS`. `Meta.Server` names the host that answered, whichever
protocol it spoke.

```go
if domain.Meta.Source == rdapapi.ProtocolWHOIS {
    fmt.Println("Answered over WHOIS — the registry publishes no RDAP")
}

// Refuse the fallback: those TLDs then return a *NotSupportedError.
domain, err := client.Domain(ctx, "example.it", rdapapi.WithoutWhois())
```

`Meta.RDAPServer` is deprecated and empty on a WHOIS answer; use `Meta.Server` for the
host, or `Meta.RawRDAPURL` for the exact RDAP endpoint queried.

### Redaction

`Redacted` reports what the upstream server *declared* it withheld, per RFC 9537. It
mirrors the shape of the record, and is `nil` when the server declared nothing — which
is not evidence that nothing was withheld. Available on domain, IP, ASN, nameserver and
entity responses.

```go
if domain.Redacted != nil {
    // replacementValue is the one to check: the field holds a substitute
    // that looks genuine.
    if m := domain.Redacted.Registrar["iana_id"]; m == rdapapi.RedactionReplacementValue {
        fmt.Println("registrar.iana_id is a placeholder, not the real ID")
    }
    for role, fields := range domain.Redacted.Entities {
        for field, method := range fields {
            fmt.Printf("%s.%s withheld by %s\n", role, field, method)
        }
    }
}
```

`RedactionMethod` is a plain string type (`removal`, `emptyValue`, `partialValue`,
`replacementValue`). A method we do not recognise passes through unchanged rather than
failing to decode.

### IP Address Lookup

```go
ip, err := client.IP(ctx, "8.8.8.8")
fmt.Println(*ip.Name)        // "LVLT-GOGL-8-8-8"
fmt.Println(*ip.Country)     // "US"
fmt.Println(ip.CIDR)         // ["8.8.8.0/24"]
```

Pass a CIDR block to look that network up instead of the most specific allocation
covering an address:

```go
ip, err := client.IP(ctx, "8.8.8.0/24")
```

`Geofeed` is the RFC 8805 geofeed URL the network publishes, returned as published —
never fetched, never inherited from a parent network, `nil` when there is none.

### ASN Lookup

```go
asn, err := client.ASN(ctx, "15169")    // or "AS15169"
fmt.Println(*asn.Name)                   // "GOOGLE"
fmt.Println(*asn.Country)                // "US", derived from the contacts
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

`WithFollow` and `WithoutWhois` apply to every domain in the request. A failing domain
does not fail the call — check each entry's `Status`.

```go
resp, err := client.BulkDomains(ctx, []string{"google.com", "github.com"}, rdapapi.WithFollow())
for _, r := range resp.Results {
    if r.Status == "success" {
        fmt.Printf("%s — %s\n", r.Domain, *r.Data.Registrar.Name)
    }
}
```

## Supported TLDs Catalog

List every TLD the API can resolve, with the date support was added and a qualitative summary of which fields the registry's RDAP server populates. Does not count against your monthly quota.

```go
tlds, err := client.TLDs(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("%d TLDs, coverage %.0f%%\n", tlds.Meta.Count, tlds.Meta.Coverage*100)
for _, tld := range tlds.Data {
    if tld.FieldAvailability != nil {
        fmt.Printf("%s: expires_at=%s\n", tld.TLD, tld.FieldAvailability.ExpiresAt)
    }
}
```

Filter to recent additions or to a single registry:

```go
recent, _ := client.TLDs(ctx, rdapapi.WithSince("2026-04-01T00:00:00Z"))
verisign, _ := client.TLDs(ctx, rdapapi.WithServer("rdap.verisign.com"))
```

Pass back the previous ETag to skip the transfer when nothing has changed. The method returns `(nil, nil)` on 304:

```go
first, _ := client.TLDs(ctx)
later, _ := client.TLDs(ctx, rdapapi.WithIfNoneMatch(first.ETag))
if later == nil {
    fmt.Println("No change since last poll")
}
```

Look up a single TLD:

```go
com, _ := client.TLD(ctx, "com")
fmt.Println(com.Data.Server)   // "rdap.verisign.com"
fmt.Println(com.Data.Protocol) // "rdap", or "whois" for a ccTLD with no RDAP server
```

`Server` is the hostname that answers for the TLD; it is what `WithServer` filters on
and what a lookup's `Meta.Server` returns. `RDAPServerHost` is deprecated and `nil` on a
WHOIS entry, as is `RDAPServerURL`. `FieldAvailability` is `nil` both when we lack
observations and when `Protocol` is `whois`, since the measurement is taken from RDAP
responses.

## Health Check

```go
pong, err := client.Ping(ctx)   // {"status":"ok"}
```

Consumes no quota and makes no upstream call, but still authenticates: a valid API
key is required.

## Error Handling

All API errors are returned as typed errors that can be checked with `errors.As`:

```go
domain, err := client.Domain(ctx, "example.nope")
if err != nil {
    // Check NotSupportedError first: it's a 404 variant for uncovered namespaces.
    var notSupported *rdapapi.NotSupportedError
    if errors.As(err, &notSupported) {
        fmt.Println("TLD not covered by RDAP:", notSupported.Message)
        return
    }

    var notFound *rdapapi.NotFoundError
    if errors.As(err, &notFound) {
        fmt.Println("Domain not registered:", notFound.Message)
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

Three types are narrower variants of another, and unwrap to it, so code that already
catches the broader type still matches:

| Narrow type | Unwraps to | Raised on |
|---|---|---|
| `NotSupportedError` | `NotFoundError` | `404 not_supported` |
| `PlanUpgradeRequiredError` | `SubscriptionRequiredError` | `403 plan_upgrade_required` |
| `QuotaExceededError` | `RateLimitError` | `429 quota_exceeded` |

Check the narrow type first, as in the example above.

| Error Type | HTTP Status | `Code` values |
|---|---|---|
| `ValidationError` | 400 | `invalid_domain`, `invalid_ip`, `invalid_asn`, `invalid_nameserver`, `invalid_handle`, `invalid_prefix`, `invalid_since`, `bad_request` |
| `AuthenticationError` | 401 | `unauthenticated` |
| `SubscriptionRequiredError` | 403 | `subscription_required`, `unknown_error` (see below) |
| `PlanUpgradeRequiredError` | 403 | `plan_upgrade_required` |
| `NotFoundError` | 404 | `not_found` |
| `NotSupportedError` | 404 | `not_supported` |
| `RateLimitError` | 429 | `rate_limit_exceeded`, `too_many_requests` |
| `QuotaExceededError` | 429 | `quota_exceeded` |
| `RequestFailedError` | 422 | `request_failed` |
| `UpstreamError` | 502 | `lookup_failed`, `bad_gateway` |
| `TemporarilyUnavailableError` | 503 | `temporarily_unavailable`, `service_unavailable` |
| `TimeoutError` | 504 | `gateway_timeout` |
| `APIError` | any other | `method_not_allowed` (405), `payload_too_large` (413), `server_error` (5xx) |

All typed errors embed `*APIError`, which provides `StatusCode`, `Code`, `Message`,
`RetryAfter` and `Errors`. Branch on `Code`, never on `Message`: the message is display
text that may be reworded at any time, and any code can answer any endpoint.

Every typed error also unwraps to `*APIError`, so one catch-all matches anything the
SDK returns and is the right last arm after the narrower checks:

```go
var apiErr *rdapapi.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("HTTP %d %s: %s\n", apiErr.StatusCode, apiErr.Code, apiErr.Message)
}
```

`SubscriptionRequiredError` is the one to read carefully: a 403 does not always mean
"buy a plan". The API raises `subscription_required` for an account with no active
subscription, but a 403 from the CDN edge — an IP block, a WAF rule — never reaches
the API and carries no JSON body, so `Code` is `unknown_error` on an account whose
billing is perfectly fine. Branch on `Code` before pointing anyone at the pricing page:

```go
var subErr *rdapapi.SubscriptionRequiredError
if errors.As(err, &subErr) {
    if subErr.Code == "subscription_required" {
        fmt.Println("No active subscription. Visit https://rdapapi.io/pricing")
    } else {
        // Blocked before the API saw the request; subscribing will not help.
        fmt.Printf("Forbidden: %s (code: %s)\n", subErr.Message, subErr.Code)
    }
}
```

`RetryAfter` is the wait in seconds, and zero when the API gave none. It comes from
the `Retry-After` header in either form RFC 9110 allows — a seconds count, or an
HTTP-date, which arrives when the API propagates a registry's own header verbatim.
The header wins whenever it parses, a literal `0` ("retry now") included, and a date
already past clamps to zero; the body's `retry_after` is read only when the header is
absent or in neither form. `Errors` names the fields that failed validation on a
`RequestFailedError`:

```go
var reqFailed *rdapapi.RequestFailedError
if errors.As(err, &reqFailed) {
    for field, messages := range reqFailed.Errors {
        fmt.Printf("%s: %v\n", field, messages)
    }
}
```

An error body that is not JSON — which the CDN edge can still produce — yields the type
for the status with `Code` set to `unknown_error`. Treat it as a retryable transport
failure.

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
