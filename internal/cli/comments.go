package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/Riki1312/nt-cli/internal/transform"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newDiscussionCmd(a app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discussion <id-or-url> <read|reply> [text]",
		Short: "Read or reply to an existing discussion",
		Long:  "Use a discussion:// URL from comment reads, or a bare discussion ID with --page. Reply text supports inline Markdown and '-' for stdin. Read cursors retrieve older comments without fetching an unbounded history.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			page, _ := cmd.Flags().GetString("page")
			page, discussion, err := discussionTarget(args[0], page)
			if err != nil {
				return err
			}

			request := map[string]any{"page_id": page, "discussion_id": discussion}

			switch args[1] {
			case "read":
				if err := requireNoArgs("read", args[2:]); err != nil {
					return err
				}
				if err := commentFlags(cmd, "page", "cursor", "include-resolved"); err != nil {
					return err
				}

				cursor, _ := cmd.Flags().GetString("cursor")
				if cmd.Flags().Changed("cursor") {
					if strings.TrimSpace(cursor) == "" {
						return fmt.Errorf("cursor must not be empty")
					}
					request["cursor"] = cursor
				}
				return readComments(cmd, a, request)
			case "reply":
				if err := commentFlags(cmd, "page"); err != nil {
					return err
				}
				text, err := commentText(cmd, "reply", args[2:])
				if err != nil {
					return err
				}
				request["markdown"] = text
				return postComment(cmd, a, request)
			default:
				return fmt.Errorf("unknown discussion verb %q; expected read or reply", args[1])
			}
		},
	}
	cmd.Flags().String("page", "", "page ID or URL; required for a bare discussion ID")
	cmd.Flags().String("cursor", "", "retrieve older comments using the discussion's next_cursor")
	cmd.Flags().Bool("include-resolved", false, "include resolved discussions")

	return cmd
}

func newCommentCmd(a app) *cobra.Command {
	return &cobra.Command{
		Use:   "comment <id> <react|unreact> <emoji>",
		Short: "Set your own emoji reaction on a comment",
		Long:  "React adds your reaction; unreact removes it. Repeating the same state is harmless. Supply a standard Unicode emoji, not a :shortcode:. Provider plan and permission restrictions remain errors.",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[1] != "react" && args[1] != "unreact" {
				return fmt.Errorf("unknown comment verb %q; expected react or unreact", args[1])
			}
			if err := commentFlags(cmd); err != nil {
				return err
			}
			if !validID(args[0]) {
				return fmt.Errorf("a comment ID is required")
			}
			if strings.TrimSpace(args[2]) == "" || !utf8.ValidString(args[2]) || len(args[2]) > 128 {
				return fmt.Errorf("one standard Unicode emoji is required")
			}

			request := map[string]any{"comment_id": args[0], "emoji": args[2], "reacted": args[1] == "react"}
			token, err := a.ensureToken(cmd.Context())
			if err != nil {
				return output.AuthError(err.Error())
			}

			raw, _ := cmd.Flags().GetBool("raw")
			if raw {
				return callAndPrintRaw(cmd.Context(), a, token.AccessToken, "notion-set-comment-reaction", request)
			}

			result, err := callTool(cmd.Context(), a, token.AccessToken, "notion-set-comment-reaction", request)
			if err != nil {
				return err
			}

			var response map[string]any
			if json.Unmarshal([]byte(result.TextContent()), &response) != nil || response == nil {
				return fmt.Errorf("unrecognized reaction response; inspect the comment")
			}
			if problem, found := response["error"]; found && problem != nil && problem != "" {
				return output.ToolError(fmt.Sprint(problem))
			}
			if response["success"] == false || response["ok"] == false {
				return output.ToolError("Notion did not confirm the reaction change")
			}

			return a.print(response)
		},
	}
}

func runPageComments(cmd *cobra.Command, a app, pageID string) error {
	if strings.TrimSpace(pageID) == "" {
		return fmt.Errorf("a page ID or URL is required")
	}
	if err := commentFlags(cmd, "include-resolved"); err != nil {
		return err
	}
	return readComments(cmd, a, map[string]any{"page_id": pageID})
}

func readComments(cmd *cobra.Command, a app, request map[string]any) error {
	includeResolved, _ := cmd.Flags().GetBool("include-resolved")
	if includeResolved {
		request["include_resolved"] = true
	}

	token, err := a.ensureToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), a, token.AccessToken, "notion-get-comments", request)
	}

	result, err := callTool(cmd.Context(), a, token.AccessToken, "notion-get-comments", request)
	if err != nil {
		return err
	}

	return a.print(transform.Comments(result))
}

func runPageComment(cmd *cobra.Command, a app, pageID string, args []string) error {
	if strings.TrimSpace(pageID) == "" {
		return fmt.Errorf("a page ID or URL is required")
	}
	if err := commentFlags(cmd, "select"); err != nil {
		return err
	}

	text, err := commentText(cmd, "comment", args)
	if err != nil {
		return err
	}

	request := map[string]any{"page_id": pageID, "markdown": text}
	if cmd.Flags().Changed("select") {
		selection, _ := cmd.Flags().GetString("select")
		if strings.TrimSpace(selection) == "" {
			return fmt.Errorf("selection must not be empty")
		}
		request["selection_with_ellipsis"] = selection
	}

	return postComment(cmd, a, request)
}

func postComment(cmd *cobra.Command, a app, request map[string]any) error {
	token, err := a.ensureToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), a, token.AccessToken, "notion-create-comment", request)
	}

	result, err := callTool(cmd.Context(), a, token.AccessToken, "notion-create-comment", request)
	if err != nil {
		return fmt.Errorf("comment request failed; inspect the discussion before posting again: %w", err)
	}

	// Never manufacture a created identifier from the requested page or discussion.
	for _, content := range result.Content {
		if content.Type != "text" {
			continue
		}
		var posted struct {
			ID           string `json:"id"`
			DiscussionID string `json:"discussion_id"`
		}
		if json.Unmarshal([]byte(content.Text), &posted) == nil && posted.ID != "" && posted.DiscussionID != "" {
			return a.print(posted)
		}
	}
	return output.NewError(output.ExitError, "WRITE_UNCONFIRMED", "comment response has no verified comment/discussion IDs; inspect the discussion before posting again")
}

func commentText(cmd *cobra.Command, verb string, args []string) (string, error) {
	if err := requireExactArgs(verb, args, 1, "one text argument, or '-' for stdin"); err != nil {
		return "", err
	}

	const limit = 1 << 20
	text := args[0]
	if text == "-" {
		data, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), limit+1))
		if err != nil {
			return "", fmt.Errorf("reading comment: %w", err)
		}
		text = string(data)
	}

	if len(text) > limit {
		return "", fmt.Errorf("comment exceeds 1 MiB")
	}
	if strings.TrimSpace(text) == "" || !utf8.ValidString(text) {
		return "", fmt.Errorf("comment must contain valid nonempty UTF-8 text")
	}
	return text, nil
}

func commentFlags(cmd *cobra.Command, allowed ...string) error {
	var err error
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		if err == nil && flag.Name != "raw" && !slices.Contains(allowed, flag.Name) {
			err = fmt.Errorf("--%s is not valid for this operation", flag.Name)
		}
	})
	return err
}

func discussionTarget(target, page string) (string, string, error) {
	if strings.Contains(target, ":") {
		parsed, err := url.Parse(target)
		if err != nil || parsed.Scheme != "discussion" || parsed.User != nil ||
			parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" {
			return "", "", fmt.Errorf("use a discussion:// URL or a bare discussion ID with --page")
		}
		parts := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
		if !validID(parsed.Host) || len(parts) != 2 || !validID(parts[0]) || !validID(parts[1]) {
			return "", "", fmt.Errorf("discussion URL must contain page, block, and discussion IDs")
		}
		if page != "" {
			return "", "", fmt.Errorf("--page is only needed with a bare discussion ID")
		}
		return parsed.Host, target, nil
	}

	if !validID(target) {
		return "", "", fmt.Errorf("a discussion ID or discussion:// URL is required")
	}
	if strings.TrimSpace(page) == "" {
		return "", "", fmt.Errorf("--page is required with a bare discussion ID")
	}
	return page, target, nil
}

func validID(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if !(character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || character == '-' || character == '_') {
			return false
		}
	}
	return true
}
