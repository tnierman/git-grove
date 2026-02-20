package add

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tnierman/git-grove/pkg/grove"
)

var Command = &cobra.Command{
	Use:   "add <tree>",
	Short: "Add a new tree to the grove",
	Long: `Adds a new tree to the grove.

The new worktree is created at the given path relative to the grove's root, unless prefixed by '/' - in which case, an absolute path is assumed.

In all cases, any subdirectory which does not already exist will be created with bit mask 0x700`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		// cobra ExactArgs guarantees exactly 1 argument to this command
		path := args[0]
		err := NewTree(path)
		if err != nil {
			return err
		}
		return nil
	},
}

func NewTree(path string) error {
	grove, err := grove.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize grove: %w", err)
	}

	err = grove.AddTree(path)
	if err != nil {
		return fmt.Errorf("failed to add tree %q: %w", path, err)
	}
	return nil
}
