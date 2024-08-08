package cmd

import (
	"github.com/spf13/cobra"
)

func logoutCmd() *cobra.Command {
	c := &cobra.Command{
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
	return c
}
