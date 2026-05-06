package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/Riki1312/nt-cli/internal/transform"
	"github.com/spf13/cobra"
)

func newPageCmd(a app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page <id> <verb> [args...]",
		Short: "Operate on a Notion page",
		Long: `Operate on a Notion page.

Verbs:
  read        Fetch page content and properties
  set         Update page properties (JSON argument)
  replace     Find-and-replace content, or full replacement with --page
  append      Append to page content (markdown argument)
  create      Create a child page (--title required)
  move        Move page to a new parent (--to required)
  duplicate   Duplicate the page
  comments    List comments on the page
  comment     Add a comment to the page (text argument)`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			verb := args[1]
			rest := args[2:]

			switch verb {
			case "read":
				if err := requireNoArgs(verb, rest); err != nil {
					return err
				}
				return runPageRead(cmd, a, pageID)
			case "set":
				return runPageSet(cmd, a, pageID, rest)
			case "replace":
				return runPageReplace(cmd, a, pageID, rest)
			case "append":
				return runPageAppend(cmd, a, pageID, rest)
			case "create":
				return runPageCreate(cmd, a, pageID, rest)
			case "move":
				if err := requireNoArgs(verb, rest); err != nil {
					return err
				}
				return runPageMove(cmd, a, pageID)
			case "duplicate":
				if err := requireNoArgs(verb, rest); err != nil {
					return err
				}
				return runPageDuplicate(cmd, a, pageID)
			case "comments":
				if err := requireNoArgs(verb, rest); err != nil {
					return err
				}
				return runPageComments(cmd, a, pageID)
			case "comment":
				return runPageComment(cmd, a, pageID, rest)
			default:
				return fmt.Errorf("unknown verb %q; expected: read, set, replace, append, create, move, duplicate, comments, comment", verb)
			}
		},
	}
	cmd.Flags().String("title", "", "title for create verb")
	cmd.Flags().String("to", "", "target parent ID for move verb")
	cmd.Flags().Bool("page", false, "replace entire page content")
	cmd.Flags().Bool("first", false, "replace only the first occurrence for targeted replace")
	return cmd
}

func runPageRead(cmd *cobra.Command, a app, pageID string) error {
	raw, _ := cmd.Flags().GetBool("raw")

	tok, err := a.ensureToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	fetchArgs := map[string]any{"id": pageID}

	if raw {
		return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-fetch", fetchArgs)
	}

	result, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-fetch", fetchArgs)
	if err != nil {
		return err
	}

	page, err := transform.PageRead(result, pageID)
	if err != nil {
		return err
	}
	if page.Hint != "" {
		a.hint(page.Hint)
	}
	return a.print(page)
}

func runPageSet(cmd *cobra.Command, a app, pageID string, args []string) error {
	if err := requireExactArgs("set", args, 1, "a JSON properties argument"); err != nil {
		return err
	}

	tok, err := a.ensureToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	propsJSON, err := readContentArg(args[0])
	if err != nil {
		return err
	}

	var properties map[string]any
	if err := json.Unmarshal([]byte(propsJSON), &properties); err != nil {
		return fmt.Errorf("invalid JSON properties: %w", err)
	}

	toolArgs := map[string]any{
		"page_id":    pageID,
		"command":    "update_properties",
		"properties": properties,
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-update-page", toolArgs)
	}

	if _, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-update-page", toolArgs); err != nil {
		return err
	}
	return a.print(map[string]any{"id": pageID, "ok": true})
}

func runPageReplace(cmd *cobra.Command, a app, pageID string, args []string) error {
	pageFlag, _ := cmd.Flags().GetBool("page")
	firstOnly, _ := cmd.Flags().GetBool("first")

	if pageFlag {
		if firstOnly {
			return fmt.Errorf("replace --page cannot be used with --first")
		}
		if err := requireExactArgs("replace --page", args, 1, "a markdown content argument"); err != nil {
			return err
		}

		tok, err := a.ensureToken(cmd.Context())
		if err != nil {
			return output.AuthError(err.Error())
		}

		content, err := readContentArg(args[0])
		if err != nil {
			return err
		}

		toolArgs := map[string]any{
			"page_id": pageID,
			"command": "replace_content",
			"new_str": content,
		}

		raw, _ := cmd.Flags().GetBool("raw")
		if raw {
			return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-update-page", toolArgs)
		}

		if _, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-update-page", toolArgs); err != nil {
			return err
		}
		return a.print(map[string]any{"id": pageID, "ok": true})
	}

	if len(args) != 2 {
		return fmt.Errorf("replace requires '<old>' '<new>' arguments, or use --page '<markdown>' for full content replacement")
	}

	tok, err := a.ensureToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	oldStr, err := readContentArg(args[0])
	if err != nil {
		return err
	}
	newStr, err := readContentArg(args[1])
	if err != nil {
		return err
	}

	// Read the current page content so targeted replace happens client-side.
	// The hosted MCP replace_content behavior is not reliable for scoped replacement.
	fetchResult, err := a.callTool(cmd.Context(), tok.AccessToken, "notion-fetch", map[string]any{
		"id": pageID,
	})
	if err != nil {
		return fmt.Errorf("reading page before replace: %w", err)
	}
	if fetchResult.IsError {
		return output.ToolError(fetchResult.TextContent())
	}

	existing := transform.ExtractPageContent(fetchResult.TextContent())
	merged, err := replacePageContent(existing, oldStr, newStr, firstOnly)
	if err != nil {
		return err
	}
	toolArgs := map[string]any{
		"page_id": pageID,
		"command": "replace_content",
		"new_str": merged,
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-update-page", toolArgs)
	}

	if _, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-update-page", toolArgs); err != nil {
		return err
	}
	return a.print(map[string]any{"id": pageID, "ok": true})
}

func replacePageContent(existing, oldStr, newStr string, firstOnly bool) (string, error) {
	if oldStr == "" {
		return "", fmt.Errorf("replace target cannot be empty")
	}
	if !strings.Contains(existing, oldStr) {
		return "", fmt.Errorf("replace target not found in page content")
	}
	if firstOnly {
		return strings.Replace(existing, oldStr, newStr, 1), nil
	}
	return strings.ReplaceAll(existing, oldStr, newStr), nil
}

func runPageAppend(cmd *cobra.Command, a app, pageID string, args []string) error {
	if err := requireExactArgs("append", args, 1, "a markdown content argument"); err != nil {
		return err
	}

	tok, err := a.ensureToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	content, err := readContentArg(args[0])
	if err != nil {
		return err
	}

	// Read the current page content so we can append to it.
	fetchResult, err := a.callTool(cmd.Context(), tok.AccessToken, "notion-fetch", map[string]any{
		"id": pageID,
	})
	if err != nil {
		return fmt.Errorf("reading page before append: %w", err)
	}
	if fetchResult.IsError {
		return output.ToolError(fetchResult.TextContent())
	}

	existing := transform.ExtractPageContent(fetchResult.TextContent())
	var merged string
	if existing == "" {
		merged = content
	} else {
		merged = existing + "\n\n" + content
	}

	toolArgs := map[string]any{
		"page_id": pageID,
		"command": "replace_content",
		"new_str": merged,
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-update-page", toolArgs)
	}

	if _, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-update-page", toolArgs); err != nil {
		return err
	}
	return a.print(map[string]any{"id": pageID, "ok": true})
}

func runPageCreate(cmd *cobra.Command, a app, parentID string, args []string) error {
	if err := requireMaxArgs("create", args, 1, "one content argument"); err != nil {
		return err
	}

	title, _ := cmd.Flags().GetString("title")
	if title == "" {
		return fmt.Errorf("create requires --title flag")
	}

	tok, err := a.ensureToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	page := map[string]any{
		"properties": map[string]any{"title": title},
	}

	if len(args) > 0 {
		content, err := readContentArg(args[0])
		if err != nil {
			return err
		}
		page["content"] = content
	}

	toolArgs := map[string]any{
		"parent": map[string]any{"page_id": parentID},
		"pages":  []any{page},
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-create-pages", toolArgs)
	}

	result, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-create-pages", toolArgs)
	if err != nil {
		return err
	}

	created, err := transform.CreatedPages(result)
	if err != nil {
		return err
	}
	return a.print(created)
}

func runPageMove(cmd *cobra.Command, a app, pageID string) error {
	target, _ := cmd.Flags().GetString("to")
	if target == "" {
		return fmt.Errorf("move requires --to flag with target parent ID")
	}

	tok, err := a.ensureToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	toolArgs := map[string]any{
		"page_or_database_ids": []string{pageID},
		"new_parent":           map[string]any{"page_id": target},
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-move-pages", toolArgs)
	}

	if _, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-move-pages", toolArgs); err != nil {
		return err
	}
	return a.print(map[string]any{"id": pageID, "ok": true})
}

func runPageDuplicate(cmd *cobra.Command, a app, pageID string) error {
	tok, err := a.ensureToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	toolArgs := map[string]any{
		"page_id": pageID,
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-duplicate-page", toolArgs)
	}

	result, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-duplicate-page", toolArgs)
	if err != nil {
		return err
	}
	return a.print(transform.DuplicateResult(result, pageID))
}

func runPageComments(cmd *cobra.Command, a app, pageID string) error {
	tok, err := a.ensureToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	toolArgs := map[string]any{
		"page_id":            pageID,
		"include_all_blocks": true,
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-get-comments", toolArgs)
	}

	result, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-get-comments", toolArgs)
	if err != nil {
		return err
	}
	return a.print(transform.Comments(result))
}

func runPageComment(cmd *cobra.Command, a app, pageID string, args []string) error {
	if err := requireExactArgs("comment", args, 1, "a text argument"); err != nil {
		return err
	}

	tok, err := a.ensureToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	text, err := readContentArg(args[0])
	if err != nil {
		return err
	}

	toolArgs := map[string]any{
		"page_id": pageID,
		"rich_text": []map[string]any{
			{"text": map[string]any{"content": text}},
		},
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-create-comment", toolArgs)
	}

	if _, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-create-comment", toolArgs); err != nil {
		return err
	}
	return a.print(map[string]any{"id": pageID, "ok": true})
}
