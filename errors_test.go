package rdapapi

import (
	"errors"
	"fmt"
	"testing"
)

func TestAPIErrorImplementsError(t *testing.T) {
	err := &APIError{
		StatusCode: 400,
		Code:       "invalid_input",
		Message:    "bad request",
		RetryAfter: 0,
	}

	var _ error = err // compile-time check

	if got := err.Error(); got != "bad request" {
		t.Errorf("Error() = %q, want %q", got, "bad request")
	}
}

func TestAPIErrorUnwrapReturnsNil(t *testing.T) {
	err := &APIError{Message: "test"}
	if got := err.Unwrap(); got != nil {
		t.Errorf("Unwrap() = %v, want nil", got)
	}
}

func TestNewErrorTypedErrors(t *testing.T) {
	tests := []struct {
		status     int
		code       string
		message    string
		retryAfter int
		wantType   string
	}{
		{400, "invalid_input", "bad domain", 0, "ValidationError"},
		{401, "unauthenticated", "invalid key", 0, "AuthenticationError"},
		{403, "subscription_required", "no plan", 0, "SubscriptionRequiredError"},
		{404, "not_found", "not found", 0, "NotFoundError"},
		{429, "rate_limited", "too many", 30, "RateLimitError"},
		{502, "upstream_error", "upstream fail", 0, "UpstreamError"},
	}

	for _, tt := range tests {
		t.Run(tt.wantType, func(t *testing.T) {
			err := newError(tt.status, tt.code, tt.message, tt.retryAfter)

			// Check Error() returns the message.
			if got := err.Error(); got != tt.message {
				t.Errorf("Error() = %q, want %q", got, tt.message)
			}

			// Check errors.As works for each typed error.
			switch tt.status {
			case 400:
				var target *ValidationError
				if !errors.As(err, &target) {
					t.Fatal("errors.As failed for ValidationError")
				}
				checkBase(t, target.APIError, tt)
			case 401:
				var target *AuthenticationError
				if !errors.As(err, &target) {
					t.Fatal("errors.As failed for AuthenticationError")
				}
				checkBase(t, target.APIError, tt)
			case 403:
				var target *SubscriptionRequiredError
				if !errors.As(err, &target) {
					t.Fatal("errors.As failed for SubscriptionRequiredError")
				}
				checkBase(t, target.APIError, tt)
			case 404:
				var target *NotFoundError
				if !errors.As(err, &target) {
					t.Fatal("errors.As failed for NotFoundError")
				}
				checkBase(t, target.APIError, tt)
			case 429:
				var target *RateLimitError
				if !errors.As(err, &target) {
					t.Fatal("errors.As failed for RateLimitError")
				}
				checkBase(t, target.APIError, tt)
				if target.RetryAfter != 30 {
					t.Errorf("RetryAfter = %d, want 30", target.RetryAfter)
				}
			case 502:
				var target *UpstreamError
				if !errors.As(err, &target) {
					t.Fatal("errors.As failed for UpstreamError")
				}
				checkBase(t, target.APIError, tt)
			}

			// Typed errors embed *APIError, accessible via the field.
			// errors.As matches the concrete typed error, not the embedded base.
		})
	}
}

func checkBase(t *testing.T, base *APIError, tt struct {
	status     int
	code       string
	message    string
	retryAfter int
	wantType   string
},
) {
	t.Helper()
	if base.StatusCode != tt.status {
		t.Errorf("StatusCode = %d, want %d", base.StatusCode, tt.status)
	}
	if base.Code != tt.code {
		t.Errorf("Code = %q, want %q", base.Code, tt.code)
	}
	if base.Message != tt.message {
		t.Errorf("Message = %q, want %q", base.Message, tt.message)
	}
}

func TestNewErrorUnknownStatus(t *testing.T) {
	err := newError(500, "server_error", "internal", 0)

	// Should be a base APIError, not a typed one.
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected APIError for unknown status")
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
	if apiErr.Message != "internal" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "internal")
	}

	// Should not match any typed error.
	var notFound *NotFoundError
	if errors.As(err, &notFound) {
		t.Error("unexpected NotFoundError for 500")
	}
}

func TestTypedErrorsDoNotMatchEachOther(t *testing.T) {
	err := newError(404, "not_found", "not found", 0)

	var v *ValidationError
	if errors.As(err, &v) {
		t.Error("NotFoundError should not match ValidationError")
	}

	var r *RateLimitError
	if errors.As(err, &r) {
		t.Error("NotFoundError should not match RateLimitError")
	}
}

func TestAPIErrorFormatting(t *testing.T) {
	err := &APIError{
		StatusCode: 429,
		Code:       "rate_limited",
		Message:    "slow down",
		RetryAfter: 60,
	}

	// fmt.Sprintf should use Error() method.
	got := fmt.Sprintf("error: %s", err)
	if got != "error: slow down" {
		t.Errorf("formatted = %q, want %q", got, "error: slow down")
	}
}
