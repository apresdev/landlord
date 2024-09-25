package cmd

import (
	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/spf13/cobra"
)

func refreshCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the current session's tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			cs := appcore.Must()
			up, err := authx.GetUserPoolByNameOrID(cs.Ctx(), cs.CogIdp(), s.UserPool)
			if err != nil {
				return err
			}
			tokens, err := authx.RefreshTokenAuth(
				cs.Ctx(),
				cs.CogIdp(),
				cs.Validator(),
				s.GetValidationURL(),
				up,
				s.RefreshToken)
			if err != nil {
				return err
			}
			s.AccessToken = tokens.AccessToken
			s.IdToken = tokens.IdToken
			s.RefreshToken = tokens.RefreshToken
			if err = s.SaveToFile(); err != nil {
				return err
			}
			cmd.Println("refreshed tokens")
			return nil
		},
	}
	return c
}
