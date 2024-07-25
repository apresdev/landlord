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

	rootCmd.AddCommand(iamCmd())
	rootCmd.AddCommand(infoCmd())
	rootCmd.AddCommand(loginCmd())
	rootCmd.AddCommand(logoutCmd())
	rootCmd.AddCommand(refreshCmd())
	rootCmd.AddCommand(restCmd())
	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(tenantCmd())
	rootCmd.AddCommand(userPoolCmd())
	rootCmd.AddCommand(versionCmd())

	return rootCmd
}

func Execute() {
	err := Root().Execute()
	if err != nil {
		os.Exit(1)
	}
}
