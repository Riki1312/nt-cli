package cli

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Riki1312/nt-cli/internal/auth"
	"github.com/Riki1312/nt-cli/internal/mcp"
)

type toolCall struct {
	token string
	name  string
	args  map[string]any
}

func TestDBQueryCommandShapesMCPRequest(t *testing.T) {
	t.Parallel()

	var calls []toolCall
	var printed []any
	a := testApp(t, &calls, &printed)
	a.callTool = func(ctx context.Context, token, tool string, args map[string]any) (*mcp.ToolResult, error) {
		calls = append(calls, toolCall{token: token, name: tool, args: args})
		return textResult(`{"results":[{"Name":"Write docs"}],"has_more":false}`), nil
	}

	err := executeTestCommand(a, "db", "ds123", "query", "SELECT * FROM _ WHERE Status = ?", "--params", "Done")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(calls))
	}

	call := calls[0]
	if call.token != "test-token" {
		t.Fatalf("token = %q, want test-token", call.token)
	}
	if call.name != "notion-query-data-sources" {
		t.Fatalf("tool = %q, want notion-query-data-sources", call.name)
	}

	data := call.args["data"].(map[string]any)
	if got, want := data["query"], `SELECT * FROM "collection://ds123" WHERE Status = ?`; got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}
	if got, want := data["data_source_urls"], []string{"collection://ds123"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("data_source_urls = %#v, want %#v", got, want)
	}
	if got, want := data["params"], []string{"Done"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("params = %#v, want %#v", got, want)
	}

	assertPrintedJSON(t, printed, `[{"Name":"Write docs"}]`)
}

func TestPageReplaceCommandReadsAndWritesMergedContent(t *testing.T) {
	t.Parallel()

	var calls []toolCall
	var printed []any
	a := testApp(t, &calls, &printed)
	a.callTool = func(ctx context.Context, token, tool string, args map[string]any) (*mcp.ToolResult, error) {
		calls = append(calls, toolCall{token: token, name: tool, args: args})
		switch tool {
		case "notion-fetch":
			return textResult(`{"metadata":{"type":"page"},"title":"Doc","text":"<content>alpha beta alpha</content>"}`), nil
		case "notion-update-page":
			return textResult(`{}`), nil
		default:
			t.Fatalf("unexpected tool call %q", tool)
			return nil, nil
		}
	}

	err := executeTestCommand(a, "page", "page123", "replace", "--first", "alpha", "gamma")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("got %d tool calls, want 2", len(calls))
	}
	if calls[0].name != "notion-fetch" {
		t.Fatalf("first tool = %q, want notion-fetch", calls[0].name)
	}
	if calls[1].name != "notion-update-page" {
		t.Fatalf("second tool = %q, want notion-update-page", calls[1].name)
	}
	if got, want := calls[1].args["new_str"], "gamma beta alpha"; got != want {
		t.Fatalf("new_str = %q, want %q", got, want)
	}

	assertPrintedJSON(t, printed, `{"id":"page123","ok":true}`)
}

func TestResourceCommandsRejectExtraArguments(t *testing.T) {
	t.Parallel()

	var calls []toolCall
	var printed []any
	err := executeTestCommand(testApp(t, &calls, &printed), "page", "page123", "read", "extra")
	if err == nil {
		t.Fatalf("Execute returned nil error")
	}
	if len(calls) != 0 {
		t.Fatalf("got %d tool calls, want 0", len(calls))
	}
	if len(printed) != 0 {
		t.Fatalf("got %d printed values, want 0", len(printed))
	}
}

func testApp(t *testing.T, calls *[]toolCall, printed *[]any) app {
	t.Helper()

	return app{
		ensureToken: func(context.Context) (*auth.Token, error) {
			return &auth.Token{AccessToken: "test-token"}, nil
		},
		callTool: func(ctx context.Context, token, tool string, args map[string]any) (*mcp.ToolResult, error) {
			*calls = append(*calls, toolCall{token: token, name: tool, args: args})
			return textResult(`{}`), nil
		},
		print: func(v any) error {
			*printed = append(*printed, v)
			return nil
		},
		hint: func(string) {},
	}
}

func executeTestCommand(a app, args ...string) error {
	cmd := newRootCmd("test", "none", a)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func textResult(text string) *mcp.ToolResult {
	return &mcp.ToolResult{
		Content: []mcp.ContentItem{{Type: "text", Text: text}},
	}
}

func assertPrintedJSON(t *testing.T, printed []any, want string) {
	t.Helper()

	if len(printed) != 1 {
		t.Fatalf("got %d printed values, want 1", len(printed))
	}
	got, err := json.Marshal(printed[0])
	if err != nil {
		t.Fatalf("marshaling printed value: %v", err)
	}
	if string(got) != want {
		t.Fatalf("printed JSON = %s, want %s", got, want)
	}
}
