package cmd

import (
	"errors"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/spf13/cobra"
)

func createCreateUsersCmd() *cobra.Command {
	var userpool string
	var email string
	var c = &cobra.Command{
		Use:   "create",
		Short: "Create a new user in the given user pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userpool == "" {
				return errors.New("userpool must be set")
			}
			if email == "" {
				return errors.New("email must be set")
			}
			s := MustSession()
			err := authx.CreateUser(s.Ctx, s.CogIdp, userpool, email)
			if err != nil {
				return err
			}
			cmd.Printf("Created user: %s\n", email)
			return nil
		},
	}
	c.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool to create user in")
	c.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email of user to create")
	return c
}

func createDeleteUsersCmd() *cobra.Command {
	var userpool string
	var email string
	var deleteUserCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a user from the given user pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userpool == "" {
				return errors.New("userpool must be set")
			}
			if email == "" {
				return errors.New("email must be set")
			}
			s := MustSession()
			err := authx.DeleteUser(s.Ctx, s.CogIdp, userpool, email)
			if err != nil {
				return err
			}
			cmd.Printf("Deleted user: %s\n", email)
			return nil
		},
	}
	deleteUserCmd.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool to delete user from")
	deleteUserCmd.PersistentFlags().StringVarP(
		&email, "email", "e", "", "Email of user to delete")
	return deleteUserCmd
}

func createListUsersCmd() *cobra.Command {
	var userpool string
	var listUsersCmd = &cobra.Command{
		Use:   "list",
		Short: "List the users in the given user pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := MustSession()
			if userpool == "" {
				userpool = s.UserPool
			}
			if userpool == "" {
				return errors.New("userpool must be set")
			}
			users, err := authx.GetUsersInUserPool(s.Ctx, s.CogIdp, userpool)
			if err != nil {
				return err
			}
			for _, user := range users {
				cmd.Printf("> %s\n", authx.UserEmailAttribute(user))
				for _, attr := range user.Attributes {
					if *attr.Name == "email" {
						continue
					}
					cmd.Printf("    %s: %s\n", *attr.Name, *attr.Value)
				}
			}
			return nil
		},
	}
	listUsersCmd.PersistentFlags().StringVarP(
		&userpool, "userpool", "U", "", "User pool to list users from")
	return listUsersCmd
}

func createUserCmd() *cobra.Command {
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "User sub-command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	userCmd.AddCommand(createCreateUsersCmd())
	userCmd.AddCommand(createDeleteUsersCmd())
	userCmd.AddCommand(createListUsersCmd())
	return userCmd
}
