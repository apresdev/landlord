package cmd

import (
	"errors"

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

func createDeleteIDPoolCmd() *cobra.Command {
	var id string
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete an identity pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return errors.New("--id is required")
			}
			s := MustSession()
			err := authx.DeleteIdentityPool(s.Ctx, s.CogId, id)
			if err != nil {
				return err
			}
			cmd.Println("Identity pool deleted")
			return nil
		},
	}
	c.PersistentFlags().StringVarP(&id, "id", "i", "", "Identity pool ID")
	return c
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
	idpoolCmd.AddCommand(createDeleteIDPoolCmd())
	return idpoolCmd
}
