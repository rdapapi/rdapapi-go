package rdapapi

// APIError is the base error type for all RDAP API errors.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RetryAfter int
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
type SubscriptionRequiredError struct{ *APIError }

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

// RateLimitError is returned when rate limit or monthly quota is exceeded (HTTP 429).
type RateLimitError struct{ *APIError }

// TemporarilyUnavailableError is returned when the domain data is temporarily unavailable (HTTP 503).
type TemporarilyUnavailableError struct{ *APIError }

// UpstreamError is returned when the upstream RDAP server fails (HTTP 502).
type UpstreamError struct{ *APIError }

// newError creates a typed error based on the HTTP status code.
func newError(statusCode int, code, message string, retryAfter int) error {
	base := &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		RetryAfter: retryAfter,
	}

	switch statusCode {
	case 400:
		return &ValidationError{base}
	case 401:
		return &AuthenticationError{base}
	case 403:
		return &SubscriptionRequiredError{base}
	case 404:
		nf := &NotFoundError{base}
		if code == "not_supported" {
			return &NotSupportedError{nf}
		}
		return nf
	case 429:
		return &RateLimitError{base}
	case 502:
		return &UpstreamError{base}
	case 503:
		return &TemporarilyUnavailableError{base}
	default:
		return base
	}
}
