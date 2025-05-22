package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/apresdev/landlord/internal/authx"

	"github.com/spf13/cobra"
)

func loginCmd() *cobra.Command {
	var userpool string
	var username string
	var password string
	c := &cobra.Command{
		Use:   "login",
		Short: "Login to a user pool. Stores tokens in session.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userpool == "" {
				return errors.New("--userpool required")
			}
			if username == "" {
				return errors.New("--username required")
			}
			if password == "" {
				return errors.New("--password required")
			}
			cs := appcoreMust(cmd.Context())
			up, err := authx.GetUserPoolByNameOrID(cs.Ctx(), cs.CogIdp(), userpool)
			if err != nil {
				return err
			}
			validationURL := jwx.ValidationURL(cs.Cfg().Region, aws.ToString(up.Id))
			tokens, err := authx.UserPasswordAuth(
				cs.Ctx(), cs.CogIdp(), cs.Validator(), validationURL, up, username, password)
			if err != nil {
				return err
			}
			s := MustSession()
			s.UserPool = userpool
			s.ValidationURL = validationURL.String()
			s.AccessToken = tokens.AccessToken.Raw
			s.IdToken = tokens.IdToken.Raw
			s.RefreshToken = tokens.RefreshToken
			if err = s.SaveToFile(); err != nil {
				return err
			}
			s = MustReloadSession()
			cmd.Println("Logged in as: " + username)
			cmd.Printf("\n==== ID Token ====\n%s\n", s.IdToken)
			cmd.Printf("\n==== Access Token ====\n%s\n", s.AccessToken)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool to login to")
	addUsernameFlags(c, &username, "Username to login with")
	c.PersistentFlags().StringVarP(
		&password, "password", "p", "", "Password to login with")

	return c
}
