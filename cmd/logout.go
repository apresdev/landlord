package cmd

import (
	"github.com/spf13/cobra"
)

func createLogoutCmd() *cobra.Command {
	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout the current logged in player",
		RunE: func(cmd *cobra.Command, args []string) error {
			ses := MustSession()
			if err := ses.Logout(); err != nil {
				return err
			}
			cmd.Println("Logged out")
			return nil
		},
	}
	return logoutCmd
}
