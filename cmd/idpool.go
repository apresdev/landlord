package cmd

import (
	"github.com/apresdev/landlord/internal/authx"
	"github.com/spf13/cobra"
)

func createListIDPoolCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the identity pools",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			ips, err := authx.GetIdentityPoolShortDescriptions(s.Ctx, s.CogId)
			if err != nil {
				return err
			}
			if len(ips) == 0 {
				cmd.Println("No identity pools found")
				return nil
			}
			for _, ip := range ips {
				cmd.Printf("%s (id=%s)\n", *ip.IdentityPoolName, *ip.IdentityPoolId)
			}
			return nil
		},
	}
}

func createIDPoolCmd() *cobra.Command {
	idpoolCmd := &cobra.Command{
		Use:   "idpool",
		Short: "Identity pool sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	idpoolCmd.AddCommand(createListIDPoolCmd())
	return idpoolCmd
}
