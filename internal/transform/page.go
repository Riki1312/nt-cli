package transform

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Riki1312/nt-cli/internal/mcp"
)

type Page struct {
	ID         string            `json:"id"`
	Title      string            `json:"title,omitempty"`
	URL        string            `json:"url,omitempty"`
	Properties map[string]any    `json:"properties,omitempty"`
	Content    string            `json:"content"`
	Hint       string            `json:"hint,omitempty"`
}

// fetchResponse matches the JSON structure returned by notion-fetch.
type fetchResponse struct {
	Metadata struct {
		Type string `json:"type"`
	} `json:"metadata"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Text  string `json:"text"`
}

// PageRead converts a notion-fetch MCP tool result into a compact page JSON object.
func PageRead(result *mcp.ToolResult, pageID string) (any, error) {
	text := result.TextContent()
	if text == "" {
		return nil, fmt.Errorf("empty page response")
	}

	var resp fetchResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		// Fallback: return raw text if not JSON
		return &Page{ID: pageID, Content: text}, nil
	}

	page := &Page{
		ID:    pageID,
		Title: resp.Title,
		URL:   resp.URL,
	}

	if resp.Metadata.Type == "database" {
		page.Hint = "this is a database; use: nt db " + pageID + " read"
	}

	// Extract properties from <properties> tags in the text
	page.Properties = extractProperties(resp.Text)

	// Extract content from <content> tags in the text
	page.Content = extractContent(resp.Text)

	return page, nil
}

// extractProperties parses the JSON inside <properties>...</properties> tags.
func extractProperties(text string) map[string]any {
	start := strings.Index(text, "<properties>")
	end := strings.Index(text, "</properties>")
	if start < 0 || end < 0 || end <= start {
		return nil
	}
	propsJSON := strings.TrimSpace(text[start+len("<properties>") : end])
	if propsJSON == "" {
		return nil
	}

	var props map[string]any
	if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
		return nil
	}
	return props
}

// ExtractPageContent extracts the page content from a notion-fetch response text.
// It parses the JSON envelope and extracts the <content> section from the text field.
func ExtractPageContent(responseText string) string {
	var resp fetchResponse
	if err := json.Unmarshal([]byte(responseText), &resp); err != nil {
		return ""
	}
	return extractContent(resp.Text)
}

// extractContent extracts the text inside <content>...</content> tags
// and cleans up Notion-specific XML artifacts.
func extractContent(text string) string {
	start := strings.Index(text, "<content>")
	end := strings.Index(text, "</content>")
	if start < 0 || end < 0 || end <= start {
		return ""
	}
	content := text[start+len("<content>") : end]
	content = strings.TrimSpace(content)

	// Clean up Notion-flavored artifacts
	content = strings.ReplaceAll(content, "<empty-block/>", "")

	// Collapse excessive blank lines (3+ newlines -> 2)
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(content)
}
