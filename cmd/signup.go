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
	var signUpCmd = &cobra.Command{
		Use:   "signup",
		Short: "Sign up",
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
			err := authx.SignUp(userpool, email, password)
			if err != nil {
				return err
			}
			fmt.Printf("Successfully signed up '%s' -- check your email for confirmation code\n", email)
			return nil
		},
	}
	signUpCmd.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool to sign up to")
	signUpCmd.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email to sign up with")
	signUpCmd.PersistentFlags().StringVarP(
		&password, "password", "p", "", "Password to sign up with")
	rootCmd.AddCommand(signUpCmd)
}
