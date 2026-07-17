package resource

import (
	"fmt"
	"strings"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

// GithubRepositoryResource is a DirectLookup for a single repo's
// Taskcluster-Github install status. The underlying Repository endpoint is
// marked EXPERIMENTAL upstream and gated by its own per-repo scope
// (github:get-repository:<owner>:<repo>) — both surfaced as a caveat in
// Describe's body, not hidden. There's no DetailAction linking to
// GithubBuildsResource: builds are now scoped to an exact PR/SHA (see that
// resource's doc comment), so "this repo's builds" isn't a well-defined
// target from here.
type GithubRepositoryResource struct {
	tc taskcluster.Taskcluster
}

func NewGithubRepositoryResource(tc taskcluster.Taskcluster) *GithubRepositoryResource {
	return &GithubRepositoryResource{tc: tc}
}

var (
	_ DirectLookup = (*GithubRepositoryResource)(nil)
	_ WebLinkable  = (*GithubRepositoryResource)(nil)
)

func (r *GithubRepositoryResource) Name() string      { return "githubrepo" }
func (r *GithubRepositoryResource) Aliases() []string { return []string{"repo"} }
func (r *GithubRepositoryResource) Description() string {
	return "A Github repository's Taskcluster integration status, looked up directly by org/repo"
}
func (r *GithubRepositoryResource) IDPromptLabel() string { return "org/repo" }
func (r *GithubRepositoryResource) Columns() []Column     { return nil }

// List is never expected to be called — GithubRepositoryResource is a
// DirectLookup; every entity is addressed directly by id.
func (r *GithubRepositoryResource) List() ([]Row, error) {
	return nil, fmt.Errorf("githubrepo requires an org/repo id")
}

func (r *GithubRepositoryResource) Describe(id string) (Detail, error) {
	org, repo, ok := splitOwnerRepo(id)
	if !ok {
		return Detail{}, fmt.Errorf("invalid github repository id %q: expected org/repo", id)
	}

	resp, err := r.tc.GetGithubRepository(org, repo)
	if err != nil {
		return Detail{}, err
	}

	installed := "[red]false[white]"
	if resp.Installed {
		installed = "[green]true[white]"
	}

	body := fmt.Sprintf(
		"[green]Organization:[white] %s\n[green]Repository:[white] %s\n[green]Installed:[white] %s\n\nNote: this endpoint is marked EXPERIMENTAL by the Taskcluster Github service, and requires its own github:get-repository:<owner>:<repo> scope — a repo that githubbuilds works for may still fail here.",
		org, repo, installed,
	)

	return Detail{Title: fmt.Sprintf("%s/%s", org, repo), Body: body}, nil
}

// RefreshInterval is 0 (manual only): unlike a cheap DB-backed lookup, the
// upstream Repository endpoint (services/github/src/api.js) paginates
// through every repo the Github App installation can see via the real
// Github API on every single call — auto-refreshing this on a timer while
// a user just sits on the Detail view would be needlessly expensive.
func (r *GithubRepositoryResource) RefreshInterval() time.Duration { return 0 }

// ListWebURL is never expected to be called — GithubRepositoryResource is
// a DirectLookup and never renders a List view.
func (r *GithubRepositoryResource) ListWebURL(rootURL, scope string) string { return "" }

func (r *GithubRepositoryResource) DetailWebURL(rootURL, id string) string {
	org, repo, ok := splitOwnerRepo(id)
	if !ok {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s", pathSegment(org), pathSegment(repo))
}

// splitOwnerRepo parses "org/repo", requiring both segments to match
// Github's own name pattern (githubNameSegmentPattern, defined in
// github_builds.go) — not just "non-empty". Beyond rejecting nonsense
// upstream ids, this also guards Describe's body: org/repo are
// interpolated directly into a tview region-tag-formatted string there, so
// without this check an id containing literal "[...]" could be rendered as
// real color tags rather than plain text.
func splitOwnerRepo(id string) (org, repo string, ok bool) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	if !githubNameSegmentPattern.MatchString(parts[0]) || !githubNameSegmentPattern.MatchString(parts[1]) {
		return "", "", false
	}
	return parts[0], parts[1], true
}
