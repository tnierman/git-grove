/*
local defines logic to perform common git operations against a local clone of a git repo
*/
package local

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v6/osfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/storage/filesystem"
	"github.com/go-git/go-git/v6/x/plumbing/worktree"
)

const (
	// GitStorePath refers to the name of the file or directory at the root of each worktree.
	//
	// For the main worktree, this is the .git/ directory that holds the repository data.
	// For linked worktrees, this is the .git txt file which references the path of
	// the main worktree
	GitStorePath = ".git"

	// GitFilePrefix refers to the standard prefix that must be present in any linked worktree's
	// .git txt file
	GitFilePrefix = "gitdir:"
)

type Repository struct {
	// initPath is the filepath the repository was opened from
	initPath string
	// repo is git repository on the local disk
	repo *git.Repository
}

// NewRepository opens the repository at the given path. The provided path does not need to be the root of the repository
func NewRepository(path string) (*Repository, error) {
	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read git repository %q (is %q a git repo or grove environment?): %w", path, path, err)
	}
	r := &Repository{
		initPath: path,
		repo:     repo,
	}
	return r, nil
}

func (r *Repository) DefaultBranch() (string, error) {
	return "main", nil
}

func (r *Repository) MainWorktree() (string, error) {
	CurrentWorktreePath, err := r.CurrentWorktree()
	if err != nil {
		return "", fmt.Errorf("failed to determine path of current worktree: %w", err)
	}
	gitPath := GitPath(CurrentWorktreePath)

	// Determine if current worktree has a .git/ directory - if so,
	// the current worktree is the main worktree for the repo. If not,
	// we're in a linked worktree, so we'll have to parse out the .git
	// txt file to find the location of the main worktree
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %q: %w", CurrentWorktreePath, err)
	}
	if info.IsDir() {
		return CurrentWorktreePath, nil
	}

	// We're in a linked worktree - find the main worktree's absolute path
	mainWorktreePath, err := r.readGitFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("failed to determine the path to the main worktree from %q: %w", gitPath, err)
	}

	// Construct the absolute path if the returned path is relative
	if !filepath.IsAbs(mainWorktreePath) {
		mainWorktreePath, err = filepath.Abs(filepath.Join(CurrentWorktreePath, mainWorktreePath))
		if err != nil {
			return "", fmt.Errorf("failed to convert relative path to main worktree %q to absolute path: %w", mainWorktreePath, err)
		}
	}

	// The returned path typically points to our linked worktree's location within the .git/ directory of the main worktree.
	// In order to find the root of the main worktree, we must work backwards through the path until we reach the directory
	// directly above the lowest-level .git/ entry
	tokens := strings.Split(mainWorktreePath, GitStorePath)
	if len(tokens) == 1 {
		return tokens[0], nil
	}

	cleanedPath := strings.Join(tokens[:len(tokens)-1], GitStorePath)
	return cleanedPath, nil
}

// readGitFile opens the .git txt file at the provided path and parses the content.
//
// It expects the first line to start with the prefix 'gitdir:' followed by an absolute
// or local path. Additional content is ignored. An error is returned if the file does
// not contain the expected prefix, or if the path provided does not evaluate as a relative
// or absolute path. If no error is returned, a non-empty path is always returned
func (r *Repository) readGitFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open %q: %w", path, err)
	}
	defer func() {
		closeErr := file.Close()
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close file %q: %v", path, err)
		}
	}()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to read %q: %w", path, err)
	}
	line := scanner.Text()

	// Ensure proper formatting
	if !strings.HasPrefix(line, GitFilePrefix) {
		return "", fmt.Errorf(".git file %q has improper format: missing prefix %q on first line", path, GitFilePrefix)
	}

	// Remove 'gitdir:' prefix from line before returning
	linkedPath := strings.TrimPrefix(line, GitFilePrefix)
	linkedPath = strings.TrimSpace(linkedPath)
	if (!filepath.IsAbs(linkedPath) && !filepath.IsLocal(linkedPath)) || linkedPath == "" {
		return "", fmt.Errorf(".git file %q has improper format: invalid path %q specified", path, linkedPath)
	}
	return linkedPath, nil
}

func (r *Repository) CurrentWorktree() (string, error) {
	wt, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to determine current worktree path of %q (is this a bare repository?): %w", r.initPath, err)
	}

	return wt.Filesystem.Root(), nil
}

// AddWorktree creates a new worktree at the provided path named after the last element in the given path
//
// If the given path does not already exist as an empty directory in the local filesystem, an error is returned
func (r *Repository) AddWorktree(path string) error {
	// Validate the directory exists & is empty
	files, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to open directory %q: %w", path, err)
	}
	if len(files) > 0 {
		return fmt.Errorf("directory %q is not empty", path)
	}

	rootDir, err := r.MainWorktree()
	if err != nil {
		return fmt.Errorf("failed to determine root directory of repository: %w", err)
	}

	gitFs := osfs.New(filepath.Join(rootDir, ".git"), osfs.WithBoundOS())
	repoStore := filesystem.NewStorageWithOptions(gitFs, nil, filesystem.Options{})
	worktreeMgr, err := worktree.New(repoStore)
	if err != nil {
		return fmt.Errorf("failed to initialize new worktree manager for %q: %w", r.initPath, err)
	}

	// Create new worktree in the provided directory
	fs := osfs.New(path)
	name := filepath.Base(path)
	err = worktreeMgr.Add(fs, name)
	if err != nil {
		return fmt.Errorf("failed to create new worktree: %w", err)
	}

	return nil
}

// GitPath returns the canonical path to the .git directory or .git txt file, given the root
// directory of a repository
func GitPath(path string) string {
	return filepath.Join(path, GitStorePath)
}
