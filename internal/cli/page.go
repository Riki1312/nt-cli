package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Riki1312/nt-cli/internal/auth"
	"github.com/Riki1312/nt-cli/internal/mcp"
	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/Riki1312/nt-cli/internal/transform"
	"github.com/spf13/cobra"
)

func newPageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page <id> <verb> [args...]",
		Short: "Operate on a Notion page",
		Long: `Operate on a Notion page.

Verbs:
  read        Fetch page content and properties
  set         Update page properties (JSON argument)
  write       Replace page content (markdown argument)
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
				return runPageRead(cmd, pageID)
			case "set":
				return runPageSet(cmd, pageID, rest)
			case "write":
				return runPageWrite(cmd, pageID, rest)
			case "append":
				return runPageAppend(cmd, pageID, rest)
			case "create":
				return runPageCreate(cmd, pageID, rest)
			case "move":
				return runPageMove(cmd, pageID)
			case "duplicate":
				return runPageDuplicate(cmd, pageID)
			case "comments":
				return runPageComments(cmd, pageID)
			case "comment":
				return runPageComment(cmd, pageID, rest)
			default:
				return fmt.Errorf("unknown verb %q; expected: read, set, write, append, create, move, duplicate, comments, comment", verb)
			}
		},
	}
	cmd.Flags().String("title", "", "title for create verb")
	cmd.Flags().String("to", "", "target parent ID for move verb")
	return cmd
}

func runPageRead(cmd *cobra.Command, pageID string) error {
	raw, _ := cmd.Flags().GetBool("raw")

	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	if raw {
		data, err := mcp.CallToolRaw(cmd.Context(), tok.AccessToken, "notion-fetch", map[string]any{
			"id": pageID,
		})
		if err != nil {
			return err
		}
		return output.Print(json.RawMessage(data))
	}

	result, err := mcp.CallTool(cmd.Context(), tok.AccessToken, "notion-fetch", map[string]any{
		"id": pageID,
	})
	if err != nil {
		return err
	}

	if result.IsError {
		return output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
	}

	page, err := transform.PageRead(result, pageID)
	if err != nil {
		return err
	}
	return output.Print(page)
}

func runPageSet(cmd *cobra.Command, pageID string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("set requires a JSON properties argument")
	}

	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	var properties map[string]any
	if err := json.Unmarshal([]byte(args[0]), &properties); err != nil {
		return fmt.Errorf("invalid JSON properties: %w", err)
	}

	toolArgs := map[string]any{
		"page_id":    pageID,
		"command":    "update_properties",
		"properties": properties,
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		data, err := mcp.CallToolRaw(cmd.Context(), tok.AccessToken, "notion-update-page", toolArgs)
		if err != nil {
			return err
		}
		return output.Print(json.RawMessage(data))
	}

	result, err := mcp.CallTool(cmd.Context(), tok.AccessToken, "notion-update-page", toolArgs)
	if err != nil {
		return err
	}

	if result.IsError {
		return output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
	}

	return output.Print(map[string]any{"id": pageID, "ok": true})
}

func runPageWrite(cmd *cobra.Command, pageID string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("write requires a markdown content argument")
	}

	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	content, err := readContentArg(args[0])
	if err != nil {
		return err
	}

	// TODO: support --replace '<old>' '<new>' for replace_content_range

	toolArgs := map[string]any{
		"page_id": pageID,
		"command": "replace_content",
		"new_str": content,
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		data, err := mcp.CallToolRaw(cmd.Context(), tok.AccessToken, "notion-update-page", toolArgs)
		if err != nil {
			return err
		}
		return output.Print(json.RawMessage(data))
	}

	result, err := mcp.CallTool(cmd.Context(), tok.AccessToken, "notion-update-page", toolArgs)
	if err != nil {
		return err
	}

	if result.IsError {
		return output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
	}

	return output.Print(map[string]any{"id": pageID, "ok": true})
}

func runPageAppend(cmd *cobra.Command, pageID string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("append requires a markdown content argument")
	}

	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	content, err := readContentArg(args[0])
	if err != nil {
		return err
	}

	// Read the current page content so we can append to it.
	fetchResult, err := mcp.CallTool(cmd.Context(), tok.AccessToken, "notion-fetch", map[string]any{
		"id": pageID,
	})
	if err != nil {
		return fmt.Errorf("reading page before append: %w", err)
	}
	if fetchResult.IsError {
		return output.NewError(output.ExitError, "TOOL_ERROR", fetchResult.TextContent())
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
		data, err := mcp.CallToolRaw(cmd.Context(), tok.AccessToken, "notion-update-page", toolArgs)
		if err != nil {
			return err
		}
		return output.Print(json.RawMessage(data))
	}

	result, err := mcp.CallTool(cmd.Context(), tok.AccessToken, "notion-update-page", toolArgs)
	if err != nil {
		return err
	}

	if result.IsError {
		return output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
	}

	return output.Print(map[string]any{"id": pageID, "ok": true})
}

func runPageCreate(cmd *cobra.Command, parentID string, args []string) error {
	title, _ := cmd.Flags().GetString("title")
	if title == "" {
		return fmt.Errorf("create requires --title flag")
	}

	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	page := map[string]any{
		"properties": map[string]any{"title": title},
	}

	// Optional content argument
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
		data, err := mcp.CallToolRaw(cmd.Context(), tok.AccessToken, "notion-create-pages", toolArgs)
		if err != nil {
			return err
		}
		return output.Print(json.RawMessage(data))
	}

	result, err := mcp.CallTool(cmd.Context(), tok.AccessToken, "notion-create-pages", toolArgs)
	if err != nil {
		return err
	}
	if result.IsError {
		return output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
	}

	created, err := transform.CreatedPages(result)
	if err != nil {
		return err
	}
	return output.Print(created)
}

func runPageMove(cmd *cobra.Command, pageID string) error {
	target, _ := cmd.Flags().GetString("to")
	if target == "" {
		return fmt.Errorf("move requires --to flag with target parent ID")
	}

	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	toolArgs := map[string]any{
		"page_or_database_ids": []string{pageID},
		"new_parent":           map[string]any{"page_id": target},
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		data, err := mcp.CallToolRaw(cmd.Context(), tok.AccessToken, "notion-move-pages", toolArgs)
		if err != nil {
			return err
		}
		return output.Print(json.RawMessage(data))
	}

	result, err := mcp.CallTool(cmd.Context(), tok.AccessToken, "notion-move-pages", toolArgs)
	if err != nil {
		return err
	}
	if result.IsError {
		return output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
	}

	return output.Print(map[string]any{"id": pageID, "ok": true})
}

func runPageDuplicate(cmd *cobra.Command, pageID string) error {
	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	toolArgs := map[string]any{
		"page_id": pageID,
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		data, err := mcp.CallToolRaw(cmd.Context(), tok.AccessToken, "notion-duplicate-page", toolArgs)
		if err != nil {
			return err
		}
		return output.Print(json.RawMessage(data))
	}

	result, err := mcp.CallTool(cmd.Context(), tok.AccessToken, "notion-duplicate-page", toolArgs)
	if err != nil {
		return err
	}
	if result.IsError {
		return output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
	}

	return output.Print(transform.DuplicateResult(result, pageID))
}

func runPageComments(cmd *cobra.Command, pageID string) error {
	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	toolArgs := map[string]any{
		"page_id":            pageID,
		"include_all_blocks": true,
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		data, err := mcp.CallToolRaw(cmd.Context(), tok.AccessToken, "notion-get-comments", toolArgs)
		if err != nil {
			return err
		}
		return output.Print(json.RawMessage(data))
	}

	result, err := mcp.CallTool(cmd.Context(), tok.AccessToken, "notion-get-comments", toolArgs)
	if err != nil {
		return err
	}
	if result.IsError {
		return output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
	}

	return output.Print(transform.Comments(result))
}

func runPageComment(cmd *cobra.Command, pageID string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("comment requires a text argument")
	}

	tok, err := auth.EnsureValidToken(cmd.Context())
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
		data, err := mcp.CallToolRaw(cmd.Context(), tok.AccessToken, "notion-create-comment", toolArgs)
		if err != nil {
			return err
		}
		return output.Print(json.RawMessage(data))
	}

	result, err := mcp.CallTool(cmd.Context(), tok.AccessToken, "notion-create-comment", toolArgs)
	if err != nil {
		return err
	}
	if result.IsError {
		return output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
	}

	return output.Print(map[string]any{"id": pageID, "ok": true})
}

// readContentArg reads content from the argument or from stdin if the argument is "-".
func readContentArg(arg string) (string, error) {
	if arg == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading from stdin: %w", err)
		}
		return string(data), nil
	}
	return arg, nil
}
