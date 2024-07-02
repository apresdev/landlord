package cmd

import (
	"github.com/apresdev/landlord/internal/authx"
	"github.com/spf13/cobra"
)

func createRefreshCmd() *cobra.Command {
	refreshCmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the current session's tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			cs := MustClients()
			tokens, err := authx.RefreshAuth(
				cs.Ctx(), cs.CogIdp(), cs.Validator(), authx.RefreshAuthArgs{
					Region:       cs.Cfg().Region,
					UserPool:     s.UserPool,
					RefreshToken: s.RefreshToken,
				})
			if err != nil {
				return err
			}
			err = s.StoreTokens(tokens.AccessToken, tokens.IdToken, tokens.RefreshToken)
			if err != nil {
				return err
			}
			cmd.Println("refreshed tokens")
			return nil
		},
	}
	return refreshCmd
}
