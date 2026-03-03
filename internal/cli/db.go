package cli

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/Riki1312/nt-cli/internal/auth"
	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/Riki1312/nt-cli/internal/transform"
	"github.com/spf13/cobra"
)

func newDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db <id> <verb> [args...]",
		Short: "Operate on a Notion database",
		Long: `Operate on a Notion database.

The <id> is the database ID for read and query, or the data source ID
(collection ID) for create and update. Use 'db <id> read' to discover
data source IDs.

Verbs:
  read        Fetch database schema and info
  query       Query rows with SQL (e.g. nt db <id> query "SELECT * FROM _ LIMIT 10")
  create      Create a row (--props required, optional content argument)
  update      Update database schema or title (--title, --schema flags)`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbID := args[0]
			verb := args[1]
			rest := args[2:]

			switch verb {
			case "read":
				return runDBRead(cmd, dbID)
			case "query":
				return runDBQuery(cmd, dbID, rest)
			case "create":
				return runDBCreate(cmd, dbID, rest)
			case "update":
				return runDBUpdate(cmd, dbID)
			default:
				return fmt.Errorf("unknown verb %q; expected: read, query, create, update", verb)
			}
		},
	}
	cmd.Flags().String("props", "", "JSON properties for create verb")
	cmd.Flags().String("title", "", "new title for update verb")
	cmd.Flags().String("schema", "", "SQL DDL statements for update verb")
	cmd.Flags().StringSlice("params", nil, "SQL query parameters for query verb (repeatable)")
	return cmd
}

func runDBRead(cmd *cobra.Command, dbID string) error {
	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	fetchArgs := map[string]any{"id": dbID}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), tok.AccessToken, "notion-fetch", fetchArgs)
	}

	result, err := callTool(cmd.Context(), tok.AccessToken, "notion-fetch", fetchArgs)
	if err != nil {
		return err
	}

	db, err := transform.DBRead(result, dbID)
	if err != nil {
		return err
	}
	return output.Print(db)
}

var tablePlaceholder = regexp.MustCompile(`(?i)\b(FROM|JOIN)\s+_\b`)

func runDBQuery(cmd *cobra.Command, dbID string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("query requires a SQL query argument")
	}

	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	dataSourceURL := "collection://" + dbID
	query, err := readContentArg(args[0])
	if err != nil {
		return err
	}

	// Replace the placeholder "_" with the full data source URL for convenience.
	// Case-insensitive to handle FROM/from/From/JOIN/join/Join etc.
	// Users can also write the full collection:// URL directly in their SQL.
	quoted := fmt.Sprintf(`"%s"`, dataSourceURL)
	query = tablePlaceholder.ReplaceAllString(query, "${1} "+quoted)

	data := map[string]any{
		"data_source_urls": []string{dataSourceURL},
		"query":            query,
	}

	params, _ := cmd.Flags().GetStringSlice("params")
	if len(params) > 0 {
		data["params"] = params
	}

	toolArgs := map[string]any{"data": data}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), tok.AccessToken, "notion-query-data-sources", toolArgs)
	}

	result, err := callTool(cmd.Context(), tok.AccessToken, "notion-query-data-sources", toolArgs)
	if err != nil {
		return err
	}

	rows, err := transform.QueryResults(result)
	if err != nil {
		return err
	}
	return output.Print(rows)
}

func runDBCreate(cmd *cobra.Command, dbID string, args []string) error {
	propsStr, _ := cmd.Flags().GetString("props")
	if propsStr == "" {
		return fmt.Errorf("create requires --props flag with JSON properties")
	}

	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	var properties map[string]any
	if err := json.Unmarshal([]byte(propsStr), &properties); err != nil {
		return fmt.Errorf("invalid JSON properties: %w", err)
	}

	page := map[string]any{
		"properties": properties,
	}

	if len(args) > 0 {
		content, err := readContentArg(args[0])
		if err != nil {
			return err
		}
		page["content"] = content
	}

	toolArgs := map[string]any{
		"parent": map[string]any{"data_source_id": dbID},
		"pages":  []any{page},
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), tok.AccessToken, "notion-create-pages", toolArgs)
	}

	result, err := callTool(cmd.Context(), tok.AccessToken, "notion-create-pages", toolArgs)
	if err != nil {
		return err
	}

	created, err := transform.CreatedPages(result)
	if err != nil {
		return err
	}
	return output.Print(created)
}

func runDBUpdate(cmd *cobra.Command, dbID string) error {
	title, _ := cmd.Flags().GetString("title")
	schema, _ := cmd.Flags().GetString("schema")

	if title == "" && schema == "" {
		return fmt.Errorf("update requires --title and/or --schema flag")
	}

	tok, err := auth.EnsureValidToken(cmd.Context())
	if err != nil {
		return output.AuthError(err.Error())
	}

	toolArgs := map[string]any{
		"data_source_id": dbID,
	}
	if title != "" {
		toolArgs["title"] = title
	}
	if schema != "" {
		toolArgs["statements"] = schema
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return callAndPrintRaw(cmd.Context(), tok.AccessToken, "notion-update-data-source", toolArgs)
	}

	if _, err := callTool(cmd.Context(), tok.AccessToken, "notion-update-data-source", toolArgs); err != nil {
		return err
	}
	return output.Print(map[string]any{"id": dbID, "ok": true})
}
