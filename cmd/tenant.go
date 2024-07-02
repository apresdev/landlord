package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/spf13/cobra"
)

func tenantApplyCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	c := &cobra.Command{
		Use:   "apply",
		Short: "Apply local settings to Cognito",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" && tenantName == "" {
				return errors.New("tenant ID or tenant name must be set")
			}
			h, err := authx.ParseTenantHandle(tenantID, tenantName)
			if err != nil {
				return err
			}
			cs := MustClients()
			tens := MustTenants(cs)
			ten, err := tens.ResolveTenant(h)
			if err != nil {
				return err
			}
			err = authx.ApplyTenant(cs.Ctx(), cs.CogIdp(), ten)
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

func tenantCreateCmd() *cobra.Command {
	var name string
	var fullName string
	var customAuthDomain string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			cs := MustClients()
			tens := MustTenants(cs)
			ten, err := tens.CreateTenant(
				cs.Ctx(), cs.IAM(), cs.CogIdp(), name, fullName, customAuthDomain)
			if err != nil {
				return err
			}
			cmd.Printf("Created tenant: %s\n", ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&name, "name", "n", "", "Name of the tenant to create")
	c.PersistentFlags().StringVarP(
		&fullName, "full-name", "f", "", "Full name of the tenant")
	c.PersistentFlags().StringVarP(
		&customAuthDomain,
		"custom-auth-domain",
		"",
		"",
		"Use this as the custom auth domain instead of the default")
	return c
}

func tenantDeleteCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	var all bool
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" && tenantName == "" && !all {
				return errors.New(
					"tenant ID (--id), tenant name (--name) or --all must be set")
			}
			if all && (tenantID != "" || tenantName != "") {
				return errors.New("cannot specify --all with --id or --name")
			}
			if all {
				tens := MustTenants(MustClients())
				err := tens.DeleteAll()
				if err != nil {
					return err
				}
				cmd.Println("Deleted all tenants")
				return nil
			} else {
				h, err := authx.ParseTenantHandle(tenantID, tenantName)
				if err != nil {
					return err
				}
				tens := MustTenants(MustClients())
				ten, err := tens.ResolveTenant(h)
				if err != nil {
					return err
				}
				err = tens.DeleteTenant(ten)
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

func tenantEmailDomainCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "email-domain",
		Short: "Email domain sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(tenantEmailDomainAddCmd())
	c.AddCommand(tenantEmailDomainRemoveCmd())
	return c
}

func tenantEmailDomainAddCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	var domain string
	var allowed bool
	c := &cobra.Command{
		Use:   "add",
		Short: "Add an email domain to a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" && tenantName == "" {
				return errors.New("tenant ID or tenant name must be set")
			}
			if domain == "" {
				return errors.New("--domain must be set")
			}
			h, err := authx.ParseTenantHandle(tenantID, tenantName)
			if err != nil {
				return err
			}
			cs := MustClients()
			tens := MustTenants(cs)
			ten, err := tens.ResolveTenant(h)
			if err != nil {
				return err
			}
			if allowed {
				err = authx.AddAllowedEmailDomain(tens, ten.ID, domain)
				if err != nil {
					return err
				}
			} else {
				err = authx.AddExclusiveEmailDomain(tens, ten.ID, domain)
				if err != nil {
					return err
				}
			}
			cmd.Printf("Added email domain %s to tenant %s\n", domain, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenantID, "id", "I", "", "ID of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&domain, "domain", "d", "", "Email domain to add to tenant")
	c.PersistentFlags().BoolVarP(
		&allowed, "allowed", "", false, "Add the email domain as allowed")
	return c
}

func tenantEmailDomainRemoveCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	var domain string
	var allowed bool
	c := &cobra.Command{
		Use:   "remove",
		Short: "Remove an email domain from a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" && tenantName == "" {
				return errors.New("tenant ID or tenant name must be set")
			}
			if domain == "" {
				return errors.New("--domain must be set")
			}
			h, err := authx.ParseTenantHandle(tenantID, tenantName)
			if err != nil {
				return err
			}
			cs := MustClients()
			tens := MustTenants(cs)
			ten, err := tens.ResolveTenant(h)
			if err != nil {
				return err
			}
			if allowed {
				err = authx.RemoveAllowedEmailDomain(tens, ten.ID, domain)
				if err != nil {
					return err
				}
			} else {
				err = authx.RemoveExclusiveEmailDomain(tens, ten.ID, domain)
				if err != nil {
					return err
				}
			}
			cmd.Printf("Removed email domain %s from tenant %s\n", domain, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenantID, "id", "I", "", "ID of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&domain, "domain", "d", "", "Email domain to add to tenant")
	c.PersistentFlags().BoolVarP(
		&allowed, "allowed", "", false, "Add the email domain as allowed")
	return c
}

func tenantInitCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "init",
		Short: "Initialize a tenant 'DB'",
		RunE: func(cmd *cobra.Command, args []string) error {
			cs := MustClients()
			tens, err := authx.InitTenants(
				cs.Ctx(), cs.Cfg(), cs.Sts(), cs.CogIdp())
			if err != nil {
				return err
			}
			if err := tens.Save(); err != nil {
				return err
			}
			return nil
		},
	}
	return c
}

func tenantInviteCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	var email string
	var role string
	c := &cobra.Command{
		Use:   "invite",
		Short: "Invite a user to a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" && tenantName == "" {
				return errors.New("tenant ID (--id) or tenant name (--name)")
			}
			if email == "" {
				return errors.New("email of user (--email) must be set")
			}
			h, err := authx.ParseTenantHandle(tenantID, tenantName)
			if err != nil {
				return err
			}
			cs := MustClients()
			tens := MustTenants(cs)
			ten, err := tens.ResolveTenant(h)
			if err != nil {
				return err
			}
			err = authx.SendInvite(cs.Ctx(), cs.SES(), cs.CogIdp(), tens, email, role)
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
	c.PersistentFlags().StringVarP(
		&role, "role", "r", authx.RoleGuest, "Role to assign to user")
	return c
}

func displayTenant(cmd *cobra.Command, ten *authx.Tenant) {
	if ten.FullName == "" {
		cmd.Printf("> %s\n", ten.Name)
	} else {
		cmd.Printf("> %s -- %s\n", ten.Name, ten.FullName)
	}
	cmd.Printf("    ID: %s\n", ten.ID.String())
	cmd.Printf("    Custom Auth Domain: %s\n", ten.CustomAuthDomain)
	cmd.Printf("    User Pool: %s (%s)\n", ten.UserPoolName, ten.UserPoolID)
	if len(ten.Users) > 0 {
		cmd.Printf("    Users:\n")
		for _, email := range ten.Users {
			cmd.Printf("    * %s\n", email)
		}
	} else {
		cmd.Printf("    No Users\n")
	}
	if len(ten.AllowedEmailDomains) > 0 {
		cmd.Printf("    Allowed Email Domains:\n")
		for _, domain := range ten.AllowedEmailDomains {
			cmd.Printf("    * %s\n", domain)
		}
	} else {
		cmd.Printf("    No Allowed Email Domains\n")
	}
	if len(ten.ExclusiveEmailDomains) > 0 {
		cmd.Printf("    Exclusive Email Domains:\n")
		for _, domain := range ten.ExclusiveEmailDomains {
			cmd.Printf("    * %s\n", domain)
		}
	} else {
		cmd.Printf("    No Exclusive Email Domains\n")
	}
}

func tenantListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the tenants",
		RunE: func(cmd *cobra.Command, args []string) error {
			tens := MustTenants(MustClients())
			for _, ten := range tens.Entries {
				displayTenant(cmd, ten)
			}
			return nil
		},
		Aliases: []string{"ls"},
	}
}

func tenantUserCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "user",
		Short: "User sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(tenantUserAddCmd())
	c.AddCommand(tenantUserRemoveCmd())
	return c
}

func tenantUserAddCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	var email string
	c := &cobra.Command{
		Use:   "add",
		Short: "Add an user to a tenant without an invite",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" && tenantName == "" {
				return errors.New("tenant ID or tenant name must be set")
			}
			if email == "" {
				return errors.New("--email must be set")
			}
			h, err := authx.ParseTenantHandle(tenantID, tenantName)
			if err != nil {
				return err
			}
			cs := MustClients()
			tens := MustTenants(cs)
			ten, err := tens.ResolveTenant(h)
			if err != nil {
				return err
			}
			err = tens.UpdateTenant(ten.ID, func(t *authx.Tenant) error {
				e, err := authx.ParseEmail(email)
				if err != nil {
					return err
				}
				if t.HasUser(e.Address()) {
					return errors.New("user already in tenant")
				}
				t.Users = append(t.Users, authx.User{
					Email:  e.Address(),
					Role:   authx.RoleGuest,
					Status: "disabled",
				})
				return nil
			})
			cmd.Printf("Added user %s to tenant %s\n", email, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenantID, "id", "I", "", "ID of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email of user to add to tenant")
	return c
}

func tenantUserRemoveCmd() *cobra.Command {
	var tenantID string
	var tenantName string
	var email string
	c := &cobra.Command{
		Use:   "remove",
		Short: "Remove a user from a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" && tenantName == "" {
				return errors.New("tenant ID or tenant name must be set")
			}
			if email == "" {
				return errors.New("--email must be set")
			}
			h, err := authx.ParseTenantHandle(tenantID, tenantName)
			if err != nil {
				return err
			}
			cs := MustClients()
			tens := MustTenants(cs)
			ten, err := tens.ResolveTenant(h)
			if err != nil {
				return err
			}
			err = authx.DeleteUserFromUserPool(
				cs.Ctx(), cs.CogIdp(), ten.UserPoolID, email)
			if err != nil {
				return err
			}
			err = tens.UpdateTenant(ten.ID, func(t *authx.Tenant) error {
				e, err := authx.ParseEmail(email)
				if err != nil {
					return err
				}
				idx := t.UserIdx(e.Address())
				if idx == -1 {
					return errors.New("user not in tenant")
				}
				t.Users = append(t.Users[:idx], t.Users[idx+1:]...)
				return nil
			})
			cmd.Printf("Removed user %s from tenant %s\n", email, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenantID, "id", "I", "", "ID of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&tenantName, "name", "n", "", "Name of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email of user to add to tenant")
	return c
}

func tenantCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "tenant",
		Short: "Tenant sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(tenantApplyCmd())
	c.AddCommand(tenantCreateCmd())
	c.AddCommand(tenantDeleteCmd())
	c.AddCommand(tenantEmailDomainCmd())
	c.AddCommand(tenantInitCmd())
	c.AddCommand(tenantInviteCmd())
	c.AddCommand(tenantListCmd())
	c.AddCommand(tenantUserCmd())
	return c
}
