package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/skip2/go-qrcode"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/google/uuid"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/spf13/cobra"
)

func tenantActivateCmd() *cobra.Command {
	var tenant string
	c := &cobra.Command{
		Use:   "activate",
		Short: "Activate a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			ten.State = multi.TenantStateActive
			err = db.StoreTenant(cs.Ctx(), cs.DDB(), ten)
			if err != nil {
				return err
			}
			cmd.Printf("Deactivated tenant: %s\n", ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant to delete")
	return c
}

func tenantApplyCmd() *cobra.Command {
	var tenant string
	c := &cobra.Command{
		Use:   "apply",
		Short: "Apply local settings to Cognito",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			err = authx.ApplyTenant(cs.Ctx(), cs.DDB(), cs.CogIdp(), ten)
			if err != nil {
				return err
			}
			cmd.Println("Applied settings")
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant to apply")
	return c
}

func tenantCreateCmd() *cobra.Command {
	var name string
	var description string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			cs := MustClients()
			ten, err := authx.CreateTenant(cs.Ctx(), cs.DDB(), cs.CogIdp(), name, description)
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
		&description, "description", "D", "", "Description of the tenant")
	return c
}

func tenantDeactivateCmd() *cobra.Command {
	var tenant string
	c := &cobra.Command{
		Use:   "deactivate",
		Short: "Deactivate a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			ten.State = multi.TenantStateInactive
			err = db.StoreTenant(cs.Ctx(), cs.DDB(), ten)
			if err != nil {
				return err
			}
			cmd.Printf("Deactivated tenant: %s\n", ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant to delete")
	return c
}

func tenantEmailDomainCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "email-domain",
		Short: "Username domain sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(tenantEmailDomainAddCmd())
	c.AddCommand(tenantEmailDomainRemoveCmd())
	return c
}

func tenantEmailDomainAddCmd() *cobra.Command {
	var tenant string
	var domain string
	var allowed bool
	c := &cobra.Command{
		Use:   "add",
		Short: "Add an email domain to a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			if domain == "" {
				return errors.New("--domain must be set")
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			dom, err := db.GetEmailDomain(cs.Ctx(), cs.DDB(), domain)
			if err != nil && !apperr.IsNotFound(err) {
				return err
			}
			if apperr.IsNotFound(err) {
				dom = &db.EmailDomain{Domain: domain, TenantIDs: []uuid.UUID{h.ID}}
				if !allowed {
					dom.Exclusive = true
				}
				if err := db.StoreEmailDomain(cs.Ctx(), cs.DDB(), dom); err != nil {
					return err
				}
			} else {
				if allowed {
					if dom.Exclusive {
						return errors.New("domain already configured as exclusive")
					}
					if idx := slices.Index(dom.TenantIDs, h.ID); idx == -1 {
						cmd.Printf("domain already configured as allowed for tenant\n")
						return nil
					}
					dom.TenantIDs = append(dom.TenantIDs, h.ID)
					if err := db.StoreEmailDomain(cs.Ctx(), cs.DDB(), dom); err != nil {
						return err
					}
				}
			}
			cmd.Printf("Added email domain %s to tenant %s\n", domain, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&domain, "domain", "d", "", "Username domain to add to tenant")
	c.PersistentFlags().BoolVarP(
		&allowed, "allowed", "", false, "Add the email domain as allowed")
	return c
}

func tenantEmailDomainRemoveCmd() *cobra.Command {
	var tenant string
	var domain string
	var allowed bool
	c := &cobra.Command{
		Use:   "remove",
		Short: "Remove an email domain from a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant name must be set")
			}
			if domain == "" {
				return errors.New("--domain must be set")
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			dom, err := db.GetEmailDomain(cs.Ctx(), cs.DDB(), domain)
			if err != nil {
				return err
			}
			idx := slices.Index(dom.TenantIDs, ten.ID)
			if idx == -1 {
				return errors.New("domain not configured for tenant")
			}
			dom.TenantIDs = append(dom.TenantIDs[:idx], dom.TenantIDs[idx+1:]...)
			if len(dom.TenantIDs) == 0 {
				err = db.DeleteEmailDomain(cs.Ctx(), cs.DDB(), domain)
				if err != nil {
					return err
				}
			} else {
				err = db.StoreEmailDomain(cs.Ctx(), cs.DDB(), dom)
				if err != nil {
					return err
				}
			}
			cmd.Printf("Removed email domain %s from tenant %s\n", domain, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&domain, "domain", "d", "", "Username domain to add to tenant")
	c.PersistentFlags().BoolVarP(
		&allowed, "allowed", "", false, "Add the email domain as allowed")
	return c
}

func tenantInitCmd() *cobra.Command {
	var reinit bool
	c := &cobra.Command{
		Use:   "init",
		Short: "Initialize the tenant database",
		RunE: func(cmd *cobra.Command, args []string) error {
			cs := MustClients()
			if reinit {
				if err := db.DeleteTables(cs.Ctx(), cs.DDB()); err != nil {
					return err
				}
			}
			err := db.Init(cs.Ctx(), cs.DDB())
			if err != nil {
				return err
			}
			return authx.CreateRootTenant(cs.Ctx(), cs.DDB(), cs.CogIdp())
		},
	}
	c.PersistentFlags().BoolVarP(
		&reinit, "reinit", "", false, "Reinitialize the tenant database")
	return c
}

func tenantInviteCmd() *cobra.Command {
	var tenant string
	var username string
	var role string
	var mfa string
	c := &cobra.Command{
		Use:   "invite",
		Short: "Invite a user to a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			if username == "" {
				return errors.New("--username must be set")
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			mfaPol, err := multi.ParseMFAPolicy(mfa)
			if err != nil {
				return err
			}
			err = authx.SendInvite(
				cs.Ctx(), cs.DDB(), cs.CogIdp(), ten, username, role, mfaPol)
			if err != nil {
				return err
			}
			cmd.Printf("Invited user %s to tenant %s\n", username, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant to add user to")
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username of user to add to tenant")
	c.PersistentFlags().StringVarP(
		&role, "role", "r", multi.RoleGuest, "Role to assign to user")
	c.PersistentFlags().StringVarP(
		&mfa, "mfa", "M", "required", "MFA policy for user. One of 'none', 'required'")
	return c
}

func displayTenant(cmd *cobra.Command, ten *multi.Tenant) {
	if ten.Description == "" {
		cmd.Printf("> %s\n", ten.Name)
	} else {
		cmd.Printf("> %s -- %s\n", ten.Name, ten.Description)
	}
	cmd.Printf("    ID: %s\n", ten.ID.String())
	cmd.Printf("    State: %s\n", ten.State)
	cmd.Printf("    User Pool: %s (%s)\n", ten.UserPoolName, ten.UserPoolID)
	cmd.Printf("    Roles:\n")
	for _, role := range ten.Roles {
		cmd.Printf("    * %s\n", role)
	}
	users, err := db.GetUsersForTenant(cmd.Context(), MustClients().DDB(), ten.ID)
	if err != nil {
		zap.L().Error("Error getting users for tenant", zap.Error(err))
	} else {
		if len(users) > 0 {
			cmd.Printf("    Users:\n")
			for _, user := range users {
				cmd.Printf("    * %s\n", user.Username)
			}
		} else {
			cmd.Printf("    No Users\n")
		}
	}
}

func tenantListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the tenants",
		RunE: func(cmd *cobra.Command, args []string) error {
			cs := MustClients()
			tens, err := db.GetAllTenants(cs.Ctx(), cs.DDB())
			if err != nil {
				return err
			}
			if len(tens) == 0 {
				cmd.Println("No tenants")
				return nil
			}
			for _, ten := range tens {
				displayTenant(cmd, ten)
			}
			return nil
		},
		Aliases: []string{"ls"},
	}
}

func tenantRoleCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "role",
		Short: "Role sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(tenantRoleCreateCmd())
	c.AddCommand(tenantRoleDeleteCmd())
	return c
}

func tenantRoleCreateCmd() *cobra.Command {
	var tenant string
	var role string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a role on a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			if role == "" {
				return errors.New("--role must be set")
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			if err = ten.AddRole(role); err != nil {
				return err
			}
			groupName := ten.GroupName(role)
			_, err = cs.CogIdp().CreateGroup(cs.Ctx(),
				&cognitoidentityprovider.CreateGroupInput{
					GroupName:  aws.String(groupName),
					UserPoolId: aws.String(ten.UserPoolID),
				})
			if err != nil {
				return err
			}
			if err = db.StoreTenant(cs.Ctx(), cs.DDB(), ten); err != nil {
				return err
			}
			cmd.Printf("Create role %s for tenant %s\n", role, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant to create a role for")
	c.PersistentFlags().StringVarP(
		&role, "role", "", "", "Role to create")
	return c
}

func tenantRoleDeleteCmd() *cobra.Command {
	var tenant string
	var role string
	c := &cobra.Command{
		Use:   "remove",
		Short: "Remove a role from a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			if role == "" {
				return errors.New("--role must be set")
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			idx := slices.Index(ten.Roles, role)
			if idx == -1 {
				return errors.New("role not found")
			}
			users, err := db.GetUsersForTenant(cs.Ctx(), cs.DDB(), ten.ID)
			if err != nil {
				return err
			}
			for _, user := range users {
				if user.Role == role {
					return errors.New("role still in use by user")
				}
			}
			if _, err := cs.CogIdp().DeleteGroup(cs.Ctx(),
				&cognitoidentityprovider.DeleteGroupInput{
					GroupName:  aws.String(ten.GroupName(role)),
					UserPoolId: aws.String(ten.UserPoolID),
				}); err != nil {
				return err
			}
			ten.Roles = append(ten.Roles[:idx], ten.Roles[idx+1:]...)
			if err = db.StoreTenant(cs.Ctx(), cs.DDB(), ten); err != nil {
				return err
			}
			cmd.Printf("Removed role %s from tenant %s\n", role, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&role, "role", "", "", "Role to add to tenant")
	return c
}

func tenantUserCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "user",
		Short: "User sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(tenantUserAssociateOTP())
	c.AddCommand(tenantUserListEvents())
	c.AddCommand(tenantUserRemoveCmd())
	c.AddCommand(tenantUserRoleCmd())
	c.AddCommand(tenantUserVerifyOTP())
	return c
}

func tenantUserAssociateOTP() *cobra.Command {
	var username string
	c := &cobra.Command{
		Use:   "associate-otp",
		Short: "Associate a OTP provider for the user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				return errors.New("--username must be set")
			}
			s := MustSession()
			if s.AccessToken == "" {
				return errors.New("no access token")
			}
			e, err := authx.ParseEmail(username)
			if err != nil {
				return err
			}
			cs := MustClients()
			result, err := authx.AssociateSoftwareToken(
				cs.Ctx(), cs.CogIdp(), s.AccessToken, e.Address())
			if err != nil {
				return err
			}
			png, err := qrcode.Encode(result.OtpAuthURI, qrcode.Medium, 256)
			if err != nil {
				return err
			}
			if err := os.WriteFile("otp.png", png, 0644); err != nil {
				return err
			}
			cmd.Printf("QR code saved to otp.png\n")
			cmd.Printf("OTP URI: %s\n", result.OtpAuthURI)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username of user to add to tenant")
	return c
}

func tenantUserListEvents() *cobra.Command {
	var tenant string
	var username string
	var results int32
	c := &cobra.Command{
		Use:   "list-events",
		Short: "List events for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			if username == "" {
				return errors.New("--username must be set")
			}
			s := MustSession()
			e, err := authx.ParseEmail(username)
			if err != nil {
				return err
			}
			cs := MustClients()
			result, err := cs.CogIdp().AdminListUserAuthEvents(cs.Ctx(),
				&cognitoidentityprovider.AdminListUserAuthEventsInput{
					UserPoolId: aws.String(s.UserPool),
					Username:   aws.String(e.Address()),
					MaxResults: aws.Int32(results),
				})
			if err != nil {
				return err
			}
			for _, event := range result.AuthEvents {
				risk := event.EventRisk
				var line strings.Builder
				line.WriteString(fmt.Sprintf(
					"%s | %s %s",
					aws.ToTime(event.CreationDate).Format("2006-01-02 15:04:05"),
					event.EventType,
					event.EventResponse))
				if risk != nil {
					if risk.RiskDecision != types.RiskDecisionTypeNoRisk {
						line.WriteString(fmt.Sprintf(
							" | %s risk: %s",
							risk.RiskLevel,
							risk.RiskDecision))
					}
					if aws.ToBool(risk.CompromisedCredentialsDetected) {
						line.WriteString(" | Compromised or trivial credentials")
					}
				}
				cmd.Println(line.String())
			}
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Tenant of the user")
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "List events for this username")
	c.PersistentFlags().Int32VarP(
		&results, "results", "r", 10, "Number of results to return")
	return c
}

func tenantUserVerifyOTP() *cobra.Command {
	var tenant string
	var username string
	var code string
	c := &cobra.Command{
		Use:   "verify-otp",
		Short: "Verify OTP code for user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			if username == "" {
				return errors.New("--username must be set")
			}
			if code == "" {
				return errors.New("--code must be set")
			}
			s := MustSession()
			e, err := authx.ParseEmail(username)
			if err != nil {
				return err
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			err = authx.VerifySoftwareToken(
				cs.Ctx(), cs.CogIdp(), ten.UserPoolID, s.AccessToken, e.Address(), code)
			if err != nil {
				return err
			}
			cmd.Printf("Verified OTP code for user %s in tenant %s\n", e.Address(), ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username of user to add to tenant")
	c.PersistentFlags().StringVarP(
		&code, "code", "", "", "MFA one-time code")
	return c
}

func tenantUserRemoveCmd() *cobra.Command {
	var tenant string
	var username string
	c := &cobra.Command{
		Use:   "remove",
		Short: "Remove a user from a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			if username == "" {
				return errors.New("--username must be set")
			}
			e, err := authx.ParseEmail(username)
			if err != nil {
				return err
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			err = authx.DeleteUserFromUserPool(
				cs.Ctx(), cs.CogIdp(), ten.UserPoolID, e.Address())
			if err != nil {
				zap.L().Info("Error deleting user from user pool", zap.Error(err))
			}
			if err = db.DeleteUser(cs.Ctx(), cs.DDB(), ten.ID, e.Address()); err != nil {
				return err
			}
			cmd.Printf("Removed user %s from tenant %s\n", e.Address(), ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant to add email domain to")
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username of user to add to tenant")
	return c
}

func tenantUserRoleCmd() *cobra.Command {
	var tenant string
	var username string
	var role string
	c := &cobra.Command{
		Use:   "role",
		Short: "Set the role for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			if username == "" {
				return errors.New("--username must be set")
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			if !ten.HasRole(role) {
				return fmt.Errorf("role '%s' does not exist, pick one of: %s\n",
					role, strings.Join(ten.Roles, ", "))
			}
			user, err := db.GetUser(cs.Ctx(), cs.DDB(), username)
			if err != nil {
				return err
			}
			user.Role = role
			if err = authx.SetCognitoGroup(cs.Ctx(), cs.CogIdp(), ten, user); err != nil {
				return err
			}
			if err = db.StoreUser(cs.Ctx(), cs.DDB(), user); err != nil {
				return err
			}
			cmd.Printf("Set role %s for user %s in tenant %s\n", role, username, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant")
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username to assign role to")
	c.PersistentFlags().StringVarP(
		&role, "role", "r", "", "Role to assign to user")
	return c
}

func tenantUserPoolCmd() *cobra.Command {
	var tenant string
	var userpool string
	c := &cobra.Command{
		Use:   "userpool",
		Short: "Set the userpool for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			if userpool == "" {
				return errors.New("--userpool must be set")
			}
			h, err := multi.ParseTenantHandle(tenant)
			if err != nil {
				return err
			}
			cs := MustClients()
			up, err := authx.GetUserPoolByNameOrID(cs.Ctx(), cs.CogIdp(), userpool)
			if err != nil {
				return err
			}
			// Ensure that there is an appropriately configured client
			_, err = authx.GetUserPoolClient(cs.Ctx(), cs.CogIdp(), aws.ToString(up.Id))
			if err != nil {
				return err
			}
			ten, err := db.ResolveTenant(cs.Ctx(), cs.DDB(), h)
			if err != nil {
				return err
			}
			ten.UserPoolID = aws.ToString(up.Id)
			ten.UserPoolName = aws.ToString(up.Name)
			if err = db.StoreTenant(cs.Ctx(), cs.DDB(), ten); err != nil {
				return err
			}
			cmd.Printf("Set userpool %s (%s) in tenant %s\n",
				ten.UserPoolID, ten.UserPoolID, ten.Name)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Name/ID of the tenant")
	c.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool set for tenant")
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
	c.AddCommand(tenantActivateCmd())
	c.AddCommand(tenantApplyCmd())
	c.AddCommand(tenantCreateCmd())
	c.AddCommand(tenantDeactivateCmd())
	c.AddCommand(tenantEmailDomainCmd())
	c.AddCommand(tenantInitCmd())
	c.AddCommand(tenantInviteCmd())
	c.AddCommand(tenantListCmd())
	c.AddCommand(tenantRoleCmd())
	c.AddCommand(tenantUserCmd())
	c.AddCommand(tenantUserPoolCmd())
	return c
}
