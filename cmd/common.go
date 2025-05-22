package cmd

import "github.com/spf13/cobra"

func addUsernameFlags(c *cobra.Command, username *string, usage string) {
	c.PersistentFlags().StringVarP(username, "username", "u", "", usage)
	c.PersistentFlags().StringVarP(username, "user", "", "", "")
	_ = c.PersistentFlags().MarkHidden("user")
	c.PersistentFlags().StringVarP(username, "email", "", "", "")
	_ = c.PersistentFlags().MarkHidden("email")
}
