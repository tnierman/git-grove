package initalize

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tnierman/git-grove/pkg/git/remote"
)

const (
	defaultDirectoryPermissions = 0o0755

	groveInitTimeout = 60 * time.Second
)

var (
	createAllBranches bool
)

var Command = &cobra.Command{
	Use:   "init <repo> [<directory>]",
	Short: "Initialize new grove",
	Long: `Initialize a new grove with the provided repository.

A directory can optionally be supplied to indicate where the grove should be created; if none is provided
the grove is created in the current directory, with the same name as the repo.`,
	Example: `
Create a new grove "linux" in the current directory:

	grove init https://github.com/torvalds/linux.git

Run "cd linux/master" to enter the default worktree.

To create a grove in a specific directory:

	grove init https://github.com/torvalds/linux.git /tmp/linux

The grove will be created in the /tmp directory instead
	`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(_ *cobra.Command, args []string) error {
		var (
			// RangeArgs ensures there's at least one argument to this command
			repo = args[0]

			dir string
			err error
		)

		// Get dir from arguments, if provided, or default to repo name
		if len(args) > 1 {
			dir = args[1]
		} else {
			dir, err = nameOf(repo)
			if err != nil {
				return fmt.Errorf("failed to determine name of repository %q: %w", repo, err)
			}
		}

		err = NewGrove(repo, dir)
		if err != nil {
			return fmt.Errorf("failed to create new grove: %w", err)
		}

		return nil
	},
}

func init() {
	Command.Flags().BoolVarP(&createAllBranches, "all-branches", "b", false, "Create worktrees for all branches when initializing a new grove. By default, only the default brach has a worktree created when initializing new grove")
}

// NewGrove creates a grove for the given repo at the provided path.
//
// The path must be a directory, or an error is returned.
// Repo must be a valid URL to the repository (remote or local).
func NewGrove(repoURL, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), groveInitTimeout)
	defer cancel()

	repository, err := remote.NewRepository(repoURL)
	if err != nil {
		return fmt.Errorf("failed to connect to remote repository: %w", err)
	}

	branch, err := repository.DefaultBranch(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine default branch for repository %q: %w", repoURL, err)
	}

	// Validate that both the root of grove and default worktree dir are empty, or do not exist on init.
	// Because we want both to be empty or newly-created, perform the check in two steps
	err = newOrEmptyDir(path)
	if err != nil {
		return fmt.Errorf("directory %q is invalid: %w", path, err)
	}

	defaultWorktreePath := filepath.Join(path, branch)
	err = newOrEmptyDir(defaultWorktreePath)
	if err != nil {
		return fmt.Errorf("directory %q is invalid: %w", path, err)
	}

	// Finally, clone the repo into the default worktree location
	err = repository.Clone(defaultWorktreePath)
	if err != nil {
		return fmt.Errorf("failed to clone %q to %q: %w", repoURL, defaultWorktreePath, err)
	}

	return nil
}

// newOrEmptyDir validates that the provided path refers to an empty directory, or creates an empty directory at the given path if none exists.
//
// If the given path refers to a non-directory file or an existing, non-empty directory, an error is returned.
func newOrEmptyDir(path string) error {
	files, err := os.ReadDir(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Directory does not exist: create it and return
			err = os.MkdirAll(path, defaultDirectoryPermissions)
			if err != nil {
				return fmt.Errorf("failed to create directory %q: %w", path, err)
			}
			return nil
		}

		// Directory could not be opened
		return fmt.Errorf("failed to open directory %q: %w", path, err)
	}

	// Directory exists - validate that it's empty
	if len(files) > 0 {
		return fmt.Errorf("directory %q is not empty", path)
	}
	return nil
}

// nameOf parses a standard URL or local path for the provided git repo to determine its name
func nameOf(repo string) (string, error) {
	index := strings.LastIndex(repo, "/")
	if index < 0 {
		return "", fmt.Errorf("invalid repository path provided: expected at least one '/' character")
	}
	// iterate past '/' character
	index++
	// remove '*.git' suffix, if present
	return strings.TrimSuffix(repo[index:], ".git"), nil
}
