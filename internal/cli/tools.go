package cli

import (
	"fmt"

	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/spf13/cobra"
)

func newToolsCmd(a app) *cobra.Command {
	return &cobra.Command{
		Use:    "tools [name]",
		Short:  "List MCP tools or inspect one tool's description and schema",
		Hidden: true,
		Args:   cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tok, err := a.ensureToken(cmd.Context())
			if err != nil {
				return output.AuthError(err.Error())
			}
			tools, err := a.listTools(cmd.Context(), tok.AccessToken)
			if err != nil {
				return err
			}
			names := make([]string, 0, len(tools))
			for _, tool := range tools {
				if len(args) == 1 && tool.Name == args[0] {
					return a.print(tool)
				}
				names = append(names, tool.Name)
			}
			if len(args) == 1 {
				return output.NewError(output.ExitNotFound, "NOT_FOUND", fmt.Sprintf("MCP tool %q is unavailable", args[0]))
			}
			return a.print(names)
		},
	}
}
