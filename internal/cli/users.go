package cli

import (
	"encoding/json"

	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/spf13/cobra"
)

func newWhoamiCmd(a app) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current user and workspace info",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tok, err := a.ensureToken(cmd.Context())
			if err != nil {
				return output.AuthError(err.Error())
			}

			toolArgs := map[string]any{"user_id": "self"}

			raw, _ := cmd.Flags().GetBool("raw")
			if raw {
				return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-get-users", toolArgs)
			}

			result, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-get-users", toolArgs)
			if err != nil {
				return err
			}

			return a.print(json.RawMessage(result.TextContent()))
		},
	}
}

func newUsersCmd(a app) *cobra.Command {
	return &cobra.Command{
		Use:   "users",
		Short: "List workspace users",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tok, err := a.ensureToken(cmd.Context())
			if err != nil {
				return output.AuthError(err.Error())
			}

			raw, _ := cmd.Flags().GetBool("raw")
			if raw {
				return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-get-users", map[string]any{})
			}

			result, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-get-users", map[string]any{})
			if err != nil {
				return err
			}

			return a.print(json.RawMessage(result.TextContent()))
		},
	}
}

func newTeamsCmd(a app) *cobra.Command {
	return &cobra.Command{
		Use:   "teams",
		Short: "List workspace teams",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tok, err := a.ensureToken(cmd.Context())
			if err != nil {
				return output.AuthError(err.Error())
			}

			raw, _ := cmd.Flags().GetBool("raw")
			if raw {
				return callAndPrintRaw(cmd.Context(), a, tok.AccessToken, "notion-get-teams", map[string]any{})
			}

			result, err := callTool(cmd.Context(), a, tok.AccessToken, "notion-get-teams", map[string]any{})
			if err != nil {
				return err
			}

			return a.print(json.RawMessage(result.TextContent()))
		},
	}
}
