package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd builds the nt command tree.
func NewRootCmd(version, commit string) *cobra.Command {
	return newRootCmd(version, commit, newApp())
}

func newRootCmd(version, commit string, a app) *cobra.Command {
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

	root.AddCommand(newLoginCmd(a))
	root.AddCommand(newLogoutCmd(a))
	root.AddCommand(newSearchCmd(a))
	root.AddCommand(newPageCmd(a))
	root.AddCommand(newDBCmd(a))
	root.AddCommand(newCreateCmd(a))
	root.AddCommand(newWhoamiCmd(a))
	root.AddCommand(newUsersCmd(a))
	root.AddCommand(newTeamsCmd(a))
	root.AddCommand(newToolsCmd(a))

	return root
}
