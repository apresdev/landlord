package cmd

import (
	"github.com/apresdev/landlord/internal/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func serverCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "server",
		Short: "Run Landlord server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := server.Run(); err != nil {
				zap.L().Fatal("failed to run", zap.Error(err))
			}
			return nil
		},
	}
	return c
}
