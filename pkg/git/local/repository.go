package local

import (
	"fmt"

	"github.com/go-git/go-git/v6"
)

type Repository struct {
	path string
	repo *git.Repository
}

func NewRepository(path string) (*Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read git repository %q: %w", path, err)
	}
	r := &Repository{
		path: path,
		repo: repo,
	}
	return r, nil
}

func (r Repository) DefaultBranch() (string, error) {
	return "main", nil
}
