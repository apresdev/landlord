package cmd

import (
	"fmt"

	"github.com/apresdev/landlord/internal/session"

	"github.com/spf13/cobra"
)

func init() {
	var logoutCmd = &cobra.Command{
		Use:   "logout",
		Short: "Logout the current logged in player",
		RunE: func(cmd *cobra.Command, args []string) error {
			ses := session.Must()
			if err := ses.Logout(); err != nil {
				return err
			}
			fmt.Println("Logged out")
			return nil
		},
	}
	rootCmd.AddCommand(logoutCmd)
}
