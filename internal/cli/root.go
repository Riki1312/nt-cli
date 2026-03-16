package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCmd(version, commit string) *cobra.Command {
	root := &cobra.Command{
		Use:     "nt",
		Short:   "A CLI for Notion, powered by MCP",
		Version: version + " (" + commit + ")",
		Long: `A CLI for Notion, powered by MCP.

All output is JSON to stdout. Errors are JSON to stderr.
IDs can be passed with or without dashes (e.g. abc123 or a1b2-c3d4-...).
Content arguments accept "-" to read from stdin.`,
	}

	root.SilenceErrors = true
	root.SilenceUsage = true

	root.PersistentFlags().Bool("raw", false, "print raw MCP JSON-RPC response")

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
