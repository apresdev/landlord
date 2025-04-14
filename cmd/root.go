package cmd

import (
	"context"
	"os"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"go.uber.org/zap"

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
	rootCmd.AddCommand(versionCmd())

	return rootCmd
}

func Execute() {
	err := Root().Execute()
	if err != nil {
		os.Exit(1)
	}
}

type SettingsKeyType string

const SettingsKey SettingsKeyType = "settings"

func appcoreMust(ctx context.Context) appcore.Clients {
	s := ctx.Value(SettingsKey)
	if s == nil {
		return appcore.Must()
	}
	settings, ok := s.(appenv.Settings)
	if !ok {
		zap.L().Fatal("invalid settings type", zap.Any("type", s))
	}
	return appcore.MustWithSettings(settings)
}
