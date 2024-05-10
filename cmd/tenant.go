package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/spf13/cobra"
)

func createAddUserToTenantCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	var email string
	c := &cobra.Command{
		Use:   "add-user",
		Short: "Add a user to a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" && tenantName == "" {
				return errors.New("tenant ID or tenant name must be set")
			}
			if email == "" {
				return errors.New("email of user must be set")
			}
			ten, err := authx.ResolveTenant(authx.TenantID(tenantID), tenantName)
			if err != nil {
				return err
			}
			err = authx.AddUserToTenant(ten, email)
			if err != nil {
				return err
			}
			cmd.Printf("Added user %s to tenant %s\n", email, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenantID, "id", "I", "", "ID of the tenant to add user to")
	c.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to add user to")
	c.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email of user to add to tenant")

	return c
}

func createApplyCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	c := &cobra.Command{
		Use:   "apply",
		Short: "Apply local settings to Cognito",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" && tenantName == "" {
				return errors.New("tenant ID or tenant name must be set")
			}
			ten, err := authx.ResolveTenant(authx.TenantID(tenantID), tenantName)
			if err != nil {
				return err
			}
			s := MustSession()
			err = authx.ApplyTenant(s.Ctx, s.CogIdp, ten)
			if err != nil {
				return err
			}
			cmd.Println("Applied settings")
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenantID, "id", "I", "", "ID of the tenant to apply")
	c.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to apply")

	return c
}

func createCreateTenantCmd() *cobra.Command {
	var userPool string
	var tenantName string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userPool == "" {
				return errors.New("user pool name must be set")
			}
			if tenantName == "" {
				return errors.New("tenant name must be set")
			}
			s := MustSession()
			_, err := authx.GetUserPoolByNameOrID(s.Ctx, s.CogIdp, userPool)
			if err != nil {
				return err
			}
			ten, err := authx.CreateTenant(s.IAM, userPool, tenantName)
			if err != nil {
				return err
			}
			cmd.Printf("Created tenant: %s\n", ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&userPool, "userpool", "U", "", "User pool to create tenant in")
	c.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to create")
	return c
}

func createDeleteTenantCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	var all bool
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" && tenantName == "" && !all {
				return errors.New("tenant ID or tenant name must be set")
			}
			if all && (tenantID != "" || tenantName != "") {
				return errors.New("cannot specify both --all and --id or --name")
			}
			if all {
				err := authx.AllTenants.DeleteAll()
				if err != nil {
					return err
				}
				cmd.Println("Deleted all tenants")
				return nil
			} else {
				ten, err := authx.ResolveTenant(authx.TenantID(tenantID), tenantName)
				if err != nil {
					return err
				}
				err = authx.DeleteTenant(ten)
				if err != nil {
					return err
				}
				cmd.Printf("Deleted tenant: %s\n", ten.Name)
				return nil
			}
		},
	}
	c.PersistentFlags().StringVarP(
		&tenantID, "id", "I", "", "ID of the tenant to delete")
	c.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to delete")
	c.PersistentFlags().BoolVarP(
		&all, "all", "a", false, "Delete all tenants")
	return c
}

func createListTenantCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the tenants",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, ten := range authx.AllTenants.Entries {
				cmd.Printf("> %s\n", ten.Name)
				cmd.Printf("    ID: %s\n", ten.ID)
				cmd.Printf("    Users:\n")
				for _, email := range ten.UserEmails {
					cmd.Printf("    * %s\n", email)
				}
			}
			return nil
		},
	}
}

func createTenantCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "tenant",
		Short: "Tenant sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(createAddUserToTenantCmd())
	c.AddCommand(createApplyCmd())
	c.AddCommand(createCreateTenantCmd())
	c.AddCommand(createDeleteTenantCmd())
	c.AddCommand(createListTenantCmd())
	return c
}
