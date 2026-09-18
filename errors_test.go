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
		{503, "temporarily_unavailable", "temporarily unavailable", 300, "TemporarilyUnavailableError"},
	}

	for _, tt := range tests {
		t.Run(tt.wantType, func(t *testing.T) {
			err := newError(tt.status, tt.code, tt.message, tt.retryAfter, nil)

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
			case 503:
				var target *TemporarilyUnavailableError
				if !errors.As(err, &target) {
					t.Fatal("errors.As failed for TemporarilyUnavailableError")
				}
				checkBase(t, target.APIError, tt)
				if target.RetryAfter != 300 {
					t.Errorf("RetryAfter = %d, want 300", target.RetryAfter)
				}
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
	err := newError(500, "server_error", "internal", 0, nil)

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
	err := newError(404, "not_found", "not found", 0, nil)

	var v *ValidationError
	if errors.As(err, &v) {
		t.Error("NotFoundError should not match ValidationError")
	}

	var r *RateLimitError
	if errors.As(err, &r) {
		t.Error("NotFoundError should not match RateLimitError")
	}
}

func TestNotSupportedErrorFor404WithNotSupportedCode(t *testing.T) {
	err := newError(404, "not_supported", "TLD .nope is not supported", 0, nil)

	var ns *NotSupportedError
	if !errors.As(err, &ns) {
		t.Fatal("errors.As failed for NotSupportedError")
	}
	if ns.Code != "not_supported" {
		t.Errorf("Code = %q, want %q", ns.Code, "not_supported")
	}
	if ns.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", ns.StatusCode)
	}

	// Backwards compatible: errors.As with *NotFoundError target also matches
	// because NotSupportedError unwraps to *NotFoundError.
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatal("errors.As should match *NotFoundError via Unwrap")
	}
}

func TestNotFoundErrorForPlainNotFoundCode(t *testing.T) {
	err := newError(404, "not_found", "Not found", 0, nil)

	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatal("errors.As failed for NotFoundError")
	}

	// Should NOT be a NotSupportedError.
	var ns *NotSupportedError
	if errors.As(err, &ns) {
		t.Error("plain not_found should not match NotSupportedError")
	}
}

func TestNotSupportedErrorUnwrapChain(t *testing.T) {
	err := newError(404, "not_supported", "not supported", 0, nil).(*NotSupportedError)

	// Unwrap returns the inner NotFoundError.
	inner := err.Unwrap()
	if _, ok := inner.(*NotFoundError); !ok {
		t.Fatalf("Unwrap returned %T, want *NotFoundError", inner)
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

func TestPlanUpgradeRequiredErrorFor403WithUpgradeCode(t *testing.T) {
	err := newError(403, "plan_upgrade_required", "Bulk lookups require a Pro or Business plan", 0, nil)

	var pu *PlanUpgradeRequiredError
	if !errors.As(err, &pu) {
		t.Fatal("errors.As failed for PlanUpgradeRequiredError")
	}
	if pu.Code != "plan_upgrade_required" {
		t.Errorf("Code = %q, want %q", pu.Code, "plan_upgrade_required")
	}

	// Backwards compatible: a *SubscriptionRequiredError target still matches.
	var sr *SubscriptionRequiredError
	if !errors.As(err, &sr) {
		t.Fatal("errors.As should match *SubscriptionRequiredError via Unwrap")
	}

	inner := pu.Unwrap()
	if _, ok := inner.(*SubscriptionRequiredError); !ok {
		t.Fatalf("Unwrap returned %T, want *SubscriptionRequiredError", inner)
	}
}

func TestSubscriptionRequiredErrorForPlainCode(t *testing.T) {
	err := newError(403, "subscription_required", "An active subscription is required", 0, nil)

	var sr *SubscriptionRequiredError
	if !errors.As(err, &sr) {
		t.Fatal("errors.As failed for SubscriptionRequiredError")
	}

	var pu *PlanUpgradeRequiredError
	if errors.As(err, &pu) {
		t.Error("plain subscription_required should not match PlanUpgradeRequiredError")
	}
}

func TestQuotaExceededErrorFor429WithQuotaCode(t *testing.T) {
	err := newError(429, "quota_exceeded", "Monthly request limit reached", 0, nil)

	var qe *QuotaExceededError
	if !errors.As(err, &qe) {
		t.Fatal("errors.As failed for QuotaExceededError")
	}
	if qe.StatusCode != 429 {
		t.Errorf("StatusCode = %d, want 429", qe.StatusCode)
	}

	// Backwards compatible: a *RateLimitError target still matches.
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatal("errors.As should match *RateLimitError via Unwrap")
	}

	inner := qe.Unwrap()
	if _, ok := inner.(*RateLimitError); !ok {
		t.Fatalf("Unwrap returned %T, want *RateLimitError", inner)
	}
}

func TestRateLimitErrorForPlainCode(t *testing.T) {
	err := newError(429, "rate_limit_exceeded", "Rate limit exceeded", 30, nil)

	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatal("errors.As failed for RateLimitError")
	}

	var qe *QuotaExceededError
	if errors.As(err, &qe) {
		t.Error("rate_limit_exceeded should not match QuotaExceededError")
	}
}

func TestRequestFailedErrorCarriesFieldErrors(t *testing.T) {
	fields := map[string][]string{"domains": {"The domains field is required."}}
	err := newError(422, "request_failed", "The given data was invalid.", 0, fields)

	var rf *RequestFailedError
	if !errors.As(err, &rf) {
		t.Fatal("errors.As failed for RequestFailedError")
	}
	if got := rf.Errors["domains"]; len(got) != 1 || got[0] != "The domains field is required." {
		t.Errorf("Errors[domains] = %v, want one message", got)
	}
}

func TestTimeoutErrorFor504(t *testing.T) {
	err := newError(504, "gateway_timeout", "The request did not complete in time.", 0, nil)

	var te *TimeoutError
	if !errors.As(err, &te) {
		t.Fatal("errors.As failed for TimeoutError")
	}
	if te.StatusCode != 504 {
		t.Errorf("StatusCode = %d, want 504", te.StatusCode)
	}
}

// Every error newError can return must reach *APIError through Unwrap, so a
// catch-all errors.As keeps matching. Add a row whenever newError gains a
// branch: embedding *APIError is not enough on its own.
func TestEveryErrorUnwrapsToAPIError(t *testing.T) {
	tests := []struct {
		status   int
		code     string
		wantType string
	}{
		{400, "invalid_domain", "*rdapapi.ValidationError"},
		{401, "unauthenticated", "*rdapapi.AuthenticationError"},
		{403, "subscription_required", "*rdapapi.SubscriptionRequiredError"},
		{403, "plan_upgrade_required", "*rdapapi.PlanUpgradeRequiredError"},
		{404, "not_found", "*rdapapi.NotFoundError"},
		{404, "not_supported", "*rdapapi.NotSupportedError"},
		{422, "request_failed", "*rdapapi.RequestFailedError"},
		{429, "rate_limit_exceeded", "*rdapapi.RateLimitError"},
		{429, "quota_exceeded", "*rdapapi.QuotaExceededError"},
		{502, "lookup_failed", "*rdapapi.UpstreamError"},
		{503, "temporarily_unavailable", "*rdapapi.TemporarilyUnavailableError"},
		{504, "gateway_timeout", "*rdapapi.TimeoutError"},
		{500, "server_error", "*rdapapi.APIError"},
	}

	for _, tt := range tests {
		t.Run(tt.wantType, func(t *testing.T) {
			err := newError(tt.status, tt.code, "message", 0, nil)
			if got := fmt.Sprintf("%T", err); got != tt.wantType {
				t.Fatalf("newError(%d, %q) = %s, want %s", tt.status, tt.code, got, tt.wantType)
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("%s does not unwrap to *APIError", tt.wantType)
			}
			if apiErr.StatusCode != tt.status || apiErr.Code != tt.code {
				t.Errorf("unwrapped to %d/%q, want %d/%q", apiErr.StatusCode, apiErr.Code, tt.status, tt.code)
			}
		})
	}
}
