package output

import (
	"errors"
	"testing"
)

func TestToolErrorClassification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		msg       string
		wantCode  string
		wantExit  int
		wantRetry int
	}{
		{
			name:     "not found",
			msg:      "page not found",
			wantCode: "NOT_FOUND",
			wantExit: ExitNotFound,
		},
		{
			name:      "rate limited with retry after",
			msg:       "rate limited: retry after 17 seconds",
			wantCode:  "RATE_LIMITED",
			wantExit:  ExitRateLimited,
			wantRetry: 17,
		},
		{
			name:     "permission denied",
			msg:      "HTTP 403: forbidden",
			wantCode: "PERMISSION_DENIED",
			wantExit: ExitPermission,
		},
		{
			name:     "auth error",
			msg:      "HTTP 401: invalid_token",
			wantCode: "AUTH_ERROR",
			wantExit: ExitAuth,
		},
		{
			name:     "selection validation is not a missing resource",
			msg:      `{"status":400,"code":"validation_error","body":"{\"message\":\"String not found in page containing permission and rate limit examples\"}"}`,
			wantCode: "INVALID_ARGUMENT",
			wantExit: ExitError,
		},
		{
			name:     "explicit forbidden status takes priority over quoted content",
			msg:      `{"status":403,"code":"restricted_resource","message":"not found in a quoted document"}`,
			wantCode: "PERMISSION_DENIED",
			wantExit: ExitPermission,
		},
		{
			name:     "unknown structured error does not classify document text",
			msg:      `{"status":500,"code":"internal_server_error","message":"document contains not found"}`,
			wantCode: "TOOL_ERROR",
			wantExit: ExitError,
		},
		{
			name:     "structured code without status",
			msg:      `{"code":"object_not_found","message":"Resource unavailable"}`,
			wantCode: "NOT_FOUND",
			wantExit: ExitNotFound,
		},
		{
			name:     "fallback",
			msg:      "unexpected tool failure",
			wantCode: "TOOL_ERROR",
			wantExit: ExitError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ToolError(tt.msg)
			if got.Code != tt.wantCode {
				t.Fatalf("Code = %q, want %q", got.Code, tt.wantCode)
			}
			if got.ExitCode != tt.wantExit {
				t.Fatalf("ExitCode = %d, want %d", got.ExitCode, tt.wantExit)
			}
			if got.RetryAfter != tt.wantRetry {
				t.Fatalf("RetryAfter = %d, want %d", got.RetryAfter, tt.wantRetry)
			}
		})
	}
}

func TestErrorFromPreservesCLIError(t *testing.T) {
	t.Parallel()

	want := NewError(ExitPermission, "PERMISSION_DENIED", "no access")
	got := ErrorFrom(want)
	if got != want {
		t.Fatalf("ErrorFrom did not preserve CLIError")
	}
}

func TestErrorFromClassifiesGenericError(t *testing.T) {
	t.Parallel()

	got := ErrorFrom(errors.New("HTTP 404: missing"))
	if got.Code != "NOT_FOUND" {
		t.Fatalf("Code = %q, want NOT_FOUND", got.Code)
	}
	if got.ExitCode != ExitNotFound {
		t.Fatalf("ExitCode = %d, want %d", got.ExitCode, ExitNotFound)
	}
}

func TestToolErrorUsesProviderMessageWithoutTransportEnvelope(t *testing.T) {
	got := ToolError(`{"status":400,"code":"validation_error","body":"{\"message\":\"String not found\"}"}`)
	if got.Message != "String not found" || got.Code != "INVALID_ARGUMENT" {
		t.Fatalf("provider message lost or misclassified: %+v", got)
	}
}
