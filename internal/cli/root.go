package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "nt",
		Short: "A CLI for Notion, powered by MCP",
	}

	root.SilenceErrors = true
	root.SilenceUsage = true

	root.PersistentFlags().Bool("raw", false, "print raw MCP JSON-RPC response")
	root.PersistentFlags().Bool("verbose", false, "print request/response details to stderr")

	root.AddCommand(newLoginCmd())
	root.AddCommand(newLogoutCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newPageCmd())
	root.AddCommand(newDBCmd())
	root.AddCommand(newCreateCmd())
	root.AddCommand(newWhoamiCmd())
	root.AddCommand(newUsersCmd())
	root.AddCommand(newTeamsCmd())
	root.AddCommand(newToolsCmd())

	return root
}
