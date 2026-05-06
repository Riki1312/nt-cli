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
	ExitOK          = 0
	ExitError       = 1
	ExitAuth        = 2
	ExitNotFound    = 3
	ExitRateLimited = 4
	ExitPermission  = 5
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

func NewError(exitCode int, code string, msg string) *CLIError {
	return &CLIError{Message: msg, Code: code, ExitCode: exitCode}
}

func AuthError(msg string) *CLIError {
	return NewError(ExitAuth, "AUTH_ERROR", msg)
}

// ToolError maps MCP tool failure text to stable CLI error codes.
func ToolError(msg string) *CLIError {
	return classifyMessage(msg, ExitError, "TOOL_ERROR")
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
