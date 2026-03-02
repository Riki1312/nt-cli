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
