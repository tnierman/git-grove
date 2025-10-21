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

	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/go-git/go-git/v6/storage/memory"
)

type RemoteRepository struct {
	URL string
}

// NewRemoteRepository creates a RemoteRepository object for the given remote URL
func NewRemoteRepository(remoteURL string) *RemoteRepository {
	r := &RemoteRepository{
		URL: remoteURL,
	}
	return r
}

// Auth parses the RemoteRepository's URL to determine the transport protocol being used and generate
// the correct authentication method. If the protocol is SSH, then authenticaion will be done via
// SSH agent - this method does not currently support specifying keyfiles (TODO!)
//
// Supported URL formats are:
//   - prefixed with http:// or https:// for HTTP
//   - prefixed with ssh:// for SSH
//   - formatted as <user>@<remote>:<repo> for SSH
//
// Other formats, such as 'git://' and 'ftp://' are supported by the git-cli tool, but not by this package.
// Local repos (ie - /path/to/repo or or file:///path/to/repo) are likewise not (yet) supported
func (r *RemoteRepository) Auth() (transport.AuthMethod, error) {
	// For HTTP(S): URL must be prefixed with either 'https://' or 'http://'
	if strings.HasPrefix(r.URL, "https://") || strings.HasPrefix(r.URL, "http://") {
		return r.httpAuth(), nil
	}

	// SSH can have two formats: either prefixed with 'ssh://' or '<user>@<remote>:<repo>'
	sshRegex := regexp.MustCompile(".+@.+:.+")
	if strings.HasPrefix(r.URL, "ssh://") || sshRegex.MatchString(r.URL) {
		return r.sshAuth()
	}

	return nil, fmt.Errorf("could not determine correct transport protocol for %q (expected one of 'https://<repo>', 'ssh://<repo>', or '<user>@<remote>:<repo>')", r.URL)
}

// httpAuth generates the authentication method used to communicate with git repos via HTTP(S).
//
// It (interactively) queries the user for a username or password; there is currently no method
// to pass in either info via argument (TODO? What would be the use-case here?)
func (r *RemoteRepository) httpAuth() transport.AuthMethod {
	// TODO
	fmt.Println("http authentication - TODO!")
	return nil
}

// sshAuth generates the authentication method used to communicate with git repos via SSH.
// Currently, only authenticating via ssh-agent is supported (TODO - add method for auth via keyfiles)
func (r *RemoteRepository) sshAuth() (transport.AuthMethod, error) {
	// parse out username from remote - start by trimming the URI scheme (the 'ssh://' bit), if present
	url := strings.TrimPrefix(r.URL, "ssh://")
	tokens := strings.Split(url, "@")
	if len(tokens) != 2 {
		return nil, fmt.Errorf("invalid format: expected exactly one '@' character for SSH authentication, found %d", len(tokens))
	}

	user := tokens[0]
	return ssh.NewSSHAgentAuth(user)
}

// DefaultBranch attempts to determine the default branch for the RemoteRepository's URL.
// This is done by looking at the target branch for the HEAD ref from the remote.
func (r *RemoteRepository) DefaultBranch(ctx context.Context) (string, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		URLs: []string{r.URL},
	})

	auth, err := r.Auth()
	if err != nil {
		return "", fmt.Errorf("failed to authenticate with %q: %w", r.URL, err)
	}

	refs, err := remote.ListContext(ctx, &git.ListOptions{
		Auth: auth,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list refs for %q: %w", r.URL, err)
	}

	for _, ref := range refs {
		if ref.Name() == "HEAD" {
			branch := ref.Target().Short()
			if branch == "" {
				return "", fmt.Errorf("HEAD ref for %q is missing target", r.URL)
			}
			return branch, nil
		}
	}
	return "", fmt.Errorf("no HEAD ref defined for %q", r.URL)
}

// Clone authenticates to the RemoteRepository and clones it into the given path
func (r *RemoteRepository) Clone(path string) error {
	auth, err := r.Auth()
	if err != nil {
		return fmt.Errorf("failed to authenticate with %q: %w", r.URL, err)

	}
	_, err = git.PlainClone(path, &git.CloneOptions{
		URL:      r.URL,
		Auth:     auth,
		Progress: os.Stdout,
	})
	return err
}
