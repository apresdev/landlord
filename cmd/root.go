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
	rootCmd.AddCommand(createConfirmCmd())
	rootCmd.AddCommand(createIDPoolCmd())
	rootCmd.AddCommand(createInfoCmd())
	rootCmd.AddCommand(createLoginCmd())
	rootCmd.AddCommand(createLogoutCmd())
	rootCmd.AddCommand(createRefreshCmd())
	rootCmd.AddCommand(createSignUpCmd())
	rootCmd.AddCommand(createTenantCmd())
	rootCmd.AddCommand(createUserCmd())
	rootCmd.AddCommand(createUserPoolCmd())
	rootCmd.AddCommand(createVersionCmd())

	return rootCmd
}

func Execute() {
	err := Root().Execute()
	if err != nil {
		os.Exit(1)
	}
}
