package cli

import (
	"fmt"

	"github.com/Riki1312/nt-cli/internal/auth"
	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/Riki1312/nt-cli/internal/transform"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [content]",
		Short: "Create a standalone page at the workspace root",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if len(args) > 0 {
				content, err := readContentArg(args[0])
				if err != nil {
					return err
				}
				page["content"] = content
			}

			toolArgs := map[string]any{
				"pages": []any{page},
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
		},
	}
	cmd.Flags().String("title", "", "page title (required)")
	return cmd
}
