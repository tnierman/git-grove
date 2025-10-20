package initalize

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/go-git/go-git/v6/storage/memory"

	"github.com/spf13/cobra"
)

const (
	defaultDirectoryPermissions = 0o0755

	repoCloneTimeout = 60 * time.Second
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
		// RangeArgs ensures there's at least one argument to this command
		repo := args[0]

		dir := ""
		var err error
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

// NewGrove creates a grove for the given repo at the provided path.
//
// The path must be a directory, or an error is returned.
// Repo must be a valid URL to the repository (remote or local).
func NewGrove(repo, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), repoCloneTimeout)
	defer cancel()

	// Create root of grove
	err := newOrEmptyDir(path)
	if err != nil {
		return fmt.Errorf("directory %q is invalid: %w", path, err)
	}

	// Create default worktree location
	branch, err := defaultBranch(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to determine default branch for repository %q: %w", repo, err)
	}
	defaultWorktreePath := filepath.Join(path, branch)

	err = newOrEmptyDir(defaultWorktreePath)
	if err != nil {
		return fmt.Errorf("directory %q is invalid: %w", path, err)
	}

	auth, err := ssh.NewSSHAgentAuth("git")
	if err != nil {
		return fmt.Errorf("failed to create ssh agent for authentication: %w", err)
	}

	_, err = git.PlainClone(defaultWorktreePath, &git.CloneOptions{
		URL:  repo,
		Auth: auth,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository %q into %q: %w", repo, defaultWorktreePath, err)
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

// defaultBranch retrieves refs from the given repository and locates the target branch for HEAD
func defaultBranch(ctx context.Context, repo string) (string, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		URLs: []string{repo},
	})

	auth, err := ssh.NewSSHAgentAuth("git")
	if err != nil {
		return "", fmt.Errorf("failed to create ssh agent for authentication: %w", err)
	}

	refs, err := remote.ListContext(ctx, &git.ListOptions{
		Auth: auth,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list refs for repository %q: %w", repo, err)
	}

	for _, ref := range refs {
		if ref.Name() == "HEAD" {
			branch := ref.Target().Short()
			if branch == "" {
				return "", fmt.Errorf("HEAD ref for repository %q has missing target", repo)
			}
			return branch, nil
		}
	}

	return "", fmt.Errorf("no HEAD ref in repository %q", repo)
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
