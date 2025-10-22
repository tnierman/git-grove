package remote

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"regexp"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing/transport"
	"golang.org/x/term"

	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/go-git/go-git/v6/storage/memory"
)

type Repository struct {
	Authentication
	URL string
}

// NewRepository creates a Repository object for the given remote URL
func NewRepository(remoteURL string) (*Repository, error) {
	auth, err := AuthMethod(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize authentication method: %w", err)
	}
	r := &Repository{
		URL:            remoteURL,
		Authentication: auth,
	}
	return r, nil
}

// DefaultBranch attempts to determine the default branch for the Repository's URL.
// This is done by looking at the target branch for the HEAD ref from the remote.
func (r *Repository) DefaultBranch(ctx context.Context) (string, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		URLs: []string{r.URL},
	})

	auth, err := r.NewAuthMethod()
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

// Clone authenticates to the Repository and clones it into the given path
func (r *Repository) Clone(path string) error {
	auth, err := r.NewAuthMethod()
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

type Authentication interface {
	NewAuthMethod() (transport.AuthMethod, error)
}

// AuthMethod parses the Repository's URL to determine the transport protocol being used and generate
// the correct Authentication method. If the protocol is SSH, then authenticaion will be done via
// SSH agent - this method does not currently support specifying keyfiles (TODO!)
//
// Supported formats are:
//   - URL prefixed with http:// or https:// for HTTP(S)
//   - URL prefixed with ssh:// for SSH
//   - URL formatted as <user>@<remote>:<repo> for SSH
//
// Other formats, such as 'git://' and 'ftp://' are supported by the git-cli tool, but not by this package.
// Local repos (ie - /path/to/repo or or file:///path/to/repo) are likewise not (yet) supported
func AuthMethod(url string) (Authentication, error) {
	// For HTTP(S): URL must be prefixed with either 'https://' or 'http://'
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		return NewHTTPAuthentication(), nil
	}

	// SSH can have two formats: either prefixed with 'ssh://' or '<user>@<remote>:<repo>'
	sshRegex := regexp.MustCompile(".+@.+:.+")
	if strings.HasPrefix(url, "ssh://") || sshRegex.MatchString(url) {
		return NewSSHAuthentication(url), nil
	}

	return nil, fmt.Errorf("could not determine correct transport protocol for %q (expected one of 'https://<repo>', 'ssh://<repo>', or '<user>@<remote>:<repo>')", url)
}

// HTTPAuthentication grants the ability to authenticate against HTTP(S) remote repositories
//
// It (interactively) queries the user for a username or password; there is currently no method
// to pass in either info via argument (TODO? What would be the use-case here?)
type HTTPAuthentication struct {
	authMethod transport.AuthMethod
}

func NewHTTPAuthentication() *HTTPAuthentication {
	return &HTTPAuthentication{}
}

const (
	httpAuthUsernamePrompt = "username: "
	httpAuthPasswordPrompt = "password: "
)

// NewAuthMethod generates the authentication method used to communicate with git repos via HTTP(S).
//
// It interactively queries the user for a username and password, if one has not yet been provided.
func (a *HTTPAuthentication) NewAuthMethod() (transport.AuthMethod, error) {
	if a.authMethod != nil {
		return a.authMethod, nil
	}
	return a.createCachedAuthMethod()
}

func (a *HTTPAuthentication) createCachedAuthMethod() (transport.AuthMethod, error) {
	fmt.Print(httpAuthUsernamePrompt)
	username, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read username entry: %w", err)
	}
	username = strings.TrimSpace(username)

	fmt.Print(httpAuthPasswordPrompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, fmt.Errorf("failed to read password entry: %w", err)
	}

	auth := &http.BasicAuth{
		Username: username,
		Password: string(password),
	}
	// cache to avoid re-querying user
	a.authMethod = auth
	return auth, nil
}

// SSHAuthentication grants the ability to authenticate against SSH remote repositories
//
// NewAuthMethod generates the authentication method used to communicate with git repos via SSH.
// Currently, only authenticating via ssh-agent is supported (TODO - add method for auth via keyfiles)
type SSHAuthentication struct {
	URL string
}

func NewSSHAuthentication(url string) *SSHAuthentication {
	a := &SSHAuthentication{
		URL: url,
	}
	return a
}

// NewAuthMethod generates the authentication method used to communicate with git repos via SSH.
// Currently, only authenticating via ssh-agent is supported (TODO - add method for auth via keyfiles)
func (a *SSHAuthentication) NewAuthMethod() (transport.AuthMethod, error) {
	a.URL = strings.TrimPrefix(a.URL, "ssh://")
	tokens := strings.Split(a.URL, "@")
	if len(tokens) != 2 {
		return nil, fmt.Errorf("invalid format: expected exactly one '@' character for SSH authentication, found %d", len(tokens))
	}

	user := tokens[0]
	return ssh.NewSSHAgentAuth(user)
}
