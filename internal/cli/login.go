package cli

import (
	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/spf13/cobra"
)

func newLoginCmd(a app) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Notion via OAuth",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.login(cmd.Context()); err != nil {
				return output.AuthError("login failed: " + err.Error())
			}
			return a.print(map[string]bool{"ok": true})
		},
	}
}

func newLogoutCmd(a app) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.logout(); err != nil {
				return err
			}
			return a.print(map[string]bool{"ok": true})
		},
	}
}
