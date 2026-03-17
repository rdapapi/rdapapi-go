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
	follow bool
}

// RequestOption configures a single request.
type RequestOption func(*requestOptions)

// WithFollow enables registrar follow-through for domain lookups.
func WithFollow() RequestOption {
	return func(o *requestOptions) { o.follow = true }
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

func (c *Client) handleError(resp *http.Response, body []byte) error {
	var errBody struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errBody); err != nil {
		errBody.Error = "unknown_error"
		errBody.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	var retryAfter int
	if resp.StatusCode == 429 || resp.StatusCode == 503 {
		if v := resp.Header.Get("Retry-After"); v != "" {
			retryAfter, _ = strconv.Atoi(v)
		}
	}

	return newError(resp.StatusCode, errBody.Error, errBody.Message, retryAfter)
}

// Domain looks up RDAP registration data for a domain name.
func (c *Client) Domain(ctx context.Context, name string, opts ...RequestOption) (*DomainResponse, error) {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}

	var query map[string]string
	if o.follow {
		query = map[string]string{"follow": "true"}
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

// BulkDomains looks up multiple domains in a single request.
// Requires a Pro or Business plan. Up to 10 domains per call.
func (c *Client) BulkDomains(ctx context.Context, domains []string, opts ...RequestOption) (*BulkDomainResponse, error) {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}

	body := map[string]any{"domains": domains}
	if o.follow {
		body["follow"] = true
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
