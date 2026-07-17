package resource

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

// GithubBuildsResource is a DirectScopedResource: the Github service's
// Builds API always returns results oldest-updated-first with no
// descending/keyset pagination option, so a capped fetch of an org- or
// repo-wide list would silently show the OLDEST builds, never recent ones.
// Scope is therefore always an exact pull request or commit SHA, both
// naturally small result sets. There is no build Detail page — every row
// jumps straight to its task group via NavTarget, mirroring
// HookFiresResource.
type GithubBuildsResource struct {
	tc taskcluster.Taskcluster
}

func NewGithubBuildsResource(tc taskcluster.Taskcluster) *GithubBuildsResource {
	return &GithubBuildsResource{tc: tc}
}

var (
	_ DirectScopedResource = (*GithubBuildsResource)(nil)
	_ WebLinkable          = (*GithubBuildsResource)(nil)
)

func (r *GithubBuildsResource) Name() string      { return "githubbuilds" }
func (r *GithubBuildsResource) Aliases() []string { return []string{"builds"} }
func (r *GithubBuildsResource) Description() string {
	return "A Github pull request's or commit's builds, looked up directly by org/repo/pull/<n> or org/repo/sha/<sha>"
}
func (r *GithubBuildsResource) IDPromptLabel() string {
	return "org/repo/pull/<number> or org/repo/sha/<sha>"
}

func (r *GithubBuildsResource) Columns() []Column {
	return []Column{
		{Title: "TASK GROUP", Width: taskIDColumnWidth},
		{Title: "PR", Width: 6},
		{Title: "SHA", Width: 42},
		{Title: "STATE", Width: 12},
		{Title: "EVENT TYPE", Width: 26},
		{Title: "CREATED", Width: 24},
		{Title: "UPDATED", Expand: true},
	}
}

// List is never expected to be called via normal navigation — a
// DirectScopedResource always either has a scope, or opens an id prompt
// first.
func (r *GithubBuildsResource) List() ([]Row, error) {
	return nil, fmt.Errorf("githubbuilds requires an org/repo/pull/<n> or org/repo/sha/<sha> scope")
}

func (r *GithubBuildsResource) ScopedList(scope string) ([]Row, error) {
	filter, err := parseGithubBuildsScope(scope)
	if err != nil {
		return nil, err
	}

	builds, err := r.tc.GetGithubBuilds(filter)
	if err != nil {
		return nil, err
	}
	sortGithubBuildsNewestFirst(builds)

	rows := make([]Row, 0, len(builds))
	for _, b := range builds {
		pr := ""
		if b.PullRequestNumber != 0 {
			pr = fmt.Sprint(b.PullRequestNumber)
		}
		rows = append(rows, Row{
			ID: b.TaskGroupID,
			Cells: []string{
				b.TaskGroupID,
				pr,
				b.SHA,
				renderGithubBuildState(b.State),
				b.EventType,
				fmt.Sprint(b.Created),
				fmt.Sprint(b.Updated),
			},
			NavTarget: &NavTarget{ResourceName: "taskgroup", ID: b.TaskGroupID, Kind: NavScopedList},
		})
	}

	return rows, nil
}

// sortGithubBuildsNewestFirst orders builds by Updated, newest first. The
// Builds API itself only ever returns oldest-updated-first (see this type's
// doc comment) — without this, the first row for a PR with several pushes
// would be its OLDEST build, not its most recent one. Mirrors
// HookFiresResource's sortHookFiresNewestFirst.
func sortGithubBuildsNewestFirst(builds taskcluster.GithubBuildList) {
	sort.SliceStable(builds, func(i, j int) bool {
		return time.Time(builds[i].Updated).After(time.Time(builds[j].Updated))
	})
}

// EmptyScopeResource returns "githubrepo" — required by ScopedResource, and
// unreachable via normal navigation (a DirectScopedResource always prompts
// for a scope rather than falling through with an empty one), but must
// still name a real, non-cyclic resource per that interface's invariant;
// this keeps a hypothetical caller within the same Github domain.
func (r *GithubBuildsResource) EmptyScopeResource() string { return "githubrepo" }

// Describe is unreachable in normal use — every row overrides navigation
// via NavTarget straight to its task group (mirrors HookFiresResource).
func (r *GithubBuildsResource) Describe(id string) (Detail, error) {
	return Detail{}, fmt.Errorf("github builds are not viewable directly — select one to open its task group")
}

func (r *GithubBuildsResource) RefreshInterval() time.Duration { return 15 * time.Second }

// DetailWebURL is never expected to be called — see Describe's doc comment.
func (r *GithubBuildsResource) DetailWebURL(rootURL, id string) string { return "" }

// ListWebURL links directly to the matching Github PR or commit page. The
// shell computes header hints — and so calls this — as soon as a scope is
// typed, BEFORE that scope's list has actually been fetched/validated (see
// Shell.renderList/currentWebURL), so a malformed or upstream-invalid scope
// (a non-numeric pull number, an abbreviated sha, an org/repo containing
// characters Github names can't have) must not produce a link that looks
// plausible but 404s or points at the wrong thing — hence validating with
// parseGithubBuildsScope first, exactly like ScopedList does, rather than
// just checking the scope's shape. org/repo/sha themselves are then taken
// from a second, un-translated split (NOT parseGithubBuildsScope's
// returned filter) — that translation is an outgoing-API-query concern
// only; Github's own pages use the real name.
func (r *GithubBuildsResource) ListWebURL(rootURL, scope string) string {
	if _, err := parseGithubBuildsScope(scope); err != nil {
		return ""
	}

	org, repo, kind, value, ok := splitGithubBuildsScope(scope)
	if !ok {
		return ""
	}

	switch kind {
	case "pull":
		return fmt.Sprintf("https://github.com/%s/%s/pull/%s", pathSegment(org), pathSegment(repo), pathSegment(value))
	case "sha":
		return fmt.Sprintf("https://github.com/%s/%s/commit/%s", pathSegment(org), pathSegment(repo), pathSegment(value))
	default:
		return ""
	}
}

var githubSHAPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)

// githubNameSegmentPattern matches Github's own org/repo name pattern
// (services/github/schemas/constants.yml's github-repo-name-pattern).
// Rejecting anything else locally — rather than accepting whatever the
// user typed — matters for two independent reasons: (1) a name containing
// '%' would either be silently misinterpreted (Taskcluster's own '.'→'%'
// sanitization means "foo%bar" could accidentally address the stored
// record for a completely different repo, "foo.bar") or produce a
// doubly-escaped, nonexistent Github URL ("foo%25bar") in ListWebURL; (2)
// GithubRepositoryResource.Describe interpolates org/repo directly into a
// tview region-tag-formatted Detail body — an id containing literal
// "[...]" sequences would otherwise be rendered as real tview color tags
// rather than as plain text. Dots are allowed (real Github names can
// contain them), which is why this runs BEFORE githubDotsToPercent.
var githubNameSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// splitGithubBuildsScope splits "<org>/<repo>/<kind>/<value>" into its four
// parts, requiring exactly 4 non-empty segments (kind is validated by
// callers, since parseGithubBuildsScope and ListWebURL need to react to an
// unrecognized kind differently: the former errors, the latter just
// returns "").
func splitGithubBuildsScope(scope string) (org, repo, kind, value string, ok bool) {
	parts := strings.Split(scope, "/")
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[3] == "" {
		return "", "", "", "", false
	}
	return parts[0], parts[1], parts[2], parts[3], true
}

// parseGithubBuildsScope parses "<org>/<repo>/pull/<number>" or
// "<org>/<repo>/sha/<sha>" into a GithubBuildFilter, rejecting anything
// malformed locally rather than sending it upstream. Organization/
// Repository are translated via githubDotsToPercent; sha is lowercased and
// validated as exactly 40 hex characters — the Github service stores and
// filters on the full SHA via plain equality, so an abbreviated sha (valid
// to `git`, but not to this API) would otherwise just silently match
// nothing.
func parseGithubBuildsScope(scope string) (taskcluster.GithubBuildFilter, error) {
	org, repo, kind, value, ok := splitGithubBuildsScope(scope)
	if !ok {
		return taskcluster.GithubBuildFilter{}, fmt.Errorf(
			"invalid github builds scope %q: expected org/repo/pull/<number> or org/repo/sha/<sha>", scope)
	}
	if !githubNameSegmentPattern.MatchString(org) || !githubNameSegmentPattern.MatchString(repo) {
		return taskcluster.GithubBuildFilter{}, fmt.Errorf(
			"invalid github builds scope %q: org/repo must match Github's own name pattern (letters, digits, '.', '_', '-' only)", scope)
	}

	filter := taskcluster.GithubBuildFilter{
		Organization: githubDotsToPercent(org),
		Repository:   githubDotsToPercent(repo),
	}

	switch kind {
	case "pull":
		if !isAllDigits(value) {
			return taskcluster.GithubBuildFilter{}, fmt.Errorf(
				"invalid github builds scope %q: pull request number %q is not numeric", scope, value)
		}
		filter.PullRequest = value
	case "sha":
		sha := strings.ToLower(value)
		if !githubSHAPattern.MatchString(sha) {
			return taskcluster.GithubBuildFilter{}, fmt.Errorf(
				"invalid github builds scope %q: %q is not a full 40-character hex sha (abbreviated shas aren't supported by this API)", scope, value)
		}
		filter.SHA = sha
	default:
		return taskcluster.GithubBuildFilter{}, fmt.Errorf(
			"invalid github builds scope %q: expected \"pull\" or \"sha\", got %q", scope, kind)
	}

	return filter, nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// githubDotsToPercent mirrors the Github service's own ingestion-time
// sanitization (services/github/src/api.js's sanitizeGitHubField): webhook
// events are stored with '.' replaced by '%' in organization/repository
// (the /builds route's own query-param regex, ^([a-zA-Z0-9-_%]*)$, doesn't
// even allow a literal dot), so a query for a dotted repo name like
// "foo.bar" must ask for "foo%bar" instead. Only ever applied to
// GithubBuildFilter's Organization/Repository — never to
// GithubRepositoryResource's owner/repo, which takes the real, unsanitized
// Github name.
func githubDotsToPercent(s string) string {
	return strings.ReplaceAll(s, ".", "%")
}

// renderGithubBuildState colors a Github build's state — success/green,
// failure or error/red, pending/yellow, cancelled or anything else/white
// (default). Deliberately not reusing taskStateColor: that switch only
// recognizes completed/failed/exception/running (Taskcluster task states),
// and would render every Github build state white by default. Empty state
// passes through unchanged, matching renderTaskState's behavior.
func renderGithubBuildState(state string) string {
	if state == "" {
		return state
	}

	color := "white"
	switch state {
	case "success":
		color = "green"
	case "failure", "error":
		color = "red"
	case "pending":
		color = "yellow"
	}
	return fmt.Sprintf("[%s]%s[white]", color, state)
}
