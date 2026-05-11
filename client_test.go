package rdapapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockServer creates a test server that returns predefined responses.
func mockServer(t *testing.T, routes map[string]mockRoute) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip query string for route matching.
		path := r.URL.Path

		route, ok := routes[path]
		if !ok {
			t.Errorf("unexpected request path: %s", path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Let the route handler validate request details if needed.
		if route.check != nil {
			route.check(t, r)
		}

		for k, v := range route.headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(route.status)
		_, _ = w.Write([]byte(route.body))
	}))
}

type mockRoute struct {
	status  int
	body    string
	headers map[string]string
	check   func(t *testing.T, r *http.Request)
}

// --- Constructor tests ---

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("test-key")

	if c.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "test-key")
	}
	if c.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, defaultBaseURL)
	}
	if c.httpClient.Timeout != defaultTimeout {
		t.Errorf("Timeout = %v, want %v", c.httpClient.Timeout, defaultTimeout)
	}
	if !strings.HasPrefix(c.userAgent, "rdapapi-go/") {
		t.Errorf("userAgent = %q, want prefix %q", c.userAgent, "rdapapi-go/")
	}
}

func TestNewClientWithBaseURL(t *testing.T) {
	c := NewClient("key", WithBaseURL("https://custom.api.com/v1"))
	if c.baseURL != "https://custom.api.com/v1" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api.com/v1")
	}
}

func TestNewClientWithTimeout(t *testing.T) {
	c := NewClient("key", WithTimeout(5*time.Second))
	if c.httpClient.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want %v", c.httpClient.Timeout, 5*time.Second)
	}
}

func TestNewClientWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 99 * time.Second}
	c := NewClient("key", WithHTTPClient(custom))
	if c.httpClient != custom {
		t.Error("httpClient was not set to custom client")
	}
}

// --- Header tests ---

func TestRequestHeaders(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domain/example.com": {
			status: 200,
			body:   `{"domain":"example.com","status":[],"registrar":{},"dates":{},"nameservers":[],"dnssec":false,"entities":{},"meta":{"rdap_server":"","raw_rdap_url":"","cached":false,"cache_expires":""}}`,
			check: func(t *testing.T, r *http.Request) {
				t.Helper()
				if got := r.Header.Get("Authorization"); got != "Bearer test-key-123" {
					t.Errorf("Authorization = %q, want %q", got, "Bearer test-key-123")
				}
				if got := r.Header.Get("User-Agent"); !strings.HasPrefix(got, "rdapapi-go/") {
					t.Errorf("User-Agent = %q, want prefix %q", got, "rdapapi-go/")
				}
				if got := r.Header.Get("Accept"); got != "application/json" {
					t.Errorf("Accept = %q, want %q", got, "application/json")
				}
			},
		},
	})
	defer srv.Close()

	c := NewClient("test-key-123", WithBaseURL(srv.URL))
	_, err := c.Domain(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Domain() error: %v", err)
	}
}

// --- Domain tests ---

func TestDomainLookup(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domain/google.com": {
			status: 200,
			body:   `{"domain":"google.com","handle":"D-123","status":["client transfer prohibited"],"registrar":{"name":"MarkMonitor"},"dates":{"registered":"1997-09-15T04:00:00Z"},"nameservers":["ns1.google.com"],"dnssec":false,"entities":{},"meta":{"rdap_server":"https://rdap.markmonitor.com","raw_rdap_url":"https://rdap.markmonitor.com/rdap/domain/google.com","cached":false,"cache_expires":""}}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	resp, err := c.Domain(context.Background(), "google.com")
	if err != nil {
		t.Fatalf("Domain() error: %v", err)
	}

	if resp.Domain != "google.com" {
		t.Errorf("Domain = %q, want %q", resp.Domain, "google.com")
	}
	if resp.Registrar.Name == nil || *resp.Registrar.Name != "MarkMonitor" {
		t.Errorf("Registrar.Name = %v, want MarkMonitor", resp.Registrar.Name)
	}
}

func TestDomainWithFollow(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domain/example.com": {
			status: 200,
			body:   `{"domain":"example.com","status":[],"registrar":{},"dates":{},"nameservers":[],"dnssec":false,"entities":{},"meta":{"rdap_server":"","raw_rdap_url":"","cached":false,"cache_expires":"","followed":true}}`,
			check: func(t *testing.T, r *http.Request) {
				t.Helper()
				if got := r.URL.Query().Get("follow"); got != "true" {
					t.Errorf("follow query = %q, want %q", got, "true")
				}
			},
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	resp, err := c.Domain(context.Background(), "example.com", WithFollow())
	if err != nil {
		t.Fatalf("Domain() error: %v", err)
	}

	if resp.Meta.Followed == nil || !*resp.Meta.Followed {
		t.Error("Meta.Followed should be true")
	}
}

func TestDomainWithoutFollow(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domain/example.com": {
			status: 200,
			body:   `{"domain":"example.com","status":[],"registrar":{},"dates":{},"nameservers":[],"dnssec":false,"entities":{},"meta":{"rdap_server":"","raw_rdap_url":"","cached":false,"cache_expires":""}}`,
			check: func(t *testing.T, r *http.Request) {
				t.Helper()
				if got := r.URL.Query().Get("follow"); got != "" {
					t.Errorf("follow query = %q, want empty (no follow)", got)
				}
			},
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Domain(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Domain() error: %v", err)
	}
}

// --- IP tests ---

func TestIPLookup(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/ip/8.8.8.8": {
			status: 200,
			body:   `{"handle":"NET-8-8-8-0-1","name":"LVLT-GOGL-8-8-8","start_address":"8.8.8.0","end_address":"8.8.8.255","ip_version":"v4","status":["active"],"dates":{},"entities":{},"cidr":["8.8.8.0/24"],"remarks":[],"meta":{"rdap_server":"https://rdap.arin.net","raw_rdap_url":"","cached":false,"cache_expires":""}}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	resp, err := c.IP(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("IP() error: %v", err)
	}

	if resp.Handle == nil || *resp.Handle != "NET-8-8-8-0-1" {
		t.Errorf("Handle = %v, want NET-8-8-8-0-1", resp.Handle)
	}
	if resp.IPVersion == nil || *resp.IPVersion != "v4" {
		t.Errorf("IPVersion = %v, want v4", resp.IPVersion)
	}
}

// --- ASN tests ---

func TestASNLookup(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/asn/15169": {
			status: 200,
			body:   `{"handle":"AS15169","name":"GOOGLE","start_autnum":15169,"end_autnum":15169,"status":["active"],"dates":{},"entities":{},"remarks":[],"meta":{"rdap_server":"https://rdap.arin.net","raw_rdap_url":"","cached":false,"cache_expires":""}}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	resp, err := c.ASN(context.Background(), "15169")
	if err != nil {
		t.Fatalf("ASN() error: %v", err)
	}

	if resp.Handle == nil || *resp.Handle != "AS15169" {
		t.Errorf("Handle = %v, want AS15169", resp.Handle)
	}
}

func TestASNLookupStripsPrefix(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/asn/15169": {
			status: 200,
			body:   `{"handle":"AS15169","name":"GOOGLE","start_autnum":15169,"end_autnum":15169,"status":[],"dates":{},"entities":{},"remarks":[],"meta":{"rdap_server":"","raw_rdap_url":"","cached":false,"cache_expires":""}}`,
			check: func(t *testing.T, r *http.Request) {
				t.Helper()
				// The path should be /asn/15169 (prefix stripped).
				if !strings.HasSuffix(r.URL.Path, "/asn/15169") {
					t.Errorf("path = %q, want suffix /asn/15169", r.URL.Path)
				}
			},
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))

	// Test with "AS" prefix.
	_, err := c.ASN(context.Background(), "AS15169")
	if err != nil {
		t.Fatalf("ASN(AS15169) error: %v", err)
	}
}

func TestASNLookupLowercasePrefix(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/asn/15169": {
			status: 200,
			body:   `{"handle":"AS15169","status":[],"dates":{},"entities":{},"remarks":[],"meta":{"rdap_server":"","raw_rdap_url":"","cached":false,"cache_expires":""}}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))

	// Test with lowercase "as" prefix.
	_, err := c.ASN(context.Background(), "as15169")
	if err != nil {
		t.Fatalf("ASN(as15169) error: %v", err)
	}
}

func TestASNInvalidInput(t *testing.T) {
	c := NewClient("key")

	_, err := c.ASN(context.Background(), "NOTANUMBER")
	if err == nil {
		t.Fatal("expected error for non-numeric ASN")
	}
	if !strings.Contains(err.Error(), "invalid ASN") {
		t.Errorf("error = %q, want 'invalid ASN'", err.Error())
	}

	_, err = c.ASN(context.Background(), "AS")
	if err == nil {
		t.Fatal("expected error for empty ASN after prefix strip")
	}
}

// --- Nameserver tests ---

func TestNameserverLookup(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/nameserver/ns1.google.com": {
			status: 200,
			body:   `{"ldh_name":"ns1.google.com","handle":"NS-001","ip_addresses":{"v4":["216.239.32.10"],"v6":[]},"status":["active"],"dates":{},"entities":{},"meta":{"rdap_server":"","raw_rdap_url":"","cached":false,"cache_expires":""}}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	resp, err := c.Nameserver(context.Background(), "ns1.google.com")
	if err != nil {
		t.Fatalf("Nameserver() error: %v", err)
	}

	if resp.LDHName != "ns1.google.com" {
		t.Errorf("LDHName = %q, want %q", resp.LDHName, "ns1.google.com")
	}
}

// --- Entity tests ---

func TestEntityLookup(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/entity/GOGL": {
			status: 200,
			body:   `{"handle":"GOGL","name":"Google LLC","roles":["registrant"],"status":["active"],"dates":{},"remarks":[],"public_ids":[],"entities":{},"autnums":[],"networks":[],"meta":{"rdap_server":"","raw_rdap_url":"","cached":false,"cache_expires":""}}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	resp, err := c.Entity(context.Background(), "GOGL")
	if err != nil {
		t.Fatalf("Entity() error: %v", err)
	}

	if resp.Handle == nil || *resp.Handle != "GOGL" {
		t.Errorf("Handle = %v, want GOGL", resp.Handle)
	}
	if resp.Name == nil || *resp.Name != "Google LLC" {
		t.Errorf("Name = %v, want Google LLC", resp.Name)
	}
}

// --- Bulk domain tests ---

func TestBulkDomains(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domains/bulk": {
			status: 200,
			body: `{
				"results": [
					{
						"domain": "example.com",
						"status": "success",
						"data": {
							"domain": "example.com",
							"status": ["active"],
							"registrar": {"name": "Test Registrar"},
							"dates": {},
							"nameservers": [],
							"dnssec": false,
							"entities": {},
							"meta": {"rdap_server": "", "raw_rdap_url": "", "cached": false, "cache_expires": ""}
						},
						"meta": {
							"rdap_server": "https://rdap.verisign.com",
							"raw_rdap_url": "https://rdap.verisign.com/com/v1/domain/example.com",
							"cached": true,
							"cache_expires": "2024-12-01T00:00:00Z"
						}
					},
					{
						"domain": "nope.example",
						"status": "error",
						"error": "not_found",
						"message": "No RDAP data found"
					}
				],
				"summary": {"total": 2, "successful": 1, "failed": 1}
			}`,
			check: func(t *testing.T, r *http.Request) {
				t.Helper()
				if r.Method != http.MethodPost {
					t.Errorf("Method = %q, want POST", r.Method)
				}
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode body: %v", err)
				}
				domains, ok := body["domains"].([]any)
				if !ok || len(domains) != 2 {
					t.Errorf("body.domains = %v, want 2 domains", body["domains"])
				}
			},
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	resp, err := c.BulkDomains(context.Background(), []string{"example.com", "nope.example"})
	if err != nil {
		t.Fatalf("BulkDomains() error: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("Results len = %d, want 2", len(resp.Results))
	}

	// Verify meta was merged into data.
	r0 := resp.Results[0]
	if r0.Data == nil {
		t.Fatal("Results[0].Data is nil")
	}
	if r0.Data.Meta.RDAPServer != "https://rdap.verisign.com" {
		t.Errorf("Data.Meta.RDAPServer = %q, want %q", r0.Data.Meta.RDAPServer, "https://rdap.verisign.com")
	}
	if !r0.Data.Meta.Cached {
		t.Error("Data.Meta.Cached = false, want true (merged from result level)")
	}
	if r0.RawMeta != nil {
		t.Error("RawMeta should be nil after merge")
	}

	// Error result.
	r1 := resp.Results[1]
	if r1.Status != "error" {
		t.Errorf("Results[1].Status = %q, want error", r1.Status)
	}
	if r1.Data != nil {
		t.Error("Results[1].Data should be nil for error result")
	}

	if resp.Summary.Total != 2 {
		t.Errorf("Summary.Total = %d, want 2", resp.Summary.Total)
	}
}

func TestBulkDomainsWithFollow(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domains/bulk": {
			status: 200,
			body:   `{"results":[],"summary":{"total":0,"successful":0,"failed":0}}`,
			check: func(t *testing.T, r *http.Request) {
				t.Helper()
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode body: %v", err)
				}
				follow, ok := body["follow"].(bool)
				if !ok || !follow {
					t.Errorf("body.follow = %v, want true", body["follow"])
				}
			},
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.BulkDomains(context.Background(), []string{"example.com"}, WithFollow())
	if err != nil {
		t.Fatalf("BulkDomains() error: %v", err)
	}
}

// --- Error handling tests ---

func TestErrorResponses(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		headers  map[string]string
		checkErr func(t *testing.T, err error)
	}{
		{
			name:   "400 ValidationError",
			status: 400,
			body:   `{"error":"invalid_input","message":"Invalid domain name"}`,
			checkErr: func(t *testing.T, err error) {
				var target *ValidationError
				if !errors.As(err, &target) {
					t.Fatal("expected ValidationError")
				}
				if target.StatusCode != 400 {
					t.Errorf("StatusCode = %d, want 400", target.StatusCode)
				}
				if target.Code != "invalid_input" {
					t.Errorf("Code = %q, want invalid_input", target.Code)
				}
			},
		},
		{
			name:   "401 AuthenticationError",
			status: 401,
			body:   `{"error":"unauthenticated","message":"Invalid API key"}`,
			checkErr: func(t *testing.T, err error) {
				var target *AuthenticationError
				if !errors.As(err, &target) {
					t.Fatal("expected AuthenticationError")
				}
			},
		},
		{
			name:   "403 SubscriptionRequiredError",
			status: 403,
			body:   `{"error":"subscription_required","message":"No active subscription"}`,
			checkErr: func(t *testing.T, err error) {
				var target *SubscriptionRequiredError
				if !errors.As(err, &target) {
					t.Fatal("expected SubscriptionRequiredError")
				}
			},
		},
		{
			name:   "404 NotFoundError",
			status: 404,
			body:   `{"error":"not_found","message":"No RDAP data found"}`,
			checkErr: func(t *testing.T, err error) {
				var target *NotFoundError
				if !errors.As(err, &target) {
					t.Fatal("expected NotFoundError")
				}
			},
		},
		{
			name:    "429 RateLimitError",
			status:  429,
			body:    `{"error":"rate_limited","message":"Rate limit exceeded"}`,
			headers: map[string]string{"Retry-After": "30"},
			checkErr: func(t *testing.T, err error) {
				var target *RateLimitError
				if !errors.As(err, &target) {
					t.Fatal("expected RateLimitError")
				}
				if target.RetryAfter != 30 {
					t.Errorf("RetryAfter = %d, want 30", target.RetryAfter)
				}
			},
		},
		{
			name:   "429 RateLimitError without Retry-After",
			status: 429,
			body:   `{"error":"rate_limited","message":"Rate limit exceeded"}`,
			checkErr: func(t *testing.T, err error) {
				var target *RateLimitError
				if !errors.As(err, &target) {
					t.Fatal("expected RateLimitError")
				}
				if target.RetryAfter != 0 {
					t.Errorf("RetryAfter = %d, want 0", target.RetryAfter)
				}
			},
		},
		{
			name:   "502 UpstreamError",
			status: 502,
			body:   `{"error":"upstream_error","message":"Upstream RDAP server failed"}`,
			checkErr: func(t *testing.T, err error) {
				var target *UpstreamError
				if !errors.As(err, &target) {
					t.Fatal("expected UpstreamError")
				}
			},
		},
		{
			name:    "503 TemporarilyUnavailableError",
			status:  503,
			body:    `{"error":"temporarily_unavailable","message":"Data for this domain is temporarily unavailable."}`,
			headers: map[string]string{"Retry-After": "300"},
			checkErr: func(t *testing.T, err error) {
				var target *TemporarilyUnavailableError
				if !errors.As(err, &target) {
					t.Fatal("expected TemporarilyUnavailableError")
				}
				if target.RetryAfter != 300 {
					t.Errorf("RetryAfter = %d, want 300", target.RetryAfter)
				}
			},
		},
		{
			name:   "503 TemporarilyUnavailableError without Retry-After",
			status: 503,
			body:   `{"error":"temporarily_unavailable","message":"Data for this domain is temporarily unavailable."}`,
			checkErr: func(t *testing.T, err error) {
				var target *TemporarilyUnavailableError
				if !errors.As(err, &target) {
					t.Fatal("expected TemporarilyUnavailableError")
				}
				if target.RetryAfter != 0 {
					t.Errorf("RetryAfter = %d, want 0", target.RetryAfter)
				}
			},
		},
		{
			name:   "500 unknown error",
			status: 500,
			body:   `{"error":"server_error","message":"Internal server error"}`,
			checkErr: func(t *testing.T, err error) {
				// Should be a base APIError.
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatal("expected APIError for 500")
				}
				if apiErr.StatusCode != 500 {
					t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
				}
				if !strings.Contains(err.Error(), "Internal server error") {
					t.Errorf("error message = %q, want to contain 'Internal server error'", err.Error())
				}
			},
		},
		{
			name:   "invalid JSON body",
			status: 400,
			body:   `not json at all`,
			checkErr: func(t *testing.T, err error) {
				var target *ValidationError
				if !errors.As(err, &target) {
					t.Fatal("expected ValidationError even with invalid JSON body")
				}
				// Falls back to "HTTP 400" message.
				if target.Code != "unknown_error" {
					t.Errorf("Code = %q, want unknown_error", target.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := mockServer(t, map[string]mockRoute{
				"/domain/test.com": {
					status:  tt.status,
					body:    tt.body,
					headers: tt.headers,
				},
			})
			defer srv.Close()

			c := NewClient("key", WithBaseURL(srv.URL))
			_, err := c.Domain(context.Background(), "test.com")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			tt.checkErr(t, err)
		})
	}
}

// --- Context cancellation ---

func TestContextCancellation(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domain/slow.com": {
			status: 200,
			body:   `{"domain":"slow.com","status":[],"registrar":{},"dates":{},"nameservers":[],"dnssec":false,"entities":{},"meta":{"rdap_server":"","raw_rdap_url":"","cached":false,"cache_expires":""}}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := c.Domain(ctx, "slow.com")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// --- Invalid URL ---

func TestInvalidBaseURL(t *testing.T) {
	c := NewClient("key", WithBaseURL("://invalid"))
	_, err := c.Domain(context.Background(), "test.com")
	if err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}

// --- Version ---

func TestVersionIsSet(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestUserAgentContainsVersion(t *testing.T) {
	c := NewClient("key")
	want := "rdapapi-go/" + Version
	if c.userAgent != want {
		t.Errorf("userAgent = %q, want %q", c.userAgent, want)
	}
}

// --- BulkDomains meta merge edge cases ---

func TestBulkDomainsNoMetaMergeOnError(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domains/bulk": {
			status: 200,
			body: `{
				"results": [
					{
						"domain": "fail.example",
						"status": "error",
						"error": "not_found",
						"message": "No data"
					}
				],
				"summary": {"total": 1, "successful": 0, "failed": 1}
			}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	resp, err := c.BulkDomains(context.Background(), []string{"fail.example"})
	if err != nil {
		t.Fatalf("BulkDomains() error: %v", err)
	}

	if resp.Results[0].Data != nil {
		t.Error("Data should be nil for error result")
	}
}

func TestBulkDomainsSuccessWithoutMeta(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domains/bulk": {
			status: 200,
			body: `{
				"results": [
					{
						"domain": "example.com",
						"status": "success",
						"data": {
							"domain": "example.com",
							"status": [],
							"registrar": {},
							"dates": {},
							"nameservers": [],
							"dnssec": false,
							"entities": {},
							"meta": {"rdap_server": "original", "raw_rdap_url": "", "cached": false, "cache_expires": ""}
						}
					}
				],
				"summary": {"total": 1, "successful": 1, "failed": 0}
			}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	resp, err := c.BulkDomains(context.Background(), []string{"example.com"})
	if err != nil {
		t.Fatalf("BulkDomains() error: %v", err)
	}

	// No result-level meta, so data.meta should remain original.
	if resp.Results[0].Data.Meta.RDAPServer != "original" {
		t.Errorf("Meta.RDAPServer = %q, want 'original'", resp.Results[0].Data.Meta.RDAPServer)
	}
}

// --- POST body for bulk ---

func TestBulkDomainsPostBody(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domains/bulk": {
			status: 200,
			body:   `{"results":[],"summary":{"total":0,"successful":0,"failed":0}}`,
			check: func(t *testing.T, r *http.Request) {
				t.Helper()
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				domains := body["domains"].([]any)
				if len(domains) != 3 {
					t.Errorf("domains len = %d, want 3", len(domains))
				}
				// follow should not be present when not set.
				if _, ok := body["follow"]; ok {
					t.Error("follow should not be in body when WithFollow is not used")
				}
			},
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.BulkDomains(context.Background(), []string{"a.com", "b.com", "c.com"})
	if err != nil {
		t.Fatalf("BulkDomains() error: %v", err)
	}
}

// --- Unmarshal error paths ---

func TestDomainInvalidJSON(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domain/bad.com": {status: 200, body: `not json`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Domain(context.Background(), "bad.com")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
	if !strings.Contains(err.Error(), "decoding response") {
		t.Errorf("error = %q, want 'decoding response'", err.Error())
	}
}

func TestIPInvalidJSON(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/ip/1.2.3.4": {status: 200, body: `{bad`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.IP(context.Background(), "1.2.3.4")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
	if !strings.Contains(err.Error(), "decoding response") {
		t.Errorf("error = %q, want 'decoding response'", err.Error())
	}
}

func TestASNInvalidJSON(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/asn/99999": {status: 200, body: `{bad`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.ASN(context.Background(), "99999")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
	if !strings.Contains(err.Error(), "decoding response") {
		t.Errorf("error = %q, want 'decoding response'", err.Error())
	}
}

func TestNameserverInvalidJSON(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/nameserver/ns.bad.com": {status: 200, body: `{bad`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Nameserver(context.Background(), "ns.bad.com")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
	if !strings.Contains(err.Error(), "decoding response") {
		t.Errorf("error = %q, want 'decoding response'", err.Error())
	}
}

func TestEntityInvalidJSON(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/entity/BAD": {status: 200, body: `{bad`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Entity(context.Background(), "BAD")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
	if !strings.Contains(err.Error(), "decoding response") {
		t.Errorf("error = %q, want 'decoding response'", err.Error())
	}
}

func TestBulkDomainsInvalidJSON(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domains/bulk": {status: 200, body: `{bad`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.BulkDomains(context.Background(), []string{"a.com"})
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
	if !strings.Contains(err.Error(), "decoding response") {
		t.Errorf("error = %q, want 'decoding response'", err.Error())
	}
}

// --- doPost marshal error ---

func TestBulkDomainsInvalidBody(t *testing.T) {
	c := NewClient("key", WithBaseURL("https://localhost"))

	// Pass an unmarshalable value via a channel (channels can't be JSON marshaled).
	// We can't directly test this through BulkDomains since it always passes valid data.
	// Instead, test the doPost internal path with an invalid base URL to cover the request-creation error.
	c.baseURL = "://invalid"
	_, err := c.BulkDomains(context.Background(), []string{"a.com"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Error responses on each endpoint ---

func TestIPErrorResponse(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/ip/1.2.3.4": {status: 404, body: `{"error":"not_found","message":"No data"}`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.IP(context.Background(), "1.2.3.4")
	if err == nil {
		t.Fatal("expected error")
	}
	var target *NotFoundError
	if !errors.As(err, &target) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestASNErrorResponse(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/asn/99999": {status: 401, body: `{"error":"unauthenticated","message":"Bad key"}`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.ASN(context.Background(), "99999")
	if err == nil {
		t.Fatal("expected error")
	}
	var target *AuthenticationError
	if !errors.As(err, &target) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestNameserverErrorResponse(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/nameserver/ns.bad.com": {status: 429, body: `{"error":"rate_limited","message":"Too many"}`, headers: map[string]string{"Retry-After": "10"}},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Nameserver(context.Background(), "ns.bad.com")
	if err == nil {
		t.Fatal("expected error")
	}
	var target *RateLimitError
	if !errors.As(err, &target) {
		t.Errorf("expected RateLimitError, got %T", err)
	}
}

func TestEntityErrorResponse(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/entity/BAD": {status: 502, body: `{"error":"upstream_error","message":"Upstream fail"}`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Entity(context.Background(), "BAD")
	if err == nil {
		t.Fatal("expected error")
	}
	var target *UpstreamError
	if !errors.As(err, &target) {
		t.Errorf("expected UpstreamError, got %T", err)
	}
}

// --- doPost marshal error (line 96-98) ---

func TestDoPostMarshalError(t *testing.T) {
	c := NewClient("key", WithBaseURL("https://localhost"))

	// Channels cannot be JSON marshaled.
	_, err := c.doPost(context.Background(), "/test", map[string]any{"bad": make(chan int)})
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if !strings.Contains(err.Error(), "marshaling body") {
		t.Errorf("error = %q, want 'marshaling body'", err.Error())
	}
}

// --- doRequest read error (line 120-123) ---

func TestDoRequestReadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100") // Lie about content length.
		w.WriteHeader(200)
		// Write less data than declared, then close — causes io.ReadAll error.
	}))
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Domain(context.Background(), "test.com")
	if err == nil {
		t.Fatal("expected read error")
	}
	if !strings.Contains(err.Error(), "reading response") {
		t.Errorf("error = %q, want 'reading response'", err.Error())
	}
}

// --- Error on POST endpoint ---

func TestBulkDomainsErrorResponse(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domains/bulk": {
			status: 401,
			body:   `{"error":"unauthenticated","message":"Invalid API key"}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.BulkDomains(context.Background(), []string{"example.com"})
	if err == nil {
		t.Fatal("expected error")
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Errorf("expected AuthenticationError, got %T: %v", err, err)
	}
}

// --- NotSupportedError routing ---

func TestDomainReturnsNotSupportedErrorForNotSupportedCode(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/domain/example.nope": {
			status: 404,
			body:   `{"error":"not_supported","message":"The TLD '.nope' is not supported."}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Domain(context.Background(), "example.nope")
	if err == nil {
		t.Fatal("expected error")
	}

	var ns *NotSupportedError
	if !errors.As(err, &ns) {
		t.Fatalf("expected NotSupportedError, got %T", err)
	}

	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatal("NotSupportedError should also satisfy *NotFoundError target")
	}
}

func TestIPReturnsNotSupportedErrorForNotSupportedCode(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/ip/203.0.113.1": {
			status: 404,
			body:   `{"error":"not_supported","message":"No RIR covers this IP range."}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.IP(context.Background(), "203.0.113.1")
	var ns *NotSupportedError
	if !errors.As(err, &ns) {
		t.Fatalf("expected NotSupportedError, got %T", err)
	}
}

// --- TLDs endpoint ---

const tldsBody = `{
  "data": [
    {
      "tld": "com",
      "supported_since": "2026-03-07T00:00:00Z",
      "rdap_server_host": "rdap.verisign.com",
      "rdap_server_url": "https://rdap.verisign.com/com/v1/",
      "field_availability": {
        "registrar": "sometimes",
        "registered_at": "always",
        "expires_at": "always",
        "nameservers": "always",
        "status": "always"
      }
    },
    {
      "tld": "fr",
      "supported_since": "2026-03-07T00:00:00Z",
      "rdap_server_host": "rdap.nic.fr",
      "rdap_server_url": "https://rdap.nic.fr/",
      "field_availability": null
    }
  ],
  "meta": {
    "computed_at": "2026-04-22T10:00:00Z",
    "count": 2,
    "coverage": 0.5,
    "thresholds": {"always": 0.99, "usually": 0.8, "sometimes": 0.0}
  }
}`

const tldBody = `{
  "data": {
    "tld": "com",
    "supported_since": "2026-03-07T00:00:00Z",
    "rdap_server_host": "rdap.verisign.com",
    "rdap_server_url": "https://rdap.verisign.com/com/v1/",
    "field_availability": {
      "registrar": "sometimes",
      "registered_at": "always",
      "expires_at": "always",
      "nameservers": "always",
      "status": "always"
    }
  },
  "meta": {
    "computed_at": "2026-04-22T10:00:00Z",
    "thresholds": {"always": 0.99, "usually": 0.8, "sometimes": 0.0}
  }
}`

func TestTLDsListSuccess(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/tlds": {
			status:  200,
			body:    tldsBody,
			headers: map[string]string{"ETag": `"abc"`},
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	result, err := c.TLDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ETag != `"abc"` {
		t.Errorf("ETag = %q, want %q", result.ETag, `"abc"`)
	}
	if result.Meta.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Meta.Count)
	}
	if result.Meta.Coverage != 0.5 {
		t.Errorf("Coverage = %f, want 0.5", result.Meta.Coverage)
	}
	if result.Meta.Thresholds.Always != 0.99 {
		t.Errorf("Thresholds.Always = %f, want 0.99", result.Meta.Thresholds.Always)
	}
	if len(result.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(result.Data))
	}
	if result.Data[0].TLD != "com" {
		t.Errorf("Data[0].TLD = %q, want %q", result.Data[0].TLD, "com")
	}
	if result.Data[0].FieldAvailability == nil {
		t.Fatal("expected Data[0].FieldAvailability to be non-nil")
	}
	if result.Data[0].FieldAvailability.RegisteredAt != AvailabilityAlways {
		t.Errorf("FieldAvailability.RegisteredAt = %q, want always", result.Data[0].FieldAvailability.RegisteredAt)
	}
	if result.Data[1].FieldAvailability != nil {
		t.Error("expected Data[1].FieldAvailability to be nil")
	}
}

func TestTLDsForwardsSinceAndServer(t *testing.T) {
	var gotQuery string
	srv := mockServer(t, map[string]mockRoute{
		"/tlds": {
			status: 200,
			body:   tldsBody,
			check: func(t *testing.T, r *http.Request) {
				gotQuery = r.URL.RawQuery
			},
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.TLDs(
		context.Background(),
		WithSince("2026-04-01T00:00:00Z"),
		WithServer("rdap.verisign.com"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(gotQuery, "since=2026-04-01T00%3A00%3A00Z") {
		t.Errorf("query = %q, want to contain since=", gotQuery)
	}
	if !strings.Contains(gotQuery, "server=rdap.verisign.com") {
		t.Errorf("query = %q, want to contain server=", gotQuery)
	}
}

func TestTLDsReturnsNilOnNotModified(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/tlds": {
			status: 304,
			check: func(t *testing.T, r *http.Request) {
				if got := r.Header.Get("If-None-Match"); got != `"abc"` {
					t.Errorf("If-None-Match = %q, want %q", got, `"abc"`)
				}
			},
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	result, err := c.TLDs(context.Background(), WithIfNoneMatch(`"abc"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil on 304, got %+v", result)
	}
}

func TestTLDsRaisesTypedErrorOnFailure(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/tlds": {status: 401, body: `{"error":"unauthenticated","message":"Invalid token."}`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.TLDs(context.Background())
	var target *AuthenticationError
	if !errors.As(err, &target) {
		t.Fatalf("expected AuthenticationError, got %T", err)
	}
}

func TestTLDsInvalidBaseURL(t *testing.T) {
	c := NewClient("key", WithBaseURL("://invalid"))
	_, err := c.TLDs(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTLDsSendErrorWhenTransportFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("hijacker not supported")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatal(err)
		}
		conn.Close() //nolint:errcheck // test cleanup
	}))
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.TLDs(context.Background())
	if err == nil {
		t.Fatal("expected transport error")
	}
}

func TestTLDsInvalidJSON(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/tlds": {status: 200, body: `{ not json`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.TLDs(context.Background())
	if err == nil || !strings.Contains(err.Error(), "decoding response") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestTLDsConditionalGetReadErrorOnNotModified(t *testing.T) {
	// Covers the io.ReadAll error path when status is not 304 and not 304 path triggers.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.TLDs(context.Background())
	if err == nil || !strings.Contains(err.Error(), "reading response") {
		t.Fatalf("expected read error, got %v", err)
	}
}

func TestTLDSingleLookup(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/tlds/com": {
			status:  200,
			body:    tldBody,
			headers: map[string]string{"ETag": `"com-1"`},
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	result, err := c.TLD(context.Background(), "com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.TLD != "com" {
		t.Errorf("Data.TLD = %q, want com", result.Data.TLD)
	}
	if result.Meta.Thresholds.Usually != 0.8 {
		t.Errorf("Thresholds.Usually = %f, want 0.8", result.Meta.Thresholds.Usually)
	}
	if result.ETag != `"com-1"` {
		t.Errorf("ETag = %q, want %q", result.ETag, `"com-1"`)
	}
}

func TestTLDSingleLookupNotModified(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/tlds/com": {status: 304},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	result, err := c.TLD(context.Background(), "com", WithIfNoneMatch(`"com-1"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil on 304")
	}
}

func TestTLDSingleLookupNotFound(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/tlds/nope": {
			status: 404,
			body:   `{"error":"not_found","message":"No RDAP server is registered for the TLD 'nope'."}`,
		},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.TLD(context.Background(), "nope")
	var target *NotFoundError
	if !errors.As(err, &target) {
		t.Fatalf("expected NotFoundError, got %T", err)
	}
}

func TestTLDInvalidBaseURL(t *testing.T) {
	c := NewClient("key", WithBaseURL("://invalid"))
	_, err := c.TLD(context.Background(), "com")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTLDInvalidJSON(t *testing.T) {
	srv := mockServer(t, map[string]mockRoute{
		"/tlds/com": {status: 200, body: `{ broken json`},
	})
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.TLD(context.Background(), "com")
	if err == nil || !strings.Contains(err.Error(), "decoding response") {
		t.Fatalf("expected decode error, got %v", err)
	}
}
