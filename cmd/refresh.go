package cmd

import (
	"fmt"

	"github.com/apresdev/landlord/internal/session"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/spf13/cobra"
)

func init() {
	var refreshCmd = &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the current session's tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := session.Must()
			err := authx.RefreshAuth(s)
			if err != nil {
				return err
			}
			fmt.Println("refreshed tokens")
			return nil
		},
	}
	rootCmd.AddCommand(refreshCmd)
}
