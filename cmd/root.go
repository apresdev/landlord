package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "landlord",
	Short: "Manage tenants in AWS Cognito",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd.Help()
		return nil
	},
}

func Root() *cobra.Command {
	return rootCmd
}

func Execute() {
	cmd := Root()
	cmd.SilenceUsage = true
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
