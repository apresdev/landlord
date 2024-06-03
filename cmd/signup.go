package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/spf13/cobra"
)

func createConfirmSignUpCmd() *cobra.Command {
	var userpool string
	var email string
	var code string
	var admin bool
	c := &cobra.Command{
		Use:   "confirm",
		Short: "Confirm sign up",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userpool == "" {
				return errors.New("--userpool required")
			}
			if email == "" {
				return errors.New("--email required")
			}
			if !admin && code == "" {
				return errors.New("--code or --admin required")
			}
			if admin && code != "" {
				return errors.New("--code and --admin are mutually exclusive")
			}
			s := MustSession()
			if admin {
				err := authx.AdminConfirmSignUp(s.Ctx, s.CogIdp, userpool, email)
				if err != nil {
					return err
				}
				cmd.Println("Sign up confirmed by administrator")
				return nil
			} else {
				err := authx.ConfirmSignUp(s.Ctx, s.CogIdp, userpool, email, code)
				if err != nil {
					return err
				}
				cmd.Println("Sign up confirmed")
				return nil
			}
		},
	}
	c.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool to confirm")
	c.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email to confirm")
	c.PersistentFlags().StringVarP(
		&code, "code", "c", "", "Confirmation code")
	c.PersistentFlags().BoolVarP(
		&admin, "admin", "a", false, "Confirm as administrator")
	return c
}

func createNewSignUpCmd() *cobra.Command {
	var userpool string
	var email string
	var password string
	c := &cobra.Command{
		Use:   "new",
		Short: "Issues a new sign up",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userpool == "" {
				return errors.New("userpool must be set")
			}
			if email == "" {
				return errors.New("email must be set")
			}
			if password == "" {
				// TODO: generate password
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
	c.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool to sign up to")
	c.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email to sign up with")
	c.PersistentFlags().StringVarP(
		&password, "password", "p", "", "Password to sign up with")
	return c
}

func createSignUpCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "signup",
		Short: "Sign up sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(createConfirmSignUpCmd())
	c.AddCommand(createNewSignUpCmd())
	return c
}
