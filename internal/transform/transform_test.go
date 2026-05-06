package transform

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Riki1312/nt-cli/internal/mcp"
)

func TestPageReadGolden(t *testing.T) {
	t.Parallel()

	got, err := PageRead(textResult(t, "page_fetch_response.json"), "page123")
	if err != nil {
		t.Fatalf("PageRead returned error: %v", err)
	}

	assertGoldenJSON(t, got, "page_read.golden.json")
}

func TestDBReadGolden(t *testing.T) {
	t.Parallel()

	got, err := DBRead(textResult(t, "db_fetch_response.json"), "db123")
	if err != nil {
		t.Fatalf("DBRead returned error: %v", err)
	}

	assertGoldenJSON(t, got, "db_read.golden.json")
}

func TestSearchResultsGolden(t *testing.T) {
	t.Parallel()

	results, err := SearchResults(textResult(t, "search_response.json"))
	if err != nil {
		t.Fatalf("SearchResults returned error: %v", err)
	}

	got := FilterSearchResults(results, "page", 1)
	assertGoldenJSON(t, got, "search_results.golden.json")
}

func TestQueryResultsGolden(t *testing.T) {
	t.Parallel()

	got, err := QueryResults(textResult(t, "query_response.json"))
	if err != nil {
		t.Fatalf("QueryResults returned error: %v", err)
	}

	assertGoldenJSON(t, got, "query_results.golden.json")
}

func TestCreatedPagesGolden(t *testing.T) {
	t.Parallel()

	got, err := CreatedPages(textResult(t, "create_response.json"))
	if err != nil {
		t.Fatalf("CreatedPages returned error: %v", err)
	}

	assertGoldenJSON(t, got, "created_pages.golden.json")
}

func textResult(t *testing.T, name string) *mcp.ToolResult {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return &mcp.ToolResult{
		Content: []mcp.ContentItem{{Type: "text", Text: string(data)}},
	}
}

func assertGoldenJSON(t *testing.T, got any, name string) {
	t.Helper()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(got); err != nil {
		t.Fatalf("marshaling result: %v", err)
	}
	gotJSON := buf.String()

	wantData, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading golden %s: %v", name, err)
	}
	wantJSON := strings.TrimSpace(string(wantData)) + "\n"

	if gotJSON != wantJSON {
		t.Fatalf("golden mismatch for %s\n got:\n%s\nwant:\n%s", name, gotJSON, wantJSON)
	}
}
