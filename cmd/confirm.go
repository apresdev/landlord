package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/spf13/cobra"
)

func createConfirmCmd() *cobra.Command {
	var userpool string
	var email string
	var code string
	confirmSignUpCmd := &cobra.Command{
		Use:   "confirm",
		Short: "Confirm sign up",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userpool == "" {
				return errors.New("--userpool required")
			}
			if email == "" {
				return errors.New("--email required")
			}
			if code == "" {
				return errors.New("--code required")
			}
			s := MustSession()
			err := authx.ConfirmSignUp(s.Ctx, s.CogIdp, userpool, email, code)
			if err != nil {
				return err
			}
			cmd.Println("Sign up confirmed")
			return nil
		},
	}
	confirmSignUpCmd.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool to confirm")
	confirmSignUpCmd.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email to confirm")
	confirmSignUpCmd.PersistentFlags().StringVarP(
		&code, "code", "c", "", "Confirmation code")
	return confirmSignUpCmd
}
