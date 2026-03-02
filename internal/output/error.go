package output

import (
	"encoding/json"
	"errors"
	"os"
)

const (
	ExitOK             = 0
	ExitError          = 1
	ExitAuth           = 2
	ExitNotFound       = 3
	ExitRateLimited    = 4
	ExitPermission     = 5
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

// HandleError writes a JSON error to stderr and exits with the appropriate code.
func HandleError(err error) {
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		writeStderr(cliErr)
		os.Exit(cliErr.ExitCode)
	}
	writeStderr(&CLIError{Message: err.Error(), Code: "ERROR"})
	os.Exit(ExitError)
}

func writeStderr(e *CLIError) {
	json.NewEncoder(os.Stderr).Encode(e)
}
