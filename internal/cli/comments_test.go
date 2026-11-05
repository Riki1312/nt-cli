package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Riki1312/nt-cli/internal/auth"
	"github.com/Riki1312/nt-cli/internal/mcp"
	"github.com/Riki1312/nt-cli/internal/output"
)

const testDiscussion = "discussion://page123/block123/discussion123"

func TestCommentCommandsMatchHostedToolContracts(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		stdin   string
		tool    string
		request map[string]any
	}{
		{
			name:    "page comment",
			args:    []string{"page", "page123", "comment", "**Review** this"},
			tool:    "notion-create-comment",
			request: map[string]any{"page_id": "page123", "markdown": "**Review** this"},
		},
		{
			name:  "selection and stdin",
			args:  []string{"page", "page123", "comment", "--select", "passage start...passage end", "-"},
			stdin: "First paragraph.\n\nSecond paragraph. 👻\n",
			tool:  "notion-create-comment",
			request: map[string]any{
				"page_id":                 "page123",
				"selection_with_ellipsis": "passage start...passage end",
				"markdown":                "First paragraph.\n\nSecond paragraph. 👻\n",
			},
		},
		{
			name:    "resolved page comments",
			args:    []string{"page", "page123", "comments", "--include-resolved"},
			tool:    "notion-get-comments",
			request: map[string]any{"page_id": "page123", "include_resolved": true},
		},
		{
			name:    "discussion reply",
			args:    []string{"discussion", testDiscussion, "reply", "Agreed"},
			tool:    "notion-create-comment",
			request: map[string]any{"page_id": "page123", "discussion_id": testDiscussion, "markdown": "Agreed"},
		},
		{
			name:    "bare ID reply",
			args:    []string{"discussion", "discussion123", "reply", "--page", "page123", "-"},
			stdin:   "A response\n",
			tool:    "notion-create-comment",
			request: map[string]any{"page_id": "page123", "discussion_id": "discussion123", "markdown": "A response\n"},
		},
		{
			name: "older discussion comments",
			args: []string{"discussion", testDiscussion, "read", "--cursor", "opaque-cursor", "--include-resolved"},
			tool: "notion-get-comments",
			request: map[string]any{
				"page_id": "page123", "discussion_id": testDiscussion,
				"cursor": "opaque-cursor", "include_resolved": true,
			},
		},
		{
			name:    "add reaction",
			args:    []string{"comment", "comment123", "react", "👀"},
			tool:    "notion-set-comment-reaction",
			request: map[string]any{"comment_id": "comment123", "emoji": "👀", "reacted": true},
		},
		{
			name:    "remove reaction",
			args:    []string{"comment", "comment123", "unreact", "👀"},
			tool:    "notion-set-comment-reaction",
			request: map[string]any{"comment_id": "comment123", "emoji": "👀", "reacted": false},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []toolCall
			var printed []any
			a := testApp(t, &calls, &printed)
			a.callTool = func(ctx context.Context, token, tool string, request map[string]any) (*mcp.ToolResult, error) {
				calls = append(calls, toolCall{token: token, name: tool, args: request})
				switch tool {
				case "notion-create-comment":
					discussion := "created-discussion"
					if target, ok := request["discussion_id"].(string); ok {
						discussion = target[strings.LastIndex(target, "/")+1:]
					}
					return textResult(fmt.Sprintf(`{"result":{"status":"success","id":"created-comment","discussion_id":%q}}`, discussion)), nil
				case "notion-set-comment-reaction":
					return textResult(`{"comment_id":"comment123","reactions":[]}`), nil
				default:
					return textResult(`{"suggested_edits_status":"included","text":"<discussions shown-count=\"0\" total-count=\"0\">\n</discussions>"}`), nil
				}
			}
			command := newRootCmd("test", "none", a)
			command.SetArgs(test.args)
			command.SetIn(strings.NewReader(test.stdin))
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			if len(calls) != 1 || calls[0].name != test.tool || !reflect.DeepEqual(calls[0].args, test.request) {
				t.Fatalf("wrong target or content: %#v", calls)
			}
			if test.tool == "notion-create-comment" {
				discussion := "created-discussion"
				if _, reply := test.request["discussion_id"]; reply {
					discussion = "discussion123"
				}
				assertPrintedJSON(t, printed, fmt.Sprintf(`{"id":"created-comment","discussion_id":%q}`, discussion))
			}
		})
	}
}

func TestInvalidCommentInputsFailBeforeAuthentication(t *testing.T) {
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"discussion", "discussion123", "reply", "Text"}, "--page is required"},
		{[]string{"discussion", "https://example.invalid/thread", "read"}, "discussion://"},
		{[]string{"discussion", "discussion://page123/discussion123", "read"}, "page, block, and discussion"},
		{[]string{"discussion", testDiscussion, "reply", "--page", "other", "Text"}, "--page is only needed"},
		{[]string{"discussion", testDiscussion, "reply", "--cursor", "cursor", "Text"}, "--cursor is not valid"},
		{[]string{"page", "page123", "comment", "--select", "", "Text"}, "selection must not be empty"},
		{[]string{"page", "page123", "comment", "--title", "wrong", "Text"}, "--title is not valid"},
		{[]string{"page", "page123", "comments", "--select", "text"}, "--select is not valid"},
		{[]string{"page", "page123", "comment", " "}, "nonempty UTF-8"},
		{[]string{"page", "page123", "comment", strings.Repeat("x", (1<<20)+1)}, "exceeds 1 MiB"},
		{[]string{"comment", "comment123", "react", ""}, "Unicode emoji"},
	} {
		t.Run(strings.Join(test.args[:min(3, len(test.args))], " ")+test.want, func(t *testing.T) {
			var calls []toolCall
			var printed []any
			a := testApp(t, &calls, &printed)
			a.ensureToken = func(context.Context) (*auth.Token, error) {
				t.Fatal("invalid input reached authentication")
				return nil, nil
			}
			err := executeTestCommand(a, test.args...)
			if err == nil || !strings.Contains(err.Error(), test.want) || len(calls) != 0 || len(printed) != 0 {
				t.Fatalf("validation: calls=%v output=%v error=%v", calls, printed, err)
			}
		})
	}
}

func TestCommentWritesDoNotClaimSuccessWithoutIDsOrOnProviderFailure(t *testing.T) {
	for _, test := range []struct {
		name  string
		reply *mcp.ToolResult
		want  string
	}{
		{"missing identifiers", textResult(`{}`), "WRITE_UNCONFIRMED"},
		{"permission", &mcp.ToolResult{IsError: true, Content: []mcp.ContentItem{{Type: "text", Text: "permission denied"}}}, "PERMISSION_DENIED"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []toolCall
			var printed []any
			a := testApp(t, &calls, &printed)
			a.callTool = func(ctx context.Context, token, tool string, args map[string]any) (*mcp.ToolResult, error) {
				calls = append(calls, toolCall{name: tool, args: args})
				return test.reply, nil
			}
			err := executeTestCommand(a, "discussion", testDiscussion, "reply", "Text")
			if err == nil || output.ErrorFrom(err).Code != test.want || len(calls) != 1 || len(printed) != 0 {
				t.Fatalf("unconfirmed write: calls=%d output=%v error=%v", len(calls), printed, err)
			}
		})
	}
}

func TestRawCommentCommandsPreserveMCPResultWithoutAnotherWrite(t *testing.T) {
	for _, args := range [][]string{
		{"--raw", "page", "page123", "comment", "Text"},
		{"--raw", "discussion", testDiscussion, "reply", "Text"},
		{"--raw", "comment", "comment123", "react", "👀"},
	} {
		t.Run(strings.Join(args[:3], " "), func(t *testing.T) {
			var calls []toolCall
			var printed []any
			a := testApp(t, &calls, &printed)
			a.callTool = func(context.Context, string, string, map[string]any) (*mcp.ToolResult, error) {
				t.Fatal("raw command used the normalized call path")
				return nil, nil
			}
			a.callToolRaw = func(ctx context.Context, token, tool string, request map[string]any) (*mcp.ToolResult, json.RawMessage, error) {
				calls = append(calls, toolCall{name: tool, args: request})
				return textResult("native payload"), json.RawMessage(`{"content":[{"type":"text","text":"native payload"}]}`), nil
			}
			if err := executeTestCommand(a, args...); err != nil {
				t.Fatal(err)
			}
			if len(calls) != 1 {
				t.Fatalf("raw write called provider %d times", len(calls))
			}
			assertPrintedJSON(t, printed, `{"content":[{"type":"text","text":"native payload"}]}`)
		})
	}
}

func TestReplyAcknowledgedInAnotherDiscussionIsNotRetried(t *testing.T) {
	var calls []toolCall
	var printed []any
	a := testApp(t, &calls, &printed)
	a.callTool = func(ctx context.Context, token, tool string, args map[string]any) (*mcp.ToolResult, error) {
		calls = append(calls, toolCall{name: tool, args: args})
		return textResult(`{"result":{"status":"success","id":"actual-comment","discussion_id":"another-discussion"}}`), nil
	}

	err := executeTestCommand(a, "discussion", testDiscussion, "reply", "Text")
	if err == nil || output.ErrorFrom(err).Code != "WRITE_UNCONFIRMED" || len(calls) != 1 {
		t.Fatalf("mismatched write: calls=%d error=%v", len(calls), err)
	}
	assertPrintedJSON(t, printed, `{"id":"actual-comment","discussion_id":"another-discussion"}`)
}
