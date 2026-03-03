package cli

import (
	"github.com/Riki1312/nt-cli/internal/auth"
	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/Riki1312/nt-cli/internal/transform"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search across your Notion workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			raw, _ := cmd.Flags().GetBool("raw")
			limit, _ := cmd.Flags().GetInt("limit")
			typeFilter, _ := cmd.Flags().GetString("type")

			tok, err := auth.EnsureValidToken(cmd.Context())
			if err != nil {
				return output.AuthError(err.Error())
			}

			toolArgs := map[string]any{"query": query}

			if raw {
				return callAndPrintRaw(cmd.Context(), tok.AccessToken, "notion-search", toolArgs)
			}

			result, err := callTool(cmd.Context(), tok.AccessToken, "notion-search", toolArgs)
			if err != nil {
				return err
			}

			results, err := transform.SearchResults(result)
			if err != nil {
				return err
			}

			filtered := transform.FilterSearchResults(results, typeFilter, limit)
			return output.Print(filtered)
		},
	}
	cmd.Flags().Int("limit", 0, "maximum number of results to return")
	cmd.Flags().String("type", "", "filter results by type (e.g. page, database)")
	return cmd
}
