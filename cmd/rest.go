package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/pkg/client"
	"github.com/spf13/cobra"
)

func apiClient(token, server string) (*client.APIClient, error) {
	cfg := client.NewConfiguration()
	if server == "" {
		server = "http://localhost:8080"
	}
	cfg.Servers[0] = client.ServerConfiguration{URL: server}
	if token != "" {
		cfg.DefaultHeader["Authorization"] = "Bearer " + token
	}
	cl := client.NewAPIClient(cfg)
	return cl, nil
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
				token = s.IdToken
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
			r, _, err := cl.AuthAPI.ClearImpersonate(ctx).ClearImpersonateRequest(
				client.ClearImpersonateRequest{RefreshToken: s.RefreshToken}).Execute()
			if err != nil {
				return err
			}
			s.AccessToken = r.AccessToken
			s.IdToken = r.IdToken
			if err = s.SaveToFile(); err != nil {
				return err
			}
			fmt.Printf("new tokens stored successfully\n")
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

func restImpersonateCmd() *cobra.Command {
	var token string
	var tenant string
	var role string
	var refreshToken string
	var server string
	c := &cobra.Command{
		Use:   "impersonate",
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
				token = s.IdToken
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
			r, _, err := cl.AuthAPI.ImpersonateRole(ctx).ImpersonateRoleRequest(
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
			fmt.Printf("new tokens stored successfully\n")
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

func restInviteCmd() *cobra.Command {
	var tenant string
	var username string
	var role string
	var server string
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
			cl, err := apiClient(s.IdToken, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			r, err := cl.AuthAPI.InviteToTenant(ctx).InviteRequest(
				client.InviteRequest{
					Tenant:   tenant,
					Username: username,
					Role:     role,
				}).Execute()
			if err != nil {
				b, err2 := io.ReadAll(r.Body)
				if err2 == nil {
					return fmt.Errorf("%w: error: %s", err, string(b))
				}
				return err
			}
			fmt.Printf("invitation sent\n")
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Tenant to impersonate")
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username to invite")
	c.PersistentFlags().StringVarP(
		&role, "role", "r", multi.RoleGuest, "Role to impersonate")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
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
			r, _, err := cl.AuthAPI.Login(ctx).LoginRequest(client.LoginRequest{
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
				fmt.Printf("tokens stored successfully\n")
			} else if r.Status == client.LoginStatusNewPasswordRequired {
				fmt.Printf("new password required: session=%s\n", r.Session)
			} else if r.Status == client.LoginStatusPasswordResetRequired {
				fmt.Printf("password reset required: session=%s\n", r.Session)
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
			r, _, err := cl.AuthAPI.SetNewPassword(ctx).SetNewPasswordRequest(
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
				fmt.Printf("tokens stored successfully\n")
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

func restTenantsCmd() *cobra.Command {
	var token string
	var server string
	c := &cobra.Command{
		Use:   "tenants",
		Short: "List the tenants for the given token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				s := MustSession()
				token = s.IdToken
				if token == "" {
					return errors.New("--token must be set")
				}
			}
			cl, err := apiClient(token, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			tenants, _, err := cl.AuthAPI.GetTenants(ctx).Execute()
			if err != nil {
				return err
			}
			for _, ten := range tenants.Tenants {
				fmt.Printf("> %s\n", ten.Name)
				fmt.Printf("    ID: %s\n", ten.Id)
				if ten.Description != "" {
					fmt.Printf("    Description: %s\n", ten.Description)
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
			if s.IdToken == "" {
				return errors.New("need to login")
			}
			if tenant == "" {
				return errors.New("--tenant must be set")
			}
			cl, err := apiClient(s.IdToken, server)
			if err != nil {
				return err
			}
			ctx := context.Background()
			o, _, err := cl.AuthAPI.GetUsersForTenant(ctx, tenant).Execute()
			if err != nil {
				return err
			}
			if len(o.Users) > 0 {
				for _, user := range o.Users {
					fmt.Printf("%s -- %s -- %s\n", user.Username, user.Role, user.Status)
				}
			} else {
				fmt.Printf("No users for tenant\n")
			}
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&tenant, "tenant", "T", "", "Tenant to list users for")
	c.PersistentFlags().StringVarP(
		&server, "server", "s", "", "Server to connect to")
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
	c.AddCommand(restClearImpersonateCmd())
	c.AddCommand(restImpersonateCmd())
	c.AddCommand(restInviteCmd())
	c.AddCommand(restLoginCmd())
	c.AddCommand(restNewPassword())
	c.AddCommand(restTenantsCmd())
	c.AddCommand(restUsersCmd())
	return c
}
