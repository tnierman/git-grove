package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// grove represents the base command when called without any subcommands
var grove = &cobra.Command{
	Use:   "grove",
	Short: "Manage git worktrees seamlessly",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

func Grove() error {
	err := grove.Execute()
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}
	return nil
}
