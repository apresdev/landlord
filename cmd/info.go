package cmd

import (
	"fmt"

	"github.com/apresdev/landlord/internal/session"

	"github.com/spf13/cobra"
)

func ifSet(s, setV, unsetV string) string {
	if s == "" {
		return unsetV
	}
	return setV
}

func init() {
	var verbose bool
	var infoCmd = &cobra.Command{
		Use:   "info",
		Short: "Display info about the current session",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := session.Must()
			fmt.Printf("User pool : %s\n", s.UserPool)
			fmt.Printf("Email     : %s\n", s.Email)
			if verbose {
				fmt.Printf("Access Token:\n%s\n", s.AccessToken)
				fmt.Printf("ID Token:\n%s\n", s.IdToken)
				fmt.Printf("RefreshToken:\n%s\n", s.RefreshToken)
			} else {
				fmt.Printf("Tokens:\n")
				fmt.Printf("  Access  : %s\n", ifSet(s.AccessToken, "valid", "empty"))
				fmt.Printf("  ID      : %s\n", ifSet(s.IdToken, "valid", "empty"))
				fmt.Printf("  Refresh : %s\n", ifSet(s.RefreshToken, "valid", "empty"))
			}
			return nil
		},
	}
	infoCmd.PersistentFlags().BoolVarP(
		&verbose, "verbose", "v", false, "verbose output")
	rootCmd.AddCommand(infoCmd)
}
