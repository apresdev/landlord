package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/tinytools"

	"github.com/skip2/go-qrcode"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/pkg/client"
	"github.com/spf13/cobra"
)

func apiClient(token, server string) (*client.APIClient, error) {
	cfg := client.NewConfiguration()
	if server == "" {
		server = "http://localhost:42001"
	}
	cfg.Servers[0] = client.ServerConfiguration{URL: server}
	if token != "" {
		cfg.DefaultHeader["Authorization"] = "Bearer " + token
	}
	cl := client.NewAPIClient(cfg)
	return cl, nil
}

func restAssociateOTP() *cobra.Command {
	var server string
	var token string
	c := &cobra.Command{
		Use:   "associate-otp",
		Short: "Associate one-time password provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			if token == "" {
				token = s.AccessToken
				if token == "" {
					return errors.New("no token found")
				}
			}
			cl, err := apiClient(token, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			r, _, err := cl.LandlordUserAPI.AssociateOTP(ctx).Execute()
			if err != nil {
				return err
			}
			cmd.Printf("OTP URI: %s\n", r.OtpURI)
			png, err := qrcode.Encode(r.OtpURI, qrcode.Medium, 256)
			if err != nil {
				return err
			}
			if err := os.WriteFile("otp.png", png, 0644); err != nil {
				return err
			}
			cmd.Printf("QR code saved to otp.png\n")
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&token, "token", "t", "", "Token to authorize with")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	return c
}

func restClearImpersonateCmd() *cobra.Command {
	var server string
	var token string
	var refreshToken string
	c := &cobra.Command{
		Use:   "clear-impersonate",
		Short: "Clear impersonation for the current user",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			if token == "" {
				token = s.AccessToken
				if token == "" {
					return errors.New("no token found")
				}
			}
			if refreshToken == "" {
				refreshToken = s.RefreshToken
				if refreshToken == "" {
					return errors.New("no refresh token found")
				}
			}
			cl, err := apiClient(token, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			r, _, err := cl.LandlordUserAPI.ClearImpersonation(ctx).ClearImpersonationRequest(
				client.ClearImpersonationRequest{RefreshToken: s.RefreshToken}).Execute()
			if err != nil {
				return err
			}
			s.AccessToken = r.AccessToken
			s.IdToken = r.IdToken
			if err = s.SaveToFile(); err != nil {
				return err
			}
			cmd.Printf("Impersonation cleared. New tokens stored.\n")
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&token, "token", "t", "", "Token to authorize with")
	c.PersistentFlags().StringVarP(
		&refreshToken, "refresh-token", "f", "", "Refresh token to use")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	return c
}

func restImpersonateRoleCmd() *cobra.Command {
	var token string
	var tenant string
	var role string
	var refreshToken string
	var server string
	c := &cobra.Command{
		Use:   "impersonate-role",
		Short: "Impersonate a role from a different tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant required")
			}
			if role == "" {
				return errors.New("--role required")
			}
			s := MustSession()
			if token == "" {
				token = s.AccessToken
				if token == "" {
					return errors.New("no token found")
				}
			}
			if refreshToken == "" {
				refreshToken = s.RefreshToken
				if refreshToken == "" {
					return errors.New("no refresh token found")
				}
			}
			cl, err := apiClient(token, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			r, _, err := cl.LandlordUserAPI.ImpersonateRole(ctx).ImpersonateRoleRequest(
				client.ImpersonateRoleRequest{
					Tenant:       tenant,
					Role:         role,
					RefreshToken: refreshToken,
				}).Execute()
			if err != nil {
				return err
			}
			s.AccessToken = r.AccessToken
			s.IdToken = r.IdToken
			if err = s.SaveToFile(); err != nil {
				return err
			}
			cmd.Printf("Impersonation successful. New tokens stored.\n")
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&token, "token", "t", "", "Token to authorize with")
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Tenant to impersonate")
	c.PersistentFlags().StringVarP(
		&role, "role", "r", "", "Role to impersonate")
	c.PersistentFlags().StringVarP(
		&refreshToken, "refresh-token", "f", "", "Refresh token to use")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	return c
}

func restImpersonateUserCmd() *cobra.Command {
	var token string
	var tenant string
	var username string
	var refreshToken string
	var server string
	c := &cobra.Command{
		Use:   "impersonate-user",
		Short: "Impersonate a user from a different tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant required")
			}
			if username == "" {
				return errors.New("--username required")
			}
			s := MustSession()
			if token == "" {
				token = s.AccessToken
				if token == "" {
					return errors.New("no token found")
				}
			}
			if refreshToken == "" {
				refreshToken = s.RefreshToken
				if refreshToken == "" {
					return errors.New("no refresh token found")
				}
			}
			cl, err := apiClient(token, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			r, _, err := cl.LandlordUserAPI.ImpersonateUser(ctx).ImpersonateUserRequest(
				client.ImpersonateUserRequest{
					Tenant:       tenant,
					Username:     username,
					RefreshToken: refreshToken,
				}).Execute()
			if err != nil {
				return err
			}
			s.AccessToken = r.AccessToken
			s.IdToken = r.IdToken
			if err = s.SaveToFile(); err != nil {
				return err
			}
			cmd.Printf("Impersonation successful. New tokens stored.\n")
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&token, "token", "t", "", "Token to authorize with")
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Tenant to impersonate")
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username to impersonate")
	c.PersistentFlags().StringVarP(
		&refreshToken, "refresh-token", "f", "", "Refresh token to use")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	return c
}

func restInviteCmd() *cobra.Command {
	var server string
	var tenant string
	var username string
	var role string
	var mfa string
	c := &cobra.Command{
		Use:   "invite",
		Short: "Send an invite to a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenant == "" {
				return errors.New("--tenant required")
			}
			if username == "" {
				return errors.New("--username required")
			}
			s := MustSession()
			cl, err := apiClient(s.AccessToken, server)
			if err != nil {
				return err
			}
			tenantID, err := uuid.Parse(tenant)
			if err != nil {
				return err
			}
			ctx := context.Background()
			r, err := cl.LandlordAdminAPI.InviteToTenant(ctx, tenantID.String()).InviteRequest(
				client.InviteRequest{
					Username:  username,
					Role:      role,
					MfaPolicy: client.MFAPolicy(mfa),
				}).Execute()
			if err != nil {
				b, err2 := io.ReadAll(r.Body)
				if err2 == nil {
					return fmt.Errorf("%w: error: %s", err, string(b))
				}
				return err
			}
			cmd.Printf("Invitation sent to: %s for %s\n", username, tenant)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Tenant to impersonate")
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username to invite")
	c.PersistentFlags().StringVarP(
		&role, "role", "r", multi.RoleGuest, "Role to impersonate")
	c.PersistentFlags().StringVarP(
		&mfa,
		"mfa",
		"M",
		multi.MFAPolicyRequired.String(),
		"MFA policy for user. One of 'none', 'required'")
	return c
}

func restLoginCmd() *cobra.Command {
	var username string
	var password string
	var server string
	c := &cobra.Command{
		Use:   "login",
		Short: "Login to the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				return errors.New("--username required")
			}
			if password == "" {
				return errors.New("--password required")
			}
			cl, err := apiClient("", server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			r, _, err := cl.LandlordUserAPI.Login(ctx).LoginRequest(client.LoginRequest{
				Username: username,
				Password: password,
			}).Execute()
			if err != nil {
				return err
			}
			if r.Status == client.LoginStatusSuccess {
				s := MustSession()
				s.AccessToken = r.AccessToken
				s.IdToken = r.IdToken
				s.RefreshToken = r.RefreshToken
				if err = s.SaveToFile(); err != nil {
					return err
				}
				cmd.Printf("Login success. Tokens stored.\n")
				return nil
			}
			cmd.Printf("login failed with status=%s\n", r.Status)
			if r.Session != "" {
				s := MustSession()
				s.ChallengeSession = r.Session
				if err = s.SaveToFile(); err != nil {
					return err
				}
				cmd.Printf("Challenge session stored.\n")
			}
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username to login with")
	c.PersistentFlags().StringVarP(
		&password, "password", "p", "", "Password to login with")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	return c
}

func restNewPassword() *cobra.Command {
	var username string
	var password string
	var session string
	var server string
	c := &cobra.Command{
		Use:   "new-password",
		Short: "Set a new password",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				return errors.New("--username required")
			}
			if password == "" {
				return errors.New("--password required")
			}
			if session == "" {
				return errors.New("--session required")
			}
			cl, err := apiClient("", server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			r, _, err := cl.LandlordUserAPI.SetNewPassword(ctx).SetNewPasswordRequest(
				client.SetNewPasswordRequest{
					Username: username,
					Password: password,
					Session:  session,
				}).Execute()
			if err != nil {
				return err
			}
			if r.Status == client.LoginStatusSuccess {
				s := MustSession()
				s.AccessToken = r.AccessToken
				s.IdToken = r.IdToken
				s.RefreshToken = r.RefreshToken
				if err = s.SaveToFile(); err != nil {
					return err
				}
				cmd.Printf("Login success. Tokens stored.\n")
			} else {
				return fmt.Errorf("unknown status: %s", r.Status)
			}
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username to login with")
	c.PersistentFlags().StringVarP(
		&password, "password", "p", "", "Password to login with")
	c.PersistentFlags().StringVarP(
		&session, "session", "", "", "Session token returned from login call")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	return c
}

func restRefreshCmd() *cobra.Command {
	var username string
	var refreshToken string
	var server string
	c := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the access and ID tokens given a refresh token",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			if username == "" {
				return errors.New("--username required")
			}
			if refreshToken == "" {
				refreshToken = s.RefreshToken
				if refreshToken == "" {
					return errors.New("--refresh-token required")
				}
			}
			cl, err := apiClient("", server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			r, _, err := cl.LandlordUserAPI.Refresh(ctx).
				RefreshRequest(client.RefreshRequest{
					Username:     username,
					RefreshToken: refreshToken,
				}).Execute()
			if err != nil {
				return err
			}
			if r.Status == client.LoginStatusSuccess {
				s := MustSession()
				s.AccessToken = r.AccessToken
				s.IdToken = r.IdToken
				if err = s.SaveToFile(); err != nil {
					return err
				}
				cmd.Printf("Login success. Tokens stored.\n")
				return nil
			}
			cmd.Printf("refresh failed with status=%s\n", r.Status)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username to login with")
	c.PersistentFlags().StringVarP(
		&refreshToken, "refresh-token", "r", "", "Refresh token")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	return c
}

func restSetRolesCmd() *cobra.Command {
	var token string
	var server string
	var tenant string
	var addRoles []string
	var removeRoles []string
	var addSubRoles []string
	var removeSubRoles []string
	c := &cobra.Command{
		Use:   "set-roles",
		Short: "Set the roles for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				s := MustSession()
				token = s.AccessToken
				if token == "" {
					return errors.New("--token must be set")
				}
			}
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			cl, err := apiClient(token, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			ten, _, err := cl.LandlordAdminAPI.RetrieveTenant(ctx, tenant).Execute()
			if err != nil {
				return err
			}

			var req client.TenantRolesRequest
			req.Roles = append(ten.Tenant.Roles, addRoles...)
			req.Roles = tinytools.RemoveAllFromSlice(req.Roles, removeRoles)
			req.SubTenantRoles = append(ten.Tenant.SubTenantRoles, addSubRoles...)
			req.SubTenantRoles = tinytools.RemoveAllFromSlice(req.SubTenantRoles, removeSubRoles)

			_, _, err = cl.LandlordAdminAPI.UpdateRolesForTenant(ctx, tenant).TenantRolesRequest(req).Execute()

			return err
		},
	}
	c.PersistentFlags().StringVarP(
		&token, "token", "t", "", "Token to authorize with")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Tenant to set roles for")
	c.PersistentFlags().StringSliceVarP(
		&addRoles, "add-roles", "a", []string{}, "Roles to add")
	c.PersistentFlags().StringSliceVarP(
		&removeRoles, "remove-roles", "r", []string{}, "Roles to remove")
	c.PersistentFlags().StringSliceVarP(
		&addSubRoles, "add-sub-roles", "A", []string{}, "Sub-roles to add")
	c.PersistentFlags().StringSliceVarP(
		&removeSubRoles, "remove-sub-roles", "R", []string{}, "Sub-roles to remove")
	return c
}

func restSubjectCmd() *cobra.Command {
	var server string
	c := &cobra.Command{
		Use:   "subject",
		Short: "Get details about the current subject",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			if s.AccessToken == "" {
				return errors.New("need to login")
			}
			cl, err := apiClient(s.AccessToken, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			sub, _, err := cl.LandlordUserAPI.RetrieveSubject(ctx).Execute()
			if err != nil {
				return err
			}
			cmd.Printf("Username: %s\n", sub.Username)
			cmd.Printf("User ID: %s\n", sub.UserID)
			cmd.Printf("Role: %s\n", sub.Role)
			cmd.Printf("Status: %s\n", sub.Status)
			cmd.Printf("Tenant ID: %s\n", sub.TenantID)
			cmd.Printf("Tenant Name: %s\n", sub.TenantName)
			if sub.Impersonating != nil && len(sub.Impersonating.TenantID) > 0 {
				cmd.Printf("Impersonating:\n")
				cmd.Printf("  Tenant ID: %s\n", sub.Impersonating.TenantID)
				cmd.Printf("  User ID: %s\n", sub.Impersonating.UserID)
				cmd.Printf("  Email: %s\n", sub.Impersonating.Email)
				cmd.Printf("  Role: %s\n", sub.Impersonating.Role)
			}
			if len(sub.TenantProfileFields) > 0 {
				cmd.Printf("Tenant Profile Fields:\n")
				table := profileFieldsTable(sub.TenantProfileFields)
				cmd.Println(table.String())
			}
			if len(sub.UserProfileFields) > 0 {
				cmd.Printf("User Profile Fields:\n")
				table := profileFieldsTable(sub.UserProfileFields)
				cmd.Println(table.String())
			}
			return err
		},
	}
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	return c
}

func profileFieldsTable(fields []client.ProfileFormatField) *Table {
	table := NewTable([]string{"Name", "Format", "Value"})
	for _, f := range fields {
		err := table.AddRow([]string{f.Name, f.Format, f.Value})
		if err != nil {
			zap.L().Error("coder fail: error adding row to table", zap.Error(err))
		}
	}
	return table
}

func restTenantsCmd() *cobra.Command {
	var token string
	var server string
	c := &cobra.Command{
		Use:   "tenants",
		Short: "List the tenants for the given token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				s := MustSession()
				token = s.AccessToken
				if token == "" {
					return errors.New("--token must be set")
				}
			}
			cl, err := apiClient(token, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			tenants, _, err := cl.LandlordAdminAPI.ListTenants(ctx).Execute()
			if err != nil {
				return err
			}
			for _, ten := range tenants.Tenants {
				cmd.Printf("> %s\n", ten.Name)
				cmd.Printf("    ID: %s\n", ten.Id)
				if ten.Description != "" {
					cmd.Printf("    Description: %s\n", ten.Description)
				}
			}
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&token, "token", "t", "", "Token to authorize with")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	return c
}

func restUsersCmd() *cobra.Command {
	var tenant string
	var server string
	c := &cobra.Command{
		Use:   "users",
		Short: "List the users for the given tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			if s.AccessToken == "" {
				return errors.New("need to login")
			}
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			cl, err := apiClient(s.AccessToken, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			o, _, err := cl.LandlordAdminAPI.ListUsersForTenant(ctx, tenant).Execute()
			if err != nil {
				return err
			}
			if len(o.Users) == 0 {
				cmd.Printf("No users for tenant\n")
				return nil
			}
			table := NewTable([]string{"Username", "Role", "Status", "CognitoID"})
			for _, user := range o.Users {
				err := table.AddRow([]string{user.Username, user.Role, string(user.Status), user.CognitoID})
				if err != nil {
					return err
				}
			}
			cmd.Println(table.String())
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Tenant to list users for")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	return c
}

func restValidateMFACodeCmd() *cobra.Command {
	var server string
	var username string
	var code string
	var session string
	c := &cobra.Command{
		Use:   "validate-mfa-code",
		Short: "Validate an MFA code",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			if username == "" {
				return errors.New("--username required")
			}
			if code == "" {
				return errors.New("--code required")
			}
			if session == "" {
				session = s.ChallengeSession
				if session == "" {
					return errors.New("--session required")
				}
			}
			cl, err := apiClient(s.AccessToken, server)
			if err != nil {
				return err
			}
			resp, _, err := cl.LandlordUserAPI.ValidateMFACode(context.Background()).ValidateMFACodeRequest(
				client.ValidateMFACodeRequest{
					Username: username,
					MfaCode:  code,
					Session:  session,
				}).Execute()
			if err != nil {
				return err
			}
			s.AccessToken = resp.AccessToken
			s.IdToken = resp.IdToken
			if err = s.SaveToFile(); err != nil {
				return err
			}
			cmd.Printf("Login success. Tokens stored.\n")
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username to validate")
	c.PersistentFlags().StringVarP(
		&code, "code", "c", "", "MFA code to validate")
	c.PersistentFlags().StringVarP(
		&session, "session", "", "", "Session token returned from login call")
	return c
}

func restVerifyOTPAssociation() *cobra.Command {
	var server string
	var tenant string
	var username string
	var code string
	c := &cobra.Command{
		Use:   "verify-otp-association",
		Short: "Verify the association of an OTP provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			if tenant == "" {
				return errors.New("--tenant required")
			}
			if username == "" {
				return errors.New("--username required")
			}
			if code == "" {
				return errors.New("--code required")
			}
			cl, err := apiClient(s.AccessToken, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			_, err = cl.LandlordUserAPI.VerifyOTPAssociation(ctx).VerifyOTPAssociationRequest(
				client.VerifyOTPAssociationRequest{
					Username: username,
					OtpCode:  code,
				}).Execute()
			if err != nil {
				return err
			}
			cmd.Printf("OTP association verified\n")
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Tenant to impersonate")
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username to validate")
	c.PersistentFlags().StringVarP(
		&code, "code", "c", "", "MFA code to validate")
	return c
}

func restCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "rest",
		Short: "Rest sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(restAssociateOTP())
	c.AddCommand(restClearImpersonateCmd())
	c.AddCommand(restImpersonateRoleCmd())
	c.AddCommand(restImpersonateUserCmd())
	c.AddCommand(restInviteCmd())
	c.AddCommand(restLoginCmd())
	c.AddCommand(restNewPassword())
	c.AddCommand(restRefreshCmd())
	c.AddCommand(restSetRolesCmd())
	c.AddCommand(restSubjectCmd())
	c.AddCommand(restTenantsCmd())
	c.AddCommand(restUsersCmd())
	c.AddCommand(restValidateMFACodeCmd())
	c.AddCommand(restVerifyOTPAssociation())
	return c
}
