package cmd

import (
	"fmt"

	"github.com/apresdev/landlord/internal/appenv"

	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Version command",
		RunE: func(cmd *cobra.Command, args []string) error {
			v := appenv.GetVersionInfo()
			fmt.Printf("Version    : %s\n", v.Version)
			fmt.Printf("Commit     : %s\n", v.Commit)
			fmt.Printf("Build time : %s\n", v.BuildTime)
			fmt.Printf("Go version : %s\n", v.GoVersion)
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(versionCmd())
}
