//go:build canary

// The production canary. Unlike the unit tests, which replay frozen fixtures
// and so can only prove the SDK is self-consistent, these probes call
// production through the SDK's public API and fail when the live contract stops
// matching what the SDK models.
//
// The canary build tag keeps them out of `go test ./...` and its coverage gate,
// which run offline. Run them with:
//
//	RDAPAPI_API_KEY=... go test -tags canary -count=1 -v -run TestCanary .
package rdapapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	rdapapi "github.com/rdapapi/rdapapi-go"
)

// retryBackoff is the pause before the single retry a transport error or a 5xx
// earns. A failed assertion is never retried: a field that is wrong is wrong.
const retryBackoff = 5 * time.Second

func TestCanary(t *testing.T) {
	apiKey := os.Getenv("RDAPAPI_API_KEY")
	if apiKey == "" {
		t.Skip("RDAPAPI_API_KEY is not set")
	}
	client := rdapapi.NewClient(apiKey)

	t.Run("ping", func(t *testing.T) {
		pong := probe(t, client.Ping)

		if pong.Status != "ok" {
			t.Errorf("status: expected %q, got %q", "ok", pong.Status)
		}
	})

	t.Run("domain google.com with follow", func(t *testing.T) {
		domain := probe(t, func(ctx context.Context) (*rdapapi.DomainResponse, error) {
			return client.Domain(ctx, "google.com", rdapapi.WithFollow())
		})

		if domain.Meta.Source != rdapapi.ProtocolRDAP {
			t.Errorf("meta.source: expected %q, got %q", rdapapi.ProtocolRDAP, domain.Meta.Source)
		}
		if isBlank(domain.Meta.Server) {
			t.Errorf("meta.server: expected a non-empty host, got %s", describe(domain.Meta.Server))
		}
		if isBlank(domain.Registrar.Name) {
			t.Errorf("registrar.name: expected a non-empty name, got %s", describe(domain.Registrar.Name))
		}
		if roles := filledRoles(domain.Entities); len(roles) == 0 {
			t.Error("entities: expected at least one contact role, got none")
		}
	})

	t.Run("domain google.it over whois", func(t *testing.T) {
		// Decoding at all is half the assertion: this is the answer whose null
		// rdap_server / raw_rdap_url crashed the Python SDK in production.
		domain := probe(t, func(ctx context.Context) (*rdapapi.DomainResponse, error) {
			return client.Domain(ctx, "google.it")
		})

		if domain.Meta.Source != rdapapi.ProtocolWHOIS {
			t.Errorf("meta.source: expected %q, got %q", rdapapi.ProtocolWHOIS, domain.Meta.Source)
		}
		const wantServer = "whois.nic.it"
		if domain.Meta.Server == nil || *domain.Meta.Server != wantServer {
			t.Errorf("meta.server: expected %q, got %s", wantServer, describe(domain.Meta.Server))
		}
		if domain.Meta.RDAPServer != "" {
			t.Errorf("meta.rdap_server: expected absent or null on a whois answer, got %q", domain.Meta.RDAPServer)
		}
		if domain.Meta.RawRDAPURL != "" {
			t.Errorf("meta.raw_rdap_url: expected absent or null on a whois answer, got %q", domain.Meta.RawRDAPURL)
		}
	})

	t.Run("ip 45.83.220.1", func(t *testing.T) {
		ip := probe(t, func(ctx context.Context) (*rdapapi.IpResponse, error) {
			return client.IP(ctx, "45.83.220.1")
		})

		// This holds only while the network's holder keeps publishing an RFC
		// 8805 geofeed; confirm they still do before assuming an SDK bug.
		if isBlank(ip.Geofeed) {
			t.Errorf("geofeed: expected a non-empty URL, got %s", describe(ip.Geofeed))
		}
	})

	t.Run("tld it over whois", func(t *testing.T) {
		tld := probe(t, func(ctx context.Context) (*rdapapi.TldResponse, error) {
			return client.TLD(ctx, "it")
		})

		if tld.Data.Protocol != rdapapi.ProtocolWHOIS {
			t.Errorf("data.protocol: expected %q, got %q", rdapapi.ProtocolWHOIS, tld.Data.Protocol)
		}
		if tld.Data.Server == "" {
			t.Error("data.server: expected a non-empty host, got an empty string")
		}
		if tld.Data.RDAPServerHost != nil {
			t.Errorf("data.rdap_server_host: expected null on a whois tld, got %q", *tld.Data.RDAPServerHost)
		}
		if tld.Data.RDAPServerURL != nil {
			t.Errorf("data.rdap_server_url: expected null on a whois tld, got %q", *tld.Data.RDAPServerURL)
		}
	})
}

// probe runs one live call, retrying it once after a transport error or a 5xx,
// and fails the subtest when the call itself does not come back.
func probe[T any](t *testing.T, call func(context.Context) (*T, error)) *T {
	t.Helper()

	ctx := context.Background()
	result, err := call(ctx)
	if err != nil && isTransient(err) {
		t.Logf("transient failure, retrying once in %s: %v", retryBackoff, err)
		time.Sleep(retryBackoff)
		result, err = call(ctx)
	}
	if err != nil {
		t.Fatalf("expected a response, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected a response, got nil")
	}
	return result
}

// isTransient reports whether err is worth the one retry: a transport failure,
// or any 5xx. A body the SDK could not decode is the contract breaking, which
// is the whole point of the canary, so it fails on the first try.
func isTransient(err error) bool {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
		return false
	}

	var apiErr *rdapapi.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= 500
	}
	return true
}

// filledRoles names the contact roles the answer actually carries.
func filledRoles(entities rdapapi.Entities) []string {
	byRole := map[string]*rdapapi.Contact{
		"registrant":     entities.Registrant,
		"administrative": entities.Administrative,
		"technical":      entities.Technical,
		"billing":        entities.Billing,
		"abuse":          entities.Abuse,
	}

	var roles []string
	for role, contact := range byRole {
		if contact != nil {
			roles = append(roles, role)
		}
	}
	return roles
}

func isBlank(value *string) bool {
	return value == nil || *value == ""
}

// describe renders a nullable string for an assertion message.
func describe(value *string) string {
	if value == nil {
		return "nil"
	}
	return strconv.Quote(*value)
}
