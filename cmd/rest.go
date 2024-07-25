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

func restClearImpersonateCmd() *cobra.Command {
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
			cfg := client.NewConfiguration()
			cfg.Servers[0] = client.ServerConfiguration{URL: "http://localhost:8080/ll-auth/api"}
			cfg.DefaultHeader["Authorization"] = "Bearer " + token
			cl := client.NewAPIClient(cfg)
			ctx := context.Background()
			r, _, err := cl.UserAuthAPI.ClearImpersonate(ctx).ClearImpersonateRequest(
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
	return c
}

func restImpersonateCmd() *cobra.Command {
	var token string
	var tenant string
	var role string
	var refreshToken string
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
			cfg := client.NewConfiguration()
			cfg.Servers[0] = client.ServerConfiguration{URL: "http://localhost:8080/ll-auth/api"}
			cfg.DefaultHeader["Authorization"] = "Bearer " + token
			cl := client.NewAPIClient(cfg)
			ctx := context.Background()
			r, _, err := cl.UserAuthAPI.Impersonate(ctx).ImpersonateRequest(
				client.ImpersonateRequest{
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
	return c
}

func restInviteCmd() *cobra.Command {
	var tenant string
	var username string
	var role string
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
			cfg := client.NewConfiguration()
			cfg.Servers[0] = client.ServerConfiguration{URL: "http://localhost:8080/ll-auth/api"}
			cfg.DefaultHeader["Authorization"] = "Bearer " + s.IdToken
			cl := client.NewAPIClient(cfg)
			ctx := context.Background()
			r, err := cl.UserAuthAPI.InviteToTenant(ctx).InviteRequest(
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
	return c
}

func restLoginCmd() *cobra.Command {
	var username string
	var password string
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
			cfg := client.NewConfiguration()
			cfg.Servers[0] = client.ServerConfiguration{URL: "http://localhost:8080/ll-auth/api"}
			cl := client.NewAPIClient(cfg)
			ctx := context.Background()
			r, _, err := cl.UserAuthAPI.Login(ctx).LoginRequest(client.LoginRequest{
				Username: username,
				Password: password,
			}).Execute()
			if err != nil {
				return err
			}
			s := MustSession()
			s.AccessToken = r.AccessToken
			s.IdToken = r.IdToken
			s.RefreshToken = r.RefreshToken
			if err = s.SaveToFile(); err != nil {
				return err
			}
			fmt.Printf("tokens stored successfully\n")
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&username, "username", "u", "", "Username to login with")
	c.PersistentFlags().StringVarP(
		&password, "password", "p", "", "Password to login with")
	return c
}

func restTenantsCmd() *cobra.Command {
	var token string
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
			cfg := client.NewConfiguration()
			cfg.Servers[0] = client.ServerConfiguration{URL: "http://localhost:8080/ll-auth/api"}
			cfg.DefaultHeader["Authorization"] = "Bearer " + token
			cl := client.NewAPIClient(cfg)
			ctx := context.Background()
			tenants, _, err := cl.UserAuthAPI.GetTenants(ctx).Execute()
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
	return c
}

func restUsersCmd() *cobra.Command {
	var tenant string
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
			cfg := client.NewConfiguration()
			cfg.Servers[0] = client.ServerConfiguration{URL: "http://localhost:8080/ll-auth/api"}
			cfg.DefaultHeader["Authorization"] = "Bearer " + s.IdToken
			cl := client.NewAPIClient(cfg)
			ctx := context.Background()
			o, _, err := cl.UserAuthAPI.GetUsersForTenant(ctx, tenant).Execute()
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
	c.AddCommand(restTenantsCmd())
	c.AddCommand(restUsersCmd())
	return c
}
