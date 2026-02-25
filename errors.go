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
type NotFoundError struct{ *APIError }

// RateLimitError is returned when rate limit or monthly quota is exceeded (HTTP 429).
type RateLimitError struct{ *APIError }

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
		return &NotFoundError{base}
	case 429:
		return &RateLimitError{base}
	case 502:
		return &UpstreamError{base}
	default:
		return base
	}
}
