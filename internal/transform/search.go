package transform

import (
	"encoding/json"
	"fmt"

	"github.com/Riki1312/nt-cli/internal/mcp"
)

type SearchResult struct {
	ID        string `json:"id"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title"`
	URL       string `json:"url,omitempty"`
	Highlight string `json:"highlight,omitempty"`
}

type searchResponse struct {
	Results []SearchResult `json:"results"`
}

// SearchResults converts a notion-search MCP tool result into a compact JSON array.
func SearchResults(result *mcp.ToolResult) ([]SearchResult, error) {
	text := result.TextContent()
	if text == "" {
		return []SearchResult{}, nil
	}

	var resp searchResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}

	if resp.Results == nil {
		return []SearchResult{}, nil
	}
	return resp.Results, nil
}

// FilterSearchResults filters results by type and applies a limit.
// An empty typeFilter matches all types. A limit of 0 means no limit.
func FilterSearchResults(results []SearchResult, typeFilter string, limit int) []SearchResult {
	filtered := results
	if typeFilter != "" {
		filtered = make([]SearchResult, 0, len(results))
		for _, r := range results {
			if r.Type == typeFilter {
				filtered = append(filtered, r)
			}
		}
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}
