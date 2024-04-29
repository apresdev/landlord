package cmd

import (
	"fmt"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/spf13/cobra"
)

func init() {
	var userpool string
	var email string
	var password string
	var loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Login to the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userpool == "" {
				return fmt.Errorf("userpool must be set")
			}
			if email == "" {
				return fmt.Errorf("email must be set")
			}
			if password == "" {
				return fmt.Errorf("password must be set")
			}
			err := authx.UserPasswordAuth(userpool, email, password)
			if err != nil {
				return err
			}
			fmt.Println("Logged in as: " + email)
			return nil
		},
	}
	loginCmd.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool to login to")
	loginCmd.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email to login with")
	loginCmd.PersistentFlags().StringVarP(
		&password, "password", "p", "", "Password to login with")
	rootCmd.AddCommand(loginCmd)
}
