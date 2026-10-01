package application

import (
	"context"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

// ghCommand is how the GitHub CLI is invoked. It is not a required tool: a run that cannot ask it
// anything falls back to the remote, and a run that cannot ask either is one flag away from working.
const ghCommand = "gh"

// gitHubHost is the only host a remote may name for its path to be a GitHub repository. A remote
// pointing anywhere else is somebody else's forge, and the repository settings field is GitHub's.
const gitHubHost = "github.com"

// SuggestIssuesRepo asks which repository dir belongs to, so the survey can offer an answer rather
// than an empty field and a non-interactive run has one less flag to pass.
//
// Two sources, in this order. gh is asked first because it answers with the repository as GitHub
// knows it today, which is the right answer after a rename and for a fork whose remote still names
// the upstream. The git remote is asked second because it needs neither authentication nor network
// and git is a tool every run already requires — so a directory that has a GitHub origin gets an
// answer whether or not gh is installed and signed in.
//
// Every way of not knowing means the same thing to the caller — there is nothing to suggest — so a
// missing tool, a directory that is not a GitHub repository, and a tool failing for its own reasons
// all return None rather than an error (ADR-GO-03).
func (i *Initialize) SuggestIssuesRepo(ctx context.Context, dir string) mo.Option[string] {
	if repo := i.repoFromGitHubCLI(ctx, dir); repo.IsPresent() {
		return repo
	}

	return i.repoFromGitRemote(ctx, dir)
}

// repoFromGitHubCLI asks gh what repository this directory belongs to. The question costs an
// authenticated API call, which is why it is allowed to come back with nothing.
func (i *Initialize) repoFromGitHubCLI(ctx context.Context, dir string) mo.Option[string] {
	if i.runner.LookPath(ghCommand).IsAbsent() {
		return mo.None[string]()
	}

	result, err := i.runner.Run(ctx, dir, ghCommand, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil || result.ExitCode != 0 {
		return mo.None[string]()
	}

	repo := strings.TrimSpace(result.Stdout)
	if repo == "" {
		return mo.None[string]()
	}

	return mo.Some(repo)
}

// repoFromGitRemote reads the origin remote's URL and takes the repository out of it. Only origin is
// consulted: a directory whose GitHub repository is under some other remote name is one the person
// running init knows better than this does, and --issues-repo is how they say so.
func (i *Initialize) repoFromGitRemote(ctx context.Context, dir string) mo.Option[string] {
	if i.runner.LookPath(gitCommand).IsAbsent() {
		return mo.None[string]()
	}

	result, err := i.runner.Run(ctx, dir, gitCommand, "remote", "get-url", "origin")
	if err != nil || result.ExitCode != 0 {
		return mo.None[string]()
	}

	return repoFromRemoteURL(result.Stdout)
}

// repoFromRemoteURL is the owner/name a remote URL points at on GitHub, when it points at one.
//
// The result is held to the same rule the flag is held to, because it is going into the same
// settings field: anything ValidateRepo would refuse is not a suggestion, it is a wrong answer
// offered as a default.
func repoFromRemoteURL(raw string) mo.Option[string] {
	path, ok := gitHubPath(strings.TrimSpace(raw))
	if !ok {
		return mo.None[string]()
	}

	repo := strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	if settings.ValidateRepo(repo) != nil {
		return mo.None[string]()
	}

	return mo.Some(repo)
}

// gitHubPath splits a remote URL into the part naming the host and the part naming the repository,
// and reports whether the host was GitHub's. It handles the two shapes git writes: a URL with a
// scheme, which every https and ssh:// remote has, and the scp-like `git@github.com:owner/name.git`
// that `git clone git@…` leaves behind.
//
// A local path remote has neither shape and falls out here, which is what should happen: a directory
// on this machine is not a repository on GitHub.
func gitHubPath(remote string) (string, bool) {
	if _, rest, scheme := strings.Cut(remote, "://"); scheme {
		authority, path, separated := strings.Cut(rest, "/")

		return path, separated && isGitHubHost(authority)
	}

	authority, path, separated := strings.Cut(remote, ":")

	return path, separated && isGitHubHost(authority)
}

// isGitHubHost reports whether a URL's authority names GitHub. The user and the port are not part of
// that question, so both are dropped before the comparison — an https remote may carry either, and
// an ssh one always carries the git user.
func isGitHubHost(authority string) bool {
	host := authority

	if _, after, found := strings.Cut(host, "@"); found {
		host = after
	}

	host, _, _ = strings.Cut(host, ":")

	return strings.EqualFold(host, gitHubHost)
}
