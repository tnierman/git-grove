package git

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing/transport"

	//"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/go-git/go-git/v6/storage/memory"
)

type Repository struct {
	Remote string
}

func NewRepository(remote string) *Repository {
	r := &Repository{
		Remote: remote,
	}
	return r
}

// Auth parses the Repository's Remote URL to determine the transport protocol being used and generate
// the correct authentication method. If the protocol is SSH, then authenticaion will be done via
// SSH agent - this method does not currently support specifying keyfiles (TODO!)
//
// Supported protocols/Remote formats are:
//   - http:// or https:// -prefixed Remotes
//   - ssh:// -prefixed Remotes
//   - <user>@<remote>:<repo> -format Remotes
//
// Other formats, such as 'git://' and 'ftp://' are supported by the git-cli tool, but not by this package
func (r *Repository) Auth() (transport.AuthMethod, error) {
	// For HTTP(S): Remote must be prefixed with either 'https://' or 'http://'
	if strings.HasPrefix(r.Remote, "https://") || strings.HasPrefix(r.Remote, "http://") {
		return r.httpAuth(), nil
	}

	// SSH can have two formats: either prefixed with 'ssh://' or '<user>@<remote>:<repo>'
	sshRegex := regexp.MustCompile(".+@.+:.+")
	if strings.HasPrefix(r.Remote, "ssh://") || sshRegex.MatchString(r.Remote) {
		return r.sshAuth()
	}

	return nil, fmt.Errorf("could not determine correct transport protocol for %q (expected one of 'https://<repo>', 'ssh://<repo>', or '<user>@<remote>:<repo>')", r.Remote)
}

// httpAuth generates the authentication method used to communicate with HTTP git repos.
//
// It (interactively) queries the user for a username or password; there is currently no method
// to pass in either info via argument (TODO? What would be the use-case here?)
func (r *Repository) httpAuth() transport.AuthMethod {
	// TODO
	fmt.Println("http authentication - TODO!")
	return nil
}

func (r *Repository) sshAuth() (transport.AuthMethod, error) {
	// parse out username from remote - start by trimming the URI scheme (the 'ssh://' bit), if present
	url := strings.TrimPrefix(r.Remote, "ssh://")
	tokens := strings.Split(url, "@")
	if len(tokens) != 2 {
		return nil, fmt.Errorf("invalid format: expected exactly one '@' character for SSH authentication, found %d", len(tokens))
	}

	user := tokens[0]
	return ssh.NewSSHAgentAuth(user)
}

func (r *Repository) DefaultBranch(ctx context.Context) (string, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		URLs: []string{r.Remote},
	})

	auth, err := r.Auth()
	if err != nil {
		return "", fmt.Errorf("failed to authenticate with %q: %w", r.Remote, err)
	}

	refs, err := remote.ListContext(ctx, &git.ListOptions{
		Auth: auth,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list refs for %q: %w", r.Remote, err)
	}

	for _, ref := range refs {
		if ref.Name() == "HEAD" {
			branch := ref.Target().Short()
			if branch == "" {
				return "", fmt.Errorf("HEAD ref for %q is missing target", r.Remote)
			}
			return branch, nil
		}
	}
	return "", fmt.Errorf("no HEAD ref defined for %q", r.Remote)
}

// Clone authenticates to the Repository and clones it into the given path
func (r *Repository) Clone(path string) error {
	auth, err := r.Auth()
	if err != nil {
		return fmt.Errorf("failed to authenticate with %q: %w", r.Remote, err)

	}
	_, err = git.PlainClone(path, &git.CloneOptions{
		URL:      r.Remote,
		Auth:     auth,
		Progress: os.Stdout,
	})
	return err
}
