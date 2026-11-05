package output

import (
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	// ExitOK is returned on success.
	ExitOK = 0
	// ExitError is returned for unclassified failures.
	ExitError = 1
	// ExitAuth is returned when authentication fails.
	ExitAuth = 2
	// ExitNotFound is returned when a requested Notion resource is missing.
	ExitNotFound = 3
	// ExitRateLimited is returned when Notion asks the client to slow down.
	ExitRateLimited = 4
	// ExitPermission is returned when the workspace denies access.
	ExitPermission = 5
)

// CLIError is a structured error that maps to a JSON output and exit code.
type CLIError struct {
	Message    string `json:"error"`
	Code       string `json:"code"`
	ExitCode   int    `json:"-"`
	RetryAfter int    `json:"retry_after,omitempty"`
}

func (e *CLIError) Error() string {
	return e.Message
}

// NewError builds a CLIError with an explicit exit code and machine code.
func NewError(exitCode int, code string, msg string) *CLIError {
	return &CLIError{Message: msg, Code: code, ExitCode: exitCode}
}

// AuthError builds an authentication failure.
func AuthError(msg string) *CLIError {
	return NewError(ExitAuth, "AUTH_ERROR", msg)
}

// ToolError maps MCP tool failure text to stable CLI error codes.
func ToolError(msg string) *CLIError {
	// Hosted API errors can include quoted page content. Prefer explicit status
	// and code over words such as "not found" in a selection error or document.
	var response struct {
		Code    string `json:"code"`
		Status  int    `json:"status"`
		Message string `json:"message"`
		Body    string `json:"body"`
	}
	if json.Unmarshal([]byte(msg), &response) != nil || response.Code == "" && response.Status == 0 {
		return classifyMessage(msg, ExitError, "TOOL_ERROR")
	}

	if response.Message == "" && response.Body != "" {
		var body struct {
			Message string `json:"message"`
		}
		if json.Unmarshal([]byte(response.Body), &body) == nil {
			response.Message = body.Message
		}
	}
	message := response.Message
	if message == "" {
		message = msg
	}

	status := response.Status
	if status == 0 {
		switch response.Code {
		case "validation_error", "invalid_input":
			status = 400
		case "unauthorized":
			status = 401
		case "restricted_resource", "permission_denied":
			status = 403
		case "object_not_found":
			status = 404
		case "rate_limited":
			status = 429
		}
	}
	switch status {
	case 400:
		return NewError(ExitError, "INVALID_ARGUMENT", message)
	case 401:
		return AuthError(message)
	case 403:
		return NewError(ExitPermission, "PERMISSION_DENIED", message)
	case 404:
		return NewError(ExitNotFound, "NOT_FOUND", message)
	case 429:
		err := NewError(ExitRateLimited, "RATE_LIMITED", message)
		err.RetryAfter = parseRetryAfter(msg)
		return err
	default:
		return NewError(ExitError, "TOOL_ERROR", message)
	}
}

// ErrorFrom maps an arbitrary error to a structured CLI error.
func ErrorFrom(err error) *CLIError {
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return cliErr
	}
	return classifyMessage(err.Error(), ExitError, "ERROR")
}

// HandleError writes a JSON error to stderr and exits with the appropriate code.
func HandleError(err error) {
	cliErr := ErrorFrom(err)
	writeStderr(cliErr)
	os.Exit(cliErr.ExitCode)
}

var retryAfterPattern = regexp.MustCompile(`(?i)retry[-_ ]?after[^0-9]*(\d+)`)

func classifyMessage(msg string, fallbackExit int, fallbackCode string) *CLIError {
	lower := strings.ToLower(msg)

	switch {
	case containsAny(lower, "rate limit", "rate_limited", "too many requests", "http 429", "(429)"):
		err := NewError(ExitRateLimited, "RATE_LIMITED", msg)
		err.RetryAfter = parseRetryAfter(msg)
		return err
	case containsAny(lower, "not found", "not_found", "http 404", "(404)"):
		return NewError(ExitNotFound, "NOT_FOUND", msg)
	case containsAny(lower, "unauthorized", "unauthenticated", "invalid token", "invalid_token", "expired token", "http 401", "(401)"):
		return AuthError(msg)
	case containsAny(lower, "permission", "permission_denied", "forbidden", "http 403", "(403)"):
		return NewError(ExitPermission, "PERMISSION_DENIED", msg)
	default:
		return NewError(fallbackExit, fallbackCode, msg)
	}
}

func containsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func parseRetryAfter(msg string) int {
	matches := retryAfterPattern.FindStringSubmatch(msg)
	if len(matches) != 2 {
		return 0
	}
	seconds, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return seconds
}

func writeStderr(e *CLIError) {
	json.NewEncoder(os.Stderr).Encode(e)
}
