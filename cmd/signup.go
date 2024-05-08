package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/spf13/cobra"
)

func createSignUpCmd() *cobra.Command {
	var userpool string
	var email string
	var password string
	signUpCmd := &cobra.Command{
		Use:   "signup",
		Short: "Sign up",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userpool == "" {
				return errors.New("userpool must be set")
			}
			if email == "" {
				return errors.New("email must be set")
			}
			if password == "" {
				return errors.New("password must be set")
			}
			s := MustSession()
			err := authx.SignUp(s.Ctx, s.CogIdp, userpool, email, password)
			if err != nil {
				return err
			}
			cmd.Printf("Successfully signed up '%s' -- check your email for confirmation code\n", email)
			return nil
		},
	}
	signUpCmd.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool to sign up to")
	signUpCmd.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email to sign up with")
	signUpCmd.PersistentFlags().StringVarP(
		&password, "password", "p", "", "Password to sign up with")
	return signUpCmd
}
