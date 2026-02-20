package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tnierman/git-grove/cmd/add"
	"github.com/tnierman/git-grove/cmd/convert"
	"github.com/tnierman/git-grove/cmd/initialize"
)

// grove represents the base command when called without any subcommands
var grove = &cobra.Command{
	Use:   "grove",
	Short: "Manage git worktrees seamlessly",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

func init() {
	grove.AddCommand(add.Command)
	grove.AddCommand(convert.Command)
	grove.AddCommand(initalize.Command)
}

func Grove() error {
	err := grove.Execute()
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}
	return nil
}
