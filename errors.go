package rdapapi

// APIError is the base error type for all RDAP API errors.
//
// Branch on Code, never on Message: the message is display text that may be
// reworded at any time. Any code can answer any endpoint.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	// RetryAfter is the wait in seconds the API asked for, taken from the
	// Retry-After header or the body's retry_after. Zero when it gave none.
	RetryAfter int
	// Errors names the fields that failed validation, set on a 422
	// request_failed and nil otherwise.
	Errors map[string][]string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return e.Message
}

// Unwrap returns nil (APIError is the root error type).
func (e *APIError) Unwrap() error {
	return nil
}

// ValidationError is returned when the input is invalid (HTTP 400).
type ValidationError struct{ *APIError }

// AuthenticationError is returned when the API key is missing or invalid (HTTP 401).
type AuthenticationError struct{ *APIError }

// SubscriptionRequiredError is returned when no active subscription exists (HTTP 403).
//
// For the 403 raised when the account has a subscription but the endpoint needs
// a higher plan, see PlanUpgradeRequiredError, which unwraps to this type so
// errors.As with a *SubscriptionRequiredError target matches both cases.
type SubscriptionRequiredError struct{ *APIError }

// PlanUpgradeRequiredError is returned when the endpoint needs a higher plan
// than the account holds (HTTP 403 plan_upgrade_required) — bulk lookups
// require Pro or Business.
//
// Unwraps to *SubscriptionRequiredError. Use errors.As with a
// *PlanUpgradeRequiredError target to tell "upgrade the plan" from "subscribe".
type PlanUpgradeRequiredError struct{ *SubscriptionRequiredError }

// Unwrap returns the inner *SubscriptionRequiredError so errors.As and errors.Is
// descend through the SubscriptionRequired -> PlanUpgradeRequired chain.
func (e *PlanUpgradeRequiredError) Unwrap() error { return e.SubscriptionRequiredError }

// RequestFailedError is returned when the request body fails validation
// (HTTP 422). Errors names the offending fields.
type RequestFailedError struct{ *APIError }

// NotFoundError is returned when no RDAP data is found for the query (HTTP 404).
//
// The namespace is covered by an RDAP server but no matching record exists.
// For queries where the namespace itself is not covered by RDAP, see
// NotSupportedError, which unwraps to this type so errors.As with a
// *NotFoundError target matches both cases.
type NotFoundError struct{ *APIError }

// NotSupportedError is returned when the query targets a namespace not covered
// by RDAP (HTTP 404).
//
// Unwraps to *NotFoundError, so errors.As(err, &nf) where nf *NotFoundError
// matches this error too. Use errors.As with a *NotSupportedError target when
// you want to distinguish "no RDAP server for this TLD/range" from "namespace
// covered but no record".
type NotSupportedError struct{ *NotFoundError }

// Unwrap returns the inner *NotFoundError so errors.As and errors.Is descend
// through the NotFound -> NotSupported chain.
func (e *NotSupportedError) Unwrap() error { return e.NotFoundError }

// RateLimitError is returned when the per-minute rate limit or the concurrent
// burst limiter rejects a request (HTTP 429). RetryAfter points at the next
// window.
//
// For the 429 raised when the monthly quota is spent, see QuotaExceededError,
// which unwraps to this type so errors.As with a *RateLimitError target matches
// both cases.
type RateLimitError struct{ *APIError }

// QuotaExceededError is returned when the month's request quota is spent
// (HTTP 429 quota_exceeded). Waiting does not help before the quota resets;
// only an upgrade does.
//
// Unwraps to *RateLimitError. Use errors.As with a *QuotaExceededError target
// to tell a spent quota from a per-minute limit worth retrying.
type QuotaExceededError struct{ *RateLimitError }

// Unwrap returns the inner *RateLimitError so errors.As and errors.Is descend
// through the RateLimit -> QuotaExceeded chain.
func (e *QuotaExceededError) Unwrap() error { return e.RateLimitError }

// TemporarilyUnavailableError is returned when the domain data is temporarily unavailable (HTTP 503).
type TemporarilyUnavailableError struct{ *APIError }

// UpstreamError is returned when the upstream RDAP server fails (HTTP 502).
type UpstreamError struct{ *APIError }

// TimeoutError is returned when the request did not complete in time
// (HTTP 504). Safe to retry after a short delay.
type TimeoutError struct{ *APIError }

// Every typed error embeds *APIError, but embedding is not unwrapping: each
// needs its own Unwrap for a catch-all errors.As(err, &apiErr) to match it.
// The narrow variants above unwrap to their broader sibling instead, and reach
// *APIError through it.
func (e *ValidationError) Unwrap() error             { return e.APIError }
func (e *AuthenticationError) Unwrap() error         { return e.APIError }
func (e *SubscriptionRequiredError) Unwrap() error   { return e.APIError }
func (e *RequestFailedError) Unwrap() error          { return e.APIError }
func (e *NotFoundError) Unwrap() error               { return e.APIError }
func (e *RateLimitError) Unwrap() error              { return e.APIError }
func (e *UpstreamError) Unwrap() error               { return e.APIError }
func (e *TemporarilyUnavailableError) Unwrap() error { return e.APIError }
func (e *TimeoutError) Unwrap() error                { return e.APIError }

// newError creates a typed error based on the HTTP status code, and on the
// error code where one status carries more than one meaning.
func newError(statusCode int, code, message string, retryAfter int, fieldErrors map[string][]string) error {
	base := &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		RetryAfter: retryAfter,
		Errors:     fieldErrors,
	}

	switch statusCode {
	case 400:
		return &ValidationError{base}
	case 401:
		return &AuthenticationError{base}
	case 403:
		sr := &SubscriptionRequiredError{base}
		if code == "plan_upgrade_required" {
			return &PlanUpgradeRequiredError{sr}
		}
		return sr
	case 404:
		nf := &NotFoundError{base}
		if code == "not_supported" {
			return &NotSupportedError{nf}
		}
		return nf
	case 422:
		return &RequestFailedError{base}
	case 429:
		rl := &RateLimitError{base}
		if code == "quota_exceeded" {
			return &QuotaExceededError{rl}
		}
		return rl
	case 502:
		return &UpstreamError{base}
	case 503:
		return &TemporarilyUnavailableError{base}
	case 504:
		return &TimeoutError{base}
	default:
		return base
	}
}
