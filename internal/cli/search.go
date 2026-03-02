package cli

import (
	"encoding/json"

	"github.com/Riki1312/nt-cli/internal/auth"
	"github.com/Riki1312/nt-cli/internal/mcp"
	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/Riki1312/nt-cli/internal/transform"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search across your Notion workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			raw, _ := cmd.Flags().GetBool("raw")

			tok, err := auth.EnsureValidToken(cmd.Context())
			if err != nil {
				return output.AuthError(err.Error())
			}

			if raw {
				data, err := mcp.CallToolRaw(cmd.Context(), tok.AccessToken, "notion-search", map[string]any{
					"query": query,
				})
				if err != nil {
					return err
				}
				return output.Print(json.RawMessage(data))
			}

			result, err := mcp.CallTool(cmd.Context(), tok.AccessToken, "notion-search", map[string]any{
				"query": query,
			})
			if err != nil {
				return err
			}

			if result.IsError {
				return output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
			}

			results, err := transform.SearchResults(result)
			if err != nil {
				return err
			}
			return output.Print(results)
		},
	}
}
