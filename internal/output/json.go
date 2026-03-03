package output

import (
	"encoding/json"
	"os"
)

// Print marshals v as JSON and writes it to stdout with a trailing newline.
func Print(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// Hint writes a JSON hint message to stderr. Hints are informational and do not
// affect the exit code or stdout output.
func Hint(msg string) {
	json.NewEncoder(os.Stderr).Encode(map[string]string{"hint": msg})
}
