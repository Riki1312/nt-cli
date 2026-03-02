package cli

import (
	"github.com/Riki1312/nt-cli/internal/auth"
	"github.com/Riki1312/nt-cli/internal/output"
	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Notion via OAuth",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.Login(cmd.Context()); err != nil {
				return output.AuthError("login failed: " + err.Error())
			}
			return output.Print(map[string]bool{"ok": true})
		},
	}
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.DeleteToken(); err != nil {
				return err
			}
			return output.Print(map[string]bool{"ok": true})
		},
	}
}
