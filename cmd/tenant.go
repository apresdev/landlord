package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/spf13/cobra"
)

func newAddUserToTenantCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	var email string
	addUserToTenantCmd := &cobra.Command{
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
	addUserToTenantCmd.PersistentFlags().StringVarP(
		&tenantID, "id", "I", "", "ID of the tenant to add user to")
	addUserToTenantCmd.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to add user to")
	addUserToTenantCmd.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email of user to add to tenant")

	return addUserToTenantCmd
}

func newApplyCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	applyToTenantCmd := &cobra.Command{
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
			err = authx.ApplyTenant(ten)
			if err != nil {
				return err
			}
			cmd.Println("Applied settings")
			return nil
		},
	}
	applyToTenantCmd.PersistentFlags().StringVarP(
		&tenantID, "id", "I", "", "ID of the tenant to add user to")
	applyToTenantCmd.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to add user to")

	return applyToTenantCmd
}

func newCreateTenantCmd() *cobra.Command {
	var userPoolName string
	var tenantName string
	createTenantCmd := &cobra.Command{
		Use:   "create",
		Short: "Delete a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userPoolName == "" {
				return errors.New("user pool name must be set")
			}
			if tenantName == "" {
				return errors.New("tenant name must be set")
			}
			_, err := authx.GetUserPoolByName(userPoolName)
			if err != nil {
				return err
			}
			ten, err := authx.CreateTenant(userPoolName, tenantName)
			if err != nil {
				return err
			}
			cmd.Printf("Created tenant: %s\n", ten.Name)
			return nil
		},
	}
	createTenantCmd.PersistentFlags().StringVarP(
		&userPoolName, "userpool", "U", "", "User pool to create tenant in")
	createTenantCmd.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to create")
	return createTenantCmd
}

func newDeleteTenantCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	var deleteTenantCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" && tenantName == "" {
				return errors.New("tenant ID or tenant name must be set")
			}
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
		},
	}
	deleteTenantCmd.PersistentFlags().StringVarP(
		&tenantID, "id", "I", "", "ID of the tenant to delete")
	deleteTenantCmd.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to delete")
	return deleteTenantCmd
}

func newListTenantCmd() *cobra.Command {
	var listTenantsCmd = &cobra.Command{
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
	return listTenantsCmd
}

func init() {
	var tenantCmd = &cobra.Command{
		Use:   "tenant",
		Short: "Tenant sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	tenantCmd.AddCommand(newAddUserToTenantCmd())
	tenantCmd.AddCommand(newApplyCmd())
	tenantCmd.AddCommand(newCreateTenantCmd())
	tenantCmd.AddCommand(newDeleteTenantCmd())
	tenantCmd.AddCommand(newListTenantCmd())
	rootCmd.AddCommand(tenantCmd)
}
