package transform

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/Riki1312/nt-cli/internal/mcp"
)

// CommentList contains the discussions supplied by one hosted-tool response.
// Complete is nil when the provider returned only availability metadata.
// Counts and cursors describe this response; callers own bounded continuation.
type CommentList struct {
	SuggestedEditsStatus string            `json:"suggested_edits_status,omitempty"`
	ShownCount           *int              `json:"shown_count"`
	TotalCount           *int              `json:"total_count"`
	OmittedTruncated     *bool             `json:"omitted_truncated"`
	OmittedDiscussions   []string          `json:"omitted_discussions"`
	Discussions          []Discussion      `json:"discussions"`
	Complete             *bool             `json:"complete"`
	Attributes           map[string]string `json:"attributes,omitempty"`
}

// Discussion retains its provider reference, anchor, visible comments and cursor.
// A nil Resolved means the provider did not supply resolution state.
type Discussion struct {
	ID                string            `json:"id"`
	Type              string            `json:"type,omitempty"`
	Kind              string            `json:"kind,omitempty"`
	Context           string            `json:"context,omitempty"`
	TextContext       string            `json:"text_context,omitempty"`
	Resolved          *bool             `json:"resolved,omitempty"`
	CommentCount      int               `json:"comment_count"`
	TotalCommentCount int               `json:"total_comment_count"`
	NextCursor        string            `json:"next_cursor,omitempty"`
	Comments          []Comment         `json:"comments"`
	Attributes        map[string]string `json:"attributes,omitempty"`
}

// Comment retains identity references and the provider's rendered inline content.
// Content is opaque Notion markup, not plain text or trusted XML. UserID is
// extracted only from an explicit user:// reference; names are never inferred.
type Comment struct {
	ID         string            `json:"id"`
	URL        string            `json:"url,omitempty"`
	UserURL    string            `json:"user_url,omitempty"`
	UserID     string            `json:"user_id,omitempty"`
	Datetime   string            `json:"datetime,omitempty"`
	ByYou      bool              `json:"by_you"`
	Content    string            `json:"content"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// CreatedComment identifies the comment and discussion acknowledged by Notion.
type CreatedComment struct {
	ID           string `json:"id"`
	DiscussionID string `json:"discussion_id"`
}

// ReactionState is Notion's current reaction list for one comment.
type ReactionState struct {
	CommentID string     `json:"comment_id"`
	Reactions []Reaction `json:"reactions"`
}

// Reaction retains the emoji, provider user references and optional own-state flag.
type Reaction struct {
	Emoji                string   `json:"emoji"`
	ReactedByCurrentUser *bool    `json:"reacted_by_current_user"`
	ReactorURLs          []string `json:"reactor_urls"`
}

// Comments normalizes the JSON envelope and line-framed Notion discussion markup.
// Comment bodies remain opaque: they can contain unescaped XML-like text.
// Unsupported framing, inconsistent counts or duplicate IDs fail visibly.
func Comments(result *mcp.ToolResult) (*CommentList, error) {
	objects, err := commentObjects(result)
	if err != nil {
		return nil, err
	}

	list := &CommentList{
		Discussions:        []Discussion{},
		OmittedDiscussions: []string{},
	}
	var listing string
	for _, object := range objects {
		if value, present := object["suggested_edits_status"]; present {
			var status string
			if json.Unmarshal(value, &status) != nil {
				return nil, fmt.Errorf("invalid suggested-edit availability")
			}
			if list.SuggestedEditsStatus != "" && list.SuggestedEditsStatus != status {
				return nil, fmt.Errorf("conflicting suggested-edit availability")
			}
			list.SuggestedEditsStatus = status
		}
		if value, present := object["text"]; present {
			if listing != "" {
				return nil, fmt.Errorf("multiple discussion listings in one response")
			}
			if json.Unmarshal(value, &listing) != nil || listing == "" {
				return nil, fmt.Errorf("invalid or empty discussion text")
			}
		}
	}

	if listing == "" {
		if list.SuggestedEditsStatus == "" {
			return nil, fmt.Errorf("response has no discussion listing or availability metadata")
		}
		return list, nil
	}
	if err := parseDiscussions(listing, list); err != nil {
		return nil, err
	}

	return list, nil
}

// PostedComment returns only actual IDs from a successful creation acknowledgement.
// An unsupported result can follow a completed write; callers must inspect before retrying.
func PostedComment(result *mcp.ToolResult) (*CreatedComment, error) {
	objects, err := commentObjects(result)
	if err != nil {
		return nil, err
	}

	var posted *CreatedComment
	for _, object := range objects {
		value, present := object["result"]
		if !present {
			continue
		}
		var response struct {
			Status       string `json:"status"`
			ID           string `json:"id"`
			DiscussionID string `json:"discussion_id"`
		}
		if json.Unmarshal(value, &response) != nil || response.Status != "success" || response.ID == "" || response.DiscussionID == "" {
			return nil, fmt.Errorf("unrecognized comment creation acknowledgement")
		}
		if posted != nil {
			return nil, fmt.Errorf("multiple comment creation acknowledgements")
		}
		posted = &CreatedComment{ID: response.ID, DiscussionID: response.DiscussionID}
	}
	if posted == nil {
		return nil, fmt.Errorf("response has no confirmed comment and discussion IDs")
	}
	return posted, nil
}

// CommentReactions returns the provider's explicit reaction list, including an
// empty list after removal. Missing reaction data is an error, not an empty list.
func CommentReactions(result *mcp.ToolResult) (*ReactionState, error) {
	objects, err := commentObjects(result)
	if err != nil {
		return nil, err
	}

	var state *ReactionState
	for _, object := range objects {
		if _, present := object["comment_id"]; !present {
			continue
		}
		if state != nil {
			return nil, fmt.Errorf("multiple reaction results")
		}
		data, err := json.Marshal(object)
		if err != nil {
			return nil, err
		}
		state = &ReactionState{}
		if json.Unmarshal(data, state) != nil || state.CommentID == "" || state.Reactions == nil {
			return nil, fmt.Errorf("unrecognized comment reaction result")
		}
		for _, reaction := range state.Reactions {
			if reaction.Emoji == "" {
				return nil, fmt.Errorf("reaction result has no emoji")
			}
		}
	}
	if state == nil {
		return nil, fmt.Errorf("response has no comment reaction state")
	}
	return state, nil
}

func commentObjects(result *mcp.ToolResult) ([]map[string]json.RawMessage, error) {
	if result == nil || result.IsError {
		return nil, fmt.Errorf("comment tool failed")
	}

	objects := []map[string]json.RawMessage{}
	total := 0
	for _, block := range result.Content {
		total += len(block.Text)
		if total > 8<<20 {
			return nil, fmt.Errorf("comment response exceeds 8 MiB")
		}
		if block.Type != "text" {
			return nil, fmt.Errorf("unsupported comment response content type")
		}
		var object map[string]json.RawMessage
		if json.Unmarshal([]byte(block.Text), &object) != nil || object == nil {
			return nil, fmt.Errorf("comment response must contain JSON objects; inspect --raw output")
		}
		if problem, present := object["error"]; present && string(problem) != "null" {
			return nil, fmt.Errorf("comment response reported an error")
		}
		objects = append(objects, object)
	}
	if len(objects) == 0 {
		return nil, fmt.Errorf("empty comment tool response")
	}
	return objects, nil
}

func parseDiscussions(text string, list *CommentList) error {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[len(lines)-1]) != "</discussions>" {
		return fmt.Errorf("unsupported discussion framing")
	}
	attributes, tail, err := commentOpening(strings.TrimSpace(lines[0]), "discussions")
	if err != nil || tail != "" {
		return fmt.Errorf("invalid discussion listing header")
	}
	shown, err := commentCount(attributes, "shown-count")
	if err != nil {
		return err
	}
	total, err := commentCount(attributes, "total-count")
	if err != nil || shown > total {
		return fmt.Errorf("invalid discussion counts")
	}
	truncated, err := commentBool(attributes, "omitted-truncated")
	if err != nil {
		return err
	}
	if truncated == nil {
		value := false
		truncated = &value
	}

	list.ShownCount, list.TotalCount, list.OmittedTruncated = &shown, &total, truncated
	list.Attributes = extraCommentAttributes(attributes, "shown-count", "total-count", "omitted-truncated")
	var current *Discussion
	seenComments, seenDiscussions := map[string]bool{}, map[string]bool{}
	complete := shown == total && !*truncated

	for _, source := range lines[1 : len(lines)-1] {
		line := strings.TrimSpace(source)
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "<discussion "):
			if current != nil {
				return fmt.Errorf("nested discussion")
			}
			attributes, tail, err := commentOpening(line, "discussion")
			if err != nil || tail != "" {
				return fmt.Errorf("invalid discussion header")
			}
			if attributes["id"] == "" || seenDiscussions[attributes["id"]] {
				return fmt.Errorf("missing or duplicate discussion ID")
			}
			seenDiscussions[attributes["id"]] = true
			count, err := commentCount(attributes, "comment-count")
			if err != nil {
				return err
			}
			totalComments, err := commentCount(attributes, "total-comment-count")
			if err != nil || count > totalComments {
				return fmt.Errorf("invalid comment counts")
			}
			resolved, err := commentBool(attributes, "resolved")
			if err != nil {
				return err
			}
			current = &Discussion{
				ID:                attributes["id"],
				Type:              attributes["type"],
				Kind:              attributes["kind"],
				Context:           attributes["context"],
				TextContext:       attributes["text-context"],
				Resolved:          resolved,
				CommentCount:      count,
				TotalCommentCount: totalComments,
				NextCursor:        attributes["next-cursor"],
				Comments:          []Comment{},
				Attributes: extraCommentAttributes(attributes, "id", "type", "kind", "context",
					"text-context", "resolved", "comment-count", "total-comment-count", "next-cursor"),
			}
			if count < totalComments || current.NextCursor != "" {
				complete = false
			}
		case line == "</discussion>":
			if current == nil || len(current.Comments) != current.CommentCount {
				return fmt.Errorf("comment count does not match the supplied records")
			}
			list.Discussions = append(list.Discussions, *current)
			current = nil
		case strings.HasPrefix(line, "<comment "):
			if current == nil {
				return fmt.Errorf("comment outside a discussion")
			}
			attributes, body, err := commentOpening(line, "comment")
			if err != nil || !strings.HasSuffix(body, "</comment>") {
				return fmt.Errorf("unsupported comment framing")
			}
			if attributes["id"] == "" || seenComments[attributes["id"]] {
				return fmt.Errorf("missing or duplicate comment ID")
			}
			seenComments[attributes["id"]] = true
			byYou, err := commentBool(attributes, "by-you")
			if err != nil {
				return err
			}
			comment := Comment{
				ID:         attributes["id"],
				URL:        attributes["url"],
				UserURL:    attributes["user-url"],
				Datetime:   attributes["datetime"],
				ByYou:      byYou != nil && *byYou,
				Content:    strings.TrimSuffix(body, "</comment>"),
				Attributes: extraCommentAttributes(attributes, "id", "url", "user-url", "datetime", "by-you"),
			}
			if reference, err := url.Parse(comment.UserURL); err == nil && reference.Scheme == "user" && reference.User == nil {
				comment.UserID = reference.Host
			}
			current.Comments = append(current.Comments, comment)
		case strings.HasPrefix(line, "<omitted-discussion "):
			attributes, tail, err := commentOpening(line, "omitted-discussion")
			if current != nil || err != nil || tail != "" || attributes["id"] == "" || !strings.HasSuffix(line, "/>") {
				return fmt.Errorf("invalid omitted discussion")
			}
			list.OmittedDiscussions = append(list.OmittedDiscussions, attributes["id"])
		default:
			return fmt.Errorf("unsupported discussion record")
		}
	}

	if current != nil || len(list.Discussions) != shown || len(list.OmittedDiscussions) > total-shown {
		return fmt.Errorf("discussion counts do not match the supplied records")
	}
	for _, id := range list.OmittedDiscussions {
		if seenDiscussions[id] {
			return fmt.Errorf("listed discussion is also marked omitted")
		}
		seenDiscussions[id] = true
	}

	list.Complete = &complete
	return nil
}

// Only metadata opening tags are XML. Bodies deliberately remain opaque because
// Notion emits rendered content such as unescaped '&' and backslash-escaped '<'.
func commentOpening(line, expected string) (map[string]string, string, error) {
	var quote byte
	end := -1
	for index := 0; index < len(line); index++ {
		character := line[index]
		if quote != 0 {
			if character == quote {
				quote = 0
			}
		} else if character == '\'' || character == '"' {
			quote = character
		} else if character == '>' {
			end = index + 1
			break
		}
	}
	if end < 0 {
		return nil, "", fmt.Errorf("unclosed metadata tag")
	}

	decoder := xml.NewDecoder(strings.NewReader(line[:end]))
	token, err := decoder.Token()
	if err != nil {
		return nil, "", err
	}
	start, ok := token.(xml.StartElement)
	if !ok || start.Name.Local != expected || start.Name.Space != "" || decoder.InputOffset() != int64(end) {
		return nil, "", fmt.Errorf("unexpected metadata tag")
	}

	attributes := map[string]string{}
	for _, attribute := range start.Attr {
		name := attribute.Name.Local
		if attribute.Name.Space != "" {
			name = attribute.Name.Space + ":" + name
		}
		if _, present := attributes[name]; present {
			return nil, "", fmt.Errorf("duplicate metadata attribute")
		}
		attributes[name] = attribute.Value
	}
	return attributes, line[end:], nil
}

func commentCount(attributes map[string]string, key string) (int, error) {
	count, err := strconv.Atoi(attributes[key])
	if err != nil || count < 0 {
		return 0, fmt.Errorf("invalid or missing %s", key)
	}
	return count, nil
}

func commentBool(attributes map[string]string, key string) (*bool, error) {
	value, present := attributes[key]
	if !present {
		return nil, nil
	}
	if value != "true" && value != "false" {
		return nil, fmt.Errorf("invalid %s", key)
	}
	result := value == "true"
	return &result, nil
}

func extraCommentAttributes(attributes map[string]string, known ...string) map[string]string {
	var extra map[string]string
	for key, value := range attributes {
		if !slices.Contains(known, key) {
			if extra == nil {
				extra = map[string]string{}
			}
			extra[key] = value
		}
	}
	return extra
}
