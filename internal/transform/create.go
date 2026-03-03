package transform

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Riki1312/nt-cli/internal/mcp"
)

type createdPage struct {
	ID  string `json:"id"`
	URL string `json:"url,omitempty"`
	OK  bool   `json:"ok"`
}

// createResponse matches the JSON returned by notion-create-pages.
type createResponse struct {
	Pages []struct {
		ID         string         `json:"id"`
		URL        string         `json:"url"`
		Properties map[string]any `json:"properties,omitempty"`
	} `json:"pages"`
}

// CreatedPages extracts page IDs and URLs from a notion-create-pages result.
// Always returns a slice of createdPage for a consistent JSON shape.
func CreatedPages(result *mcp.ToolResult) (any, error) {
	text := result.TextContent()
	if text == "" {
		return nil, fmt.Errorf("empty create response")
	}

	var resp createResponse
	if err := json.Unmarshal([]byte(text), &resp); err == nil && len(resp.Pages) > 0 {
		pages := make([]createdPage, 0, len(resp.Pages))
		for _, p := range resp.Pages {
			pages = append(pages, createdPage{ID: p.ID, URL: p.URL, OK: true})
		}
		return pages, nil
	}

	// Fallback: try to find <page> tags in text
	return parsePageTags(text)
}

// duplicateResponse matches the JSON returned by notion-duplicate-page.
type duplicateResponse struct {
	PageID  string `json:"page_id"`
	PageURL string `json:"page_url"`
}

// DuplicateResult extracts the duplicate page info from the response.
func DuplicateResult(result *mcp.ToolResult, originalID string) any {
	text := result.TextContent()
	if text == "" {
		return map[string]any{"id": originalID, "ok": true}
	}

	var resp duplicateResponse
	if err := json.Unmarshal([]byte(text), &resp); err == nil && resp.PageID != "" {
		return createdPage{ID: resp.PageID, URL: resp.PageURL, OK: true}
	}

	return map[string]any{"id": originalID, "ok": true}
}

func parsePageTags(text string) (any, error) {
	type tagPage struct {
		ID    string `json:"id"`
		Title string `json:"title,omitempty"`
		URL   string `json:"url,omitempty"`
		OK    bool   `json:"ok"`
	}

	var pages []tagPage
	remaining := text
	for {
		start := strings.Index(remaining, "<page url=\"")
		if start < 0 {
			break
		}
		remaining = remaining[start+len("<page url=\""):]
		urlEnd := strings.Index(remaining, "\"")
		if urlEnd < 0 {
			break
		}
		pageURL := remaining[:urlEnd]
		remaining = remaining[urlEnd:]

		titleStart := strings.Index(remaining, ">")
		titleEnd := strings.Index(remaining, "</page>")
		var title string
		if titleStart >= 0 && titleEnd > titleStart {
			title = remaining[titleStart+1 : titleEnd]
		}

		id := extractIDFromURL(pageURL)
		pages = append(pages, tagPage{ID: id, Title: title, URL: pageURL, OK: true})

		if titleEnd >= 0 {
			remaining = remaining[titleEnd+len("</page>"):]
		}
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("could not parse create response")
	}
	return pages, nil
}

func extractIDFromURL(u string) string {
	parts := strings.Split(u, "/")
	last := parts[len(parts)-1]
	// The ID might have a slug prefix separated by - (e.g. "My-Page-abc123def456...")
	if idx := strings.LastIndex(last, "-"); idx >= 0 && len(last)-idx-1 == 32 {
		return last[idx+1:]
	}
	// Bare UUID with dashes (e.g. "abcd1234-5678-...")
	clean := strings.ReplaceAll(last, "-", "")
	if len(clean) == 32 {
		return clean
	}
	return last
}
