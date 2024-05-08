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
			tokens, err := authx.RefreshAuth(
				s.Ctx, s.CogIdp, s.Validator, authx.RefreshAuthArgs{
					Region:       s.GetRegion(),
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
