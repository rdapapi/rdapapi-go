package rdapapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://rdapapi.io/api/v1"
	defaultTimeout = 30 * time.Second
)

// Client is the RDAP API client.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

// Option configures the Client.
type Option func(*Client)

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient creates a new RDAP API client.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "rdapapi-go/" + Version,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// requestOptions holds per-request options.
type requestOptions struct {
	follow      bool
	noWhois     bool
	since       string
	server      string
	ifNoneMatch string
}

// RequestOption configures a single request.
type RequestOption func(*requestOptions)

// WithFollow enables registrar follow-through for domain lookups.
func WithFollow() RequestOption {
	return func(o *requestOptions) { o.follow = true }
}

// WithoutWhois refuses the WHOIS fallback on domain lookups, so a TLD with no
// RDAP server returns a *NotSupportedError instead of an answer read over
// WHOIS. Applies to domain and bulk domain lookups only.
func WithoutWhois() RequestOption {
	return func(o *requestOptions) { o.noWhois = true }
}

// WithSince filters TLDs to those added strictly after the given ISO 8601
// timestamp. Applies to TLDs only.
func WithSince(since string) RequestOption {
	return func(o *requestOptions) { o.since = since }
}

// WithServer filters TLDs to those served by the given RDAP server hostname
// (case-insensitive). Applies to TLDs only.
func WithServer(host string) RequestOption {
	return func(o *requestOptions) { o.server = host }
}

// WithIfNoneMatch sets the If-None-Match header for a conditional request.
// When the server's ETag matches, the call returns (nil, nil). Applies to
// TLDs only.
func WithIfNoneMatch(etag string) RequestOption {
	return func(o *requestOptions) { o.ifNoneMatch = etag }
}

func (c *Client) doGet(ctx context.Context, path string, query map[string]string) ([]byte, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("rdapapi: creating request: %w", err)
	}

	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	return c.doRequest(req)
}

func (c *Client) doPost(ctx context.Context, path string, body any) ([]byte, error) {
	url := c.baseURL + path

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("rdapapi: marshaling body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("rdapapi: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.doRequest(req)
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rdapapi: sending request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rdapapi: reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, c.handleError(resp, data)
	}

	return data, nil
}

// doConditionalGet performs a GET that may return 304 Not Modified. Returns
// body, etag, notModified flag, and error. When notModified is true, the
// caller should skip decoding and return nil to the user.
func (c *Client) doConditionalGet(ctx context.Context, path string, query map[string]string, ifNoneMatch string) ([]byte, string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, "", false, fmt.Errorf("rdapapi: creating request: %w", err)
	}
	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", false, fmt.Errorf("rdapapi: sending request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body

	if resp.StatusCode == http.StatusNotModified {
		return nil, resp.Header.Get("ETag"), true, nil
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", false, fmt.Errorf("rdapapi: reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, "", false, c.handleError(resp, data)
	}

	return data, resp.Header.Get("ETag"), false, nil
}

func (c *Client) handleError(resp *http.Response, body []byte) error {
	var errBody struct {
		Error      string              `json:"error"`
		Message    string              `json:"message"`
		RetryAfter *int                `json:"retry_after"`
		Errors     map[string][]string `json:"errors"`
	}
	// json.Unmarshal keeps every field it decoded before it reports a type
	// mismatch, so an unexpected retry_after or errors must not cost us the
	// code and message that parsed: fill in only what is still missing. A
	// non-JSON body — which the CDN edge can still produce — leaves both
	// empty, and the status code is then all we have to go on.
	if err := json.Unmarshal(body, &errBody); err != nil {
		if errBody.Error == "" {
			errBody.Error = "unknown_error"
		}
		if errBody.Message == "" {
			errBody.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}

	// The header wins whenever it parses, a literal 0 ("retry now") included,
	// so the body is a fallback for an absent or unreadable header only.
	retryAfter, fromHeader := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
	if !fromHeader && errBody.RetryAfter != nil {
		retryAfter = *errBody.RetryAfter
	}

	return newError(resp.StatusCode, errBody.Error, errBody.Message, retryAfter, errBody.Errors)
}

// parseRetryAfter reads a Retry-After value in either form RFC 9110 allows: a
// delta-seconds count, or an HTTP-date, which reaches a caller verbatim when
// the API propagates a registry's own header. The bool reports whether the
// value was in either form; a wait already past clamps to 0 rather than going
// negative.
func parseRetryAfter(value string, now time.Time) (int, bool) {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		return max(seconds, 0), true
	}

	deadline, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}
	wait := deadline.Sub(now)
	if wait <= 0 {
		return 0, true
	}
	// Round up: a fraction of a second is still a second to wait.
	return int((wait + time.Second - 1) / time.Second), true
}

// Domain looks up registration data for a domain name, over RDAP or — for the
// TLDs with no RDAP server — WHOIS. Meta.Source says which answered. Pass
// WithoutWhois to refuse the fallback, and WithFollow to merge in the contacts
// held by the registrar.
func (c *Client) Domain(ctx context.Context, name string, opts ...RequestOption) (*DomainResponse, error) {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}

	query := map[string]string{}
	if o.follow {
		query["follow"] = "true"
	}
	if o.noWhois {
		query["whois"] = "false"
	}

	data, err := c.doGet(ctx, "/domain/"+name, query)
	if err != nil {
		return nil, err
	}

	var result DomainResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("rdapapi: decoding response: %w", err)
	}
	return &result, nil
}

// IP looks up RDAP registration data for an IP address.
func (c *Client) IP(ctx context.Context, address string) (*IpResponse, error) {
	data, err := c.doGet(ctx, "/ip/"+address, nil)
	if err != nil {
		return nil, err
	}

	var result IpResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("rdapapi: decoding response: %w", err)
	}
	return &result, nil
}

// ASN looks up RDAP registration data for an ASN.
// Accepts a plain number ("15169") or with prefix ("AS15169").
func (c *Client) ASN(ctx context.Context, number string) (*AsnResponse, error) {
	value := strings.TrimPrefix(strings.ToUpper(number), "AS")
	if _, err := strconv.Atoi(value); err != nil {
		return nil, fmt.Errorf("rdapapi: invalid ASN: %q", number)
	}

	data, err := c.doGet(ctx, "/asn/"+value, nil)
	if err != nil {
		return nil, err
	}

	var result AsnResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("rdapapi: decoding response: %w", err)
	}
	return &result, nil
}

// Nameserver looks up RDAP registration data for a nameserver.
func (c *Client) Nameserver(ctx context.Context, host string) (*NameserverResponse, error) {
	data, err := c.doGet(ctx, "/nameserver/"+host, nil)
	if err != nil {
		return nil, err
	}

	var result NameserverResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("rdapapi: decoding response: %w", err)
	}
	return &result, nil
}

// Entity looks up RDAP registration data for an entity by handle.
func (c *Client) Entity(ctx context.Context, handle string) (*EntityResponse, error) {
	data, err := c.doGet(ctx, "/entity/"+handle, nil)
	if err != nil {
		return nil, err
	}

	var result EntityResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("rdapapi: decoding response: %w", err)
	}
	return &result, nil
}

// TLDs lists every TLD the API can resolve via RDAP.
//
// Does not count against the monthly quota. Use WithSince, WithServer, and
// WithIfNoneMatch to filter or skip unchanged transfers. Returns (nil, nil)
// when an If-None-Match ETag matches the server's current value (HTTP 304).
func (c *Client) TLDs(ctx context.Context, opts ...RequestOption) (*TldListResponse, error) {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}

	query := map[string]string{}
	if o.since != "" {
		query["since"] = o.since
	}
	if o.server != "" {
		query["server"] = o.server
	}

	data, etag, notModified, err := c.doConditionalGet(ctx, "/tlds", query, o.ifNoneMatch)
	if err != nil {
		return nil, err
	}
	if notModified {
		return nil, nil
	}

	var result TldListResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("rdapapi: decoding response: %w", err)
	}
	result.ETag = etag
	return &result, nil
}

// TLD returns catalog metadata for a single TLD.
//
// Does not count against the monthly quota. Returns (nil, nil) when an
// If-None-Match ETag matches (HTTP 304). Returns a *NotFoundError when no
// RDAP server is registered for the TLD.
func (c *Client) TLD(ctx context.Context, tld string, opts ...RequestOption) (*TldResponse, error) {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}

	data, etag, notModified, err := c.doConditionalGet(ctx, "/tlds/"+tld, nil, o.ifNoneMatch)
	if err != nil {
		return nil, err
	}
	if notModified {
		return nil, nil
	}

	var result TldResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("rdapapi: decoding response: %w", err)
	}
	result.ETag = etag
	return &result, nil
}

// BulkDomains looks up multiple domains in a single request.
// Requires a Pro or Business plan. Up to 10 domains per call.
//
// A failing domain does not fail the request: the call returns successfully and
// each entry carries its own Status, so check every entry. WithFollow and
// WithoutWhois apply to every domain in the request.
func (c *Client) BulkDomains(ctx context.Context, domains []string, opts ...RequestOption) (*BulkDomainResponse, error) {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}

	body := map[string]any{"domains": domains}
	if o.follow {
		body["follow"] = true
	}
	if o.noWhois {
		body["whois"] = false
	}

	data, err := c.doPost(ctx, "/domains/bulk", body)
	if err != nil {
		return nil, err
	}

	var result BulkDomainResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("rdapapi: decoding response: %w", err)
	}

	// Merge meta from result level into data for each successful result.
	for i := range result.Results {
		r := &result.Results[i]
		if r.Status == "success" && r.Data != nil && r.RawMeta != nil {
			r.Data.Meta = *r.RawMeta
			r.RawMeta = nil
		}
	}

	return &result, nil
}

// Ping reports whether the API is reachable. It consumes no quota and makes no
// upstream RDAP call, but still authenticates: a valid API key is required.
func (c *Client) Ping(ctx context.Context) (*PingResponse, error) {
	data, err := c.doGet(ctx, "/ping", nil)
	if err != nil {
		return nil, err
	}

	var result PingResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("rdapapi: decoding response: %w", err)
	}
	return &result, nil
}
