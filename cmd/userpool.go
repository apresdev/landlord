package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/spf13/cobra"
)

func createCreateUserPoolCmd() *cobra.Command {
	var userpool string
	var idpool string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a user pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userpool == "" {
				return errors.New("userpool must be set")
			}
			s := MustSession()
			up, err := authx.CreateUserPool(s.Ctx, s.CogIdp, s.CogId, userpool, idpool)
			if err != nil {
				return err
			}
			cmd.Printf("Created user pool %s (id=%s)\n", *up.Name, *up.Id)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&userpool, "name", "n", "", "Name of the user pool")
	c.PersistentFlags().StringVarP(
		&idpool, "idpool", "i", "", "Identity pool to associate with")
	return c
}

func createDeleteUserPoolCmd() *cobra.Command {
	var userpool string
	var all bool
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete a user pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userpool == "" && !all {
				return errors.New("--name must be set")
			}
			if userpool != "" && all {
				return errors.New("cannot specify both --name and --all")
			}
			s := MustSession()
			if all {
				err := authx.DeleteAllUserPools(s.Ctx, s.CogIdp)
				if err != nil {
					return err
				}
				cmd.Printf("Deleted all user pools\n")
			} else {
				up, err := authx.DeleteUserPool(s.Ctx, s.CogIdp, userpool)
				if err != nil {
					return err
				}
				cmd.Printf("Deleted user pool %s (id=%s)\n", *up.Name, *up.Id)
			}
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&userpool, "name", "n", "", "Name of the user pool")
	c.PersistentFlags().BoolVarP(
		&all, "all", "a", false, "Delete all user pools")
	return c
}

func createListUserPoolCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the user pools",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			userPools, err := authx.GetUserPoolDescriptions(s.Ctx, s.CogIdp)
			if err != nil {
				return err
			}
			for _, up := range userPools {
				cmd.Printf("%s (id=%s)\n", *up.Name, *up.Id)
			}
			return nil
		},
	}
}

func createUserPoolCmd() *cobra.Command {
	userpoolCmd := &cobra.Command{
		Use:   "userpool",
		Short: "Identity pool sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	userpoolCmd.AddCommand(createCreateUserPoolCmd())
	userpoolCmd.AddCommand(createDeleteUserPoolCmd())
	userpoolCmd.AddCommand(createListUserPoolCmd())
	return userpoolCmd
}
