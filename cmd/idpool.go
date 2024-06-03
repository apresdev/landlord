package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/spf13/cobra"
)

func createAssociateIDPoolCmd() *cobra.Command {
	var idPool string
	var userPool string
	c := &cobra.Command{
		Use:   "assoc",
		Short: "Associate a user pool with an identity pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if idPool == "" {
				return errors.New("--idpool is required")
			}
			if userPool == "" {
				return errors.New("--userpool is required")
			}
			s := MustSession()
			err := authx.AssociateUserPool(s.Ctx, s.CogId, s.CogIdp, idPool, userPool)
			if err != nil {
				return err
			}
			cmd.Printf("User pool '%s' associated with identity pool '%s'\n",
				userPool, idPool)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&idPool, "idpool", "i", "", "Identity pool ID")
	c.PersistentFlags().StringVarP(
		&userPool, "userpool", "u", "", "User pool ID")
	return c
}

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
	c := &cobra.Command{
		Use:   "idpool",
		Short: "Identity pool sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(createAssociateIDPoolCmd())
	c.AddCommand(createListIDPoolCmd())
	c.AddCommand(createDeleteIDPoolCmd())
	return c
}
