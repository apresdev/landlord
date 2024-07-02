package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/spf13/cobra"
)

func createDeleteRoleCmd() *cobra.Command {
	var name string
	var prefix string
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete a role",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" && prefix == "" {
				return errors.New("--name or --prefix must be set")
			}
			if name != "" && prefix != "" {
				return errors.New("--name and --prefix cannot be set together")
			}
			cs := MustClients()
			if prefix != "" {
				roles, err := authx.DeleteIAMRoleByPrefix(cs.Ctx(), cs.IAM(), prefix)
				if err != nil {
					return err
				}
				for _, role := range roles {
					cmd.Printf("Deleted role: %s\n", role)
				}
			} else {
				err := authx.DeleteIAMRole(cs.Ctx(), cs.IAM(), name)
				if err != nil {
					return err
				}
				cmd.Printf("Deleted role: %s\n", name)
			}
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&name, "name", "n", "", "Name of the role")
	c.PersistentFlags().StringVarP(
		&prefix, "prefix", "p", "", "Prefix of the roles to delete")
	return c
}

func createListRoleCmd() *cobra.Command {
	var prefix string
	c := cobra.Command{
		Use:   "list",
		Short: "List the roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cs := MustClients()
			roles, err := authx.GetIAMRoles(cs.Ctx(), cs.IAM(), prefix)
			if err != nil {
				return err
			}
			for _, role := range roles {
				cmd.Printf("%s (arn=%s)\n", *role.RoleName, *role.Arn)
			}
			return nil
		},
		Aliases: []string{"ls"},
	}
	c.PersistentFlags().StringVarP(
		&prefix, "prefix", "p", "", "Prefix of the role name")
	return &c
}

func createRoleCmd() *cobra.Command {
	roleCmd := &cobra.Command{
		Use:   "role",
		Short: "Role sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	roleCmd.AddCommand(createDeleteRoleCmd())
	roleCmd.AddCommand(createListRoleCmd())
	return roleCmd
}
