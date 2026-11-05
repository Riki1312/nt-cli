package transform

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Riki1312/nt-cli/internal/mcp"
)

func TestCommentsGoldenPreservesIdentityAnchorsAndOpaqueMarkup(t *testing.T) {
	got, err := Comments(textResult(t, "comments_response.json"))
	if err != nil {
		t.Fatal(err)
	}
	assertGoldenJSON(t, got, "comments.golden.json")
}

func TestCommentPaginationAndUnknownCompleteness(t *testing.T) {
	payload := `<discussions total-count="4" shown-count="1" omitted-truncated="true">
<discussion id="discussion://page/block/one" comment-count="1" total-comment-count="3" resolved="false" next-cursor="older-cursor">
<comment id="one">Latest comment.</comment>
</discussion>
<omitted-discussion id="discussion://page/block/two"/>
<omitted-discussion id="discussion://page/block/three"/>
</discussions>`
	result, err := Comments(discussionFixture(t, payload))
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete == nil || *result.Complete || result.ShownCount == nil || *result.ShownCount != 1 || *result.TotalCount != 4 || !*result.OmittedTruncated || len(result.OmittedDiscussions) != 2 || result.Discussions[0].NextCursor != "older-cursor" {
		t.Fatalf("partial result hidden: %+v", result)
	}

	// A terminal older page may still be only part of the whole discussion.
	older := strings.ReplaceAll(payload, ` next-cursor="older-cursor"`, ``)
	result, err = Comments(discussionFixture(t, older))
	if err != nil || result.Complete == nil || *result.Complete {
		t.Fatalf("terminal partial page: %+v %v", result, err)
	}

	metadata := &mcp.ToolResult{Content: []mcp.ContentItem{{Type: "text", Text: `{"suggested_edits_status":"included"}`}}}
	result, err = Comments(metadata)
	if err != nil || result.Complete != nil || result.ShownCount != nil || result.TotalCount != nil || result.OmittedTruncated != nil || len(result.Discussions) != 0 {
		t.Fatalf("metadata invented completeness: %+v, %v", result, err)
	}
}

func TestCommentsRejectUnsupportedOrInconsistentRecords(t *testing.T) {
	valid := `<discussions total-count="1" shown-count="1">
<discussion id="discussion://page/block/one" comment-count="1" total-comment-count="1" resolved="false">
<comment id="one">Body & literal \< text.</comment>
</discussion>
</discussions>`
	for _, test := range []struct{ name, payload string }{
		{"missing root counts", strings.Replace(valid, `total-count="1"`, ``, 1)},
		{"wrong row count", strings.Replace(valid, `shown-count="1"`, `shown-count="2"`, 1)},
		{"missing comment ID", strings.Replace(valid, `id="one"`, ``, 1)},
		{"inconsistent comment count", strings.Replace(valid, `comment-count="1"`, `comment-count="0"`, 1)},
		{"unsupported record", strings.Replace(valid, `</discussion>`, `<unexpected/>`+"\n"+`</discussion>`, 1)},
		{"unclosed discussion", strings.Replace(valid, "</discussion>\n", ``, 1)},
		{"invalid boolean", strings.Replace(valid, `resolved="false"`, `resolved="unknown"`, 1)},
		{"duplicate ID", strings.Replace(strings.Replace(valid, `comment-count="1" total-comment-count="1"`, `comment-count="2" total-comment-count="2"`, 1), `</discussion>`, `<comment id="one">Duplicate.</comment>`+"\n"+`</discussion>`, 1)},
		{"duplicate attribute", strings.Replace(valid, `id="one"`, `id="one" id="two"`, 1)},
		{"extra root", valid + "\n" + valid},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Comments(discussionFixture(t, test.payload)); err == nil {
				t.Fatal("unsupported result accepted")
			}
		})
	}
	for _, payload := range []string{`{}`, `[]`, `invalid JSON`, `{"text":12}`, `{"suggested_edits_status":"included","text":null}`, `{"suggested_edits_status":"included","text":""}`} {
		if _, err := Comments(&mcp.ToolResult{Content: []mcp.ContentItem{{Type: "text", Text: payload}}}); err == nil {
			t.Fatalf("unsupported payload accepted: %s", payload)
		}
	}
}

func TestActualCommentCreationAndReactionEnvelopes(t *testing.T) {
	posted, err := PostedComment(textResult(t, "comment_created_response.json"))
	if err != nil || posted.ID != "55555555-5555-4555-8555-555555555555" || posted.DiscussionID != "33333333-3333-4333-8333-333333333333" {
		t.Fatalf("creation: %+v %v", posted, err)
	}
	state, err := CommentReactions(textResult(t, "comment_reactions_response.json"))
	if err != nil || len(state.Reactions) != 1 || state.Reactions[0].Emoji != "👀" || state.Reactions[0].ReactedByCurrentUser == nil || !*state.Reactions[0].ReactedByCurrentUser {
		t.Fatalf("reaction: %+v %v", state, err)
	}
	removed := &mcp.ToolResult{Content: []mcp.ContentItem{{Type: "text", Text: `{"comment_id":"comment","reactions":[]}`}}}
	state, err = CommentReactions(removed)
	if err != nil || state.Reactions == nil || len(state.Reactions) != 0 {
		t.Fatalf("removal: %+v %v", state, err)
	}
	for _, payload := range []string{`{}`, `{"result":{"status":"success","id":"comment"}}`, `{"result":{"status":"error","id":"comment","discussion_id":"thread"}}`} {
		if _, err := PostedComment(&mcp.ToolResult{Content: []mcp.ContentItem{{Type: "text", Text: payload}}}); err == nil {
			t.Fatal("unverified write accepted")
		}
	}
	if _, err := CommentReactions(&mcp.ToolResult{Content: []mcp.ContentItem{{Type: "text", Text: `{"comment_id":"comment"}`}}}); err == nil {
		t.Fatal("missing reaction list became empty")
	}
}

func discussionFixture(t *testing.T, text string) *mcp.ToolResult {
	t.Helper()
	data, err := json.Marshal(map[string]string{"suggested_edits_status": "included", "text": text})
	if err != nil {
		t.Fatal(err)
	}
	return &mcp.ToolResult{Content: []mcp.ContentItem{{Type: "text", Text: string(data)}}}
}
