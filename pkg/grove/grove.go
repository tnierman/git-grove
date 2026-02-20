package grove

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tnierman/git-grove/pkg/git/local"
)

type Grove struct {
	repo *local.Repository
}

// Open creates a new grove
func Init() (*Grove, error) {
	// We can safely assume that this operation is either being executed A) directly within the
	// main worktree itself, or B) within a directory that has $GIT_COMMON_DIR set (either via
	// non-main worktree initialization or via grove itself). If this is not the main worktree or
	// an environment with $GIT_COMMON_DIR set, then it's invalid to perform grove operations anyway
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to determine current working directory: %w", err)
	}

	repo, err := local.NewRepository(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize git repo %q: %w", cwd, err)
	}

	g := &Grove{
		repo: repo,
	}

	return g, nil
}

// Root gives the absolute path of the root directory of the grove
func (g *Grove) Root() (string, error) {
	mainWorktree, err := g.repo.MainWorktree()
	if err != nil {
		return "", fmt.Errorf("failed to determine path to main worktree of repository: %w", err)
	}

	// The grove's root directory is always one directory level above the root of the main worktree
	return filepath.Clean(filepath.Join(mainWorktree, "..")), nil
}

// AddTree creates a new worktree at the given path relative to the grove's root, unless prefixed with /
//
// If the provided path contains a directory that does not exist, it will be created with mode 0700
func (g *Grove) AddTree(path string) error {
	if !strings.HasPrefix(path, "/") {
		// Absolute path not provided: construct absolute path of new worktree relative to grove root
		root, err := g.Root()
		if err != nil {
			return fmt.Errorf("failed to determine grove root: %w", err)
		}
		path = filepath.Join(root, path)
	}

	err := os.MkdirAll(path, 0o700)
	if err != nil {
		return fmt.Errorf("failed to create directory %q: %w", path, err)
	}

	err = g.repo.AddWorktree(path)
	if err != nil {
		return fmt.Errorf("failed to create worktree %q: %w", path, err)
	}

	return nil
}
