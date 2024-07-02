package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/spf13/cobra"
)

func createLoginCmd() *cobra.Command {
	var userpool string
	var email string
	var password string
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userpool == "" {
				return errors.New("--userpool required")
			}
			if email == "" {
				return errors.New("--email required")
			}
			if password == "" {
				return errors.New("--password required")
			}
			cs := MustClients()
			tokens, err := authx.UserPasswordAuth(
				cs.Ctx(), cs.CogIdp(), cs.Validator(), authx.UserPasswordAuthArgs{
					Region:   cs.Cfg().Region,
					UserPool: userpool,
					Email:    email,
					Password: password,
				})
			if err != nil {
				return err
			}
			s := MustSession()
			err = s.StoreIdentity(userpool, email)
			if err != nil {
				return err
			}
			err = s.StoreTokens(tokens.AccessToken, tokens.IdToken, tokens.RefreshToken)
			if err != nil {
				return err
			}
			s = MustReloadSession()
			cmd.Println("Logged in as: " + email)
			cmd.Printf("\n==== ID Token ====\n%s\n", s.IdToken)
			cmd.Printf("\n==== Access Token ====\n%s\n", s.AccessToken)
			return nil
		},
	}
	loginCmd.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool to login to")
	loginCmd.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email to login with")
	loginCmd.PersistentFlags().StringVarP(
		&password, "password", "p", "", "Password to login with")
	return loginCmd
}
