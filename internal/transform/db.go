package transform

import (
	"encoding/json"
	"fmt"

	"github.com/Riki1312/nt-cli/internal/mcp"
)

// Database represents a parsed database from notion-fetch.
type Database struct {
	ID     string `json:"id"`
	Title  string `json:"title,omitempty"`
	URL    string `json:"url,omitempty"`
	Schema string `json:"schema,omitempty"`
}

// DBRead converts a notion-fetch result for a database into a compact JSON object.
func DBRead(result *mcp.ToolResult, dbID string) (any, error) {
	text := result.TextContent()
	if text == "" {
		return nil, fmt.Errorf("empty database response")
	}

	var resp fetchResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return &Database{ID: dbID, Schema: text}, nil
	}

	db := &Database{
		ID:    dbID,
		Title: resp.Title,
		URL:   resp.URL,
	}

	// The text field contains the full schema with data-source, views, etc.
	db.Schema = resp.Text

	return db, nil
}
