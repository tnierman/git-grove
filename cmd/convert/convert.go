package convert

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tnierman/git-grove/pkg/git/local"
)

var Command = &cobra.Command{
	Use:   "convert <path>",
	Short: "Convert an existing git repository to a grove",
	Long:  "TODO",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		err := ToGrove(path)
		if err != nil {
			return fmt.Errorf("failed to convert %q to grove: %w", path, err)
		}
		return nil
	},
}

// TODO: this initial method sucks. Rather than move everything, recreate the original directory,
// then move everything back, we can just create the grove in the tmp dir, and move it once to the designated path,
// replacing the current directory
func ToGrove(path string) error {
	// Open repository and retrieve defaultBranch name before moving to tmp dir
	// Additionally, verifies that the provided path is a git directory before migrating anything
	repo, err := local.NewRepository(path)
	if err != nil {
		return err
	}
	defaultBranch, err := repo.DefaultBranch()
	if err != nil {
		return fmt.Errorf("failed to determine default branch for %q: %w", path, err)
	}

	// Migrate local repo to temporary directory
	tmp, err := os.MkdirTemp(os.TempDir(), "convert-grove-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer func() {
		// Cleanup: expect directory to be empty and remove it
		// Using os.Remove() instead of os.RemoveAll() will cause an error if the directory is not empty, which is informative, in this case
		tmpErr := os.Remove(tmp)
		if tmpErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to cleanup directory %q: %w", err)
		}
	}()

	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to determine absolute path of %q: %w", path, err)
	}

	current, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("failed to retrieve file info for %q: %w", abs, err)
	}

	tmpRelocationPath := filepath.Join(tmp, filepath.Base(abs))
	err = os.Rename(abs, tmpRelocationPath)
	if err != nil {
		return fmt.Errorf("failed to move %q to temporary directory %q: %w", abs, tmpRelocationPath, err)
	}

	// Setup

	// Recreate original directory following relocation
	err = os.MkdirAll(abs, current.Mode())
	if err != nil {
		return fmt.Errorf("failed to create directory %q: %w", abs, err)
	}

	defaultBranchPath := filepath.Join(abs, defaultBranch)
	err = os.Rename(tmpRelocationPath, defaultBranchPath)
	if err != nil {
		return fmt.Errorf("failed to move temporary directory %q to new grove at %q: %w", tmpRelocationPath, defaultBranchPath, err)
	}

	return nil
}
