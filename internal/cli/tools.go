package cli

import (
	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/spf13/cobra"
)

func newToolsCmd(a app) *cobra.Command {
	return &cobra.Command{
		Use:    "tools",
		Short:  "List available MCP tools (debug)",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tok, err := a.ensureToken(cmd.Context())
			if err != nil {
				return output.AuthError(err.Error())
			}

			tools, err := a.listTools(cmd.Context(), tok.AccessToken)
			if err != nil {
				return err
			}
			return a.print(tools)
		},
	}
}
