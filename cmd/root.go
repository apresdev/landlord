package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "landlord",
		Short: "Manage tenants in AWS Cognito",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = cmd.Help()
			return nil
		},
		SilenceUsage: true,
	}
	rootCmd.AddCommand(createIDPoolCmd())
	rootCmd.AddCommand(createInfoCmd())
	rootCmd.AddCommand(createLoginCmd())
	rootCmd.AddCommand(createLogoutCmd())
	rootCmd.AddCommand(createRefreshCmd())
	rootCmd.AddCommand(createRoleCmd())
	rootCmd.AddCommand(createSignUpCmd())
	rootCmd.AddCommand(tenantCmd())
	rootCmd.AddCommand(createUserCmd())
	rootCmd.AddCommand(createUserPoolCmd())
	rootCmd.AddCommand(createVersionCmd())
	rootCmd.AddCommand(createServerCmd())

	return rootCmd
}

func Execute() {
	err := Root().Execute()
	if err != nil {
		os.Exit(1)
	}
}
