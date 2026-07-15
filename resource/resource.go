package resource

import "time"

// Column describes one column of a Resource's list view. Width and Expand
// are independent: Width is a truncation cap (removed entirely by the 'x'
// key — see shell.TableView), while Expand says whether the column also
// soaks up leftover terminal width beyond that cap, the way a CSS flex-grow
// column would. Width: 0 implies Expand regardless of Expand's own value —
// a column with no cap at all only makes sense if it also grows to fill
// whatever space it's given, matching every existing zero-Width column's
// original "flexible" behavior.
type Column struct {
	Title  string
	Width  int  // 0 = size to content, uncapped even without Expand
	Expand bool // grow into leftover terminal width past Width, if any remains
}

// taskIDColumnWidth fits a Taskcluster task ID (a 22-character slugid) with a
// little breathing room. Task ID columns are given this fixed width rather
// than left flexible (Width: 0), so a flexible column elsewhere (e.g. NAME)
// absorbs the terminal's leftover width instead.
const taskIDColumnWidth = 24

// workerPoolColumnWidth fits a workerPoolId/taskQueueId
// (provisionerId/workerType, e.g. "proj-fuzzing/grizzly-reduce-worker-android")
// — wider than a bare workerType, since the provisioner prefix can push the
// combined string well past 24 characters.
const workerPoolColumnWidth = 40

// Row is one row of a Resource's list view. ID is the stable identity used
// to look the entity back up (e.g. via Describe) — never a slice index.
type Row struct {
	ID    string
	Cells []string
	// NavTarget, if non-nil, overrides the default "select this row → open
	// Detail for (the resource currently listed, row.ID)" behavior. Only
	// HistoryResource sets this today — every other resource's rows leave it
	// nil and get today's behavior unchanged.
	NavTarget *NavTarget
}

// NavTargetKind distinguishes what a NavTarget navigates to.
type NavTargetKind int

const (
	// NavDetail pushes another entity's Detail view — the default/original
	// behavior, so existing zero-value NavTargets are unaffected.
	NavDetail NavTargetKind = iota
	// NavScopedList pushes a Resource's List view, scoped to this ID.
	NavScopedList
)

// NavTarget identifies a resource/id pair to navigate to. The shell executes
// the navigation without interpreting what it means.
type NavTarget struct {
	ResourceName string
	ID           string
	Kind         NavTargetKind
}

// DetailAction is a keybinding shown in a Detail view's header hints that
// navigates to another resource/id when pressed.
type DetailAction struct {
	Key    rune
	Label  string
	Target NavTarget
}

// Detail is the single-entity view rendered by a Resource's Describe method.
type Detail struct {
	Title   string
	Body    string // tview region-tag formatted text
	Actions []DetailAction
}

// Resource is implemented once per entity type (roles, worker pools, ...).
// The shell is generic over this interface and never references a concrete
// resource type.
type Resource interface {
	Name() string        // e.g. "workerpools" — matched in the command bar
	Aliases() []string   // e.g. ["wp", "pools"]
	Description() string // one-line summary shown on the help screen
	Columns() []Column
	List() ([]Row, error)
	Describe(id string) (Detail, error)
	RefreshInterval() time.Duration // 0 disables auto-refresh for this resource
}

// ScopedResource is implemented by resources whose list can be narrowed to
// a parent scope (e.g. workers within one worker pool). The shell calls
// ScopedList when navigating with a scope (drill-down, or `:name scope` in
// the command bar); List (from the embedded Resource) is not expected to be
// called via normal navigation for a ScopedResource — the shell always
// either has a scope, or redirects to EmptyScopeResource() first.
//
// Invariant: EmptyScopeResource must not resolve (directly or transitively)
// to another ScopedResource with an empty scope — the shell does not guard
// against redirect cycles.
type ScopedResource interface {
	Resource
	ScopedList(scope string) ([]Row, error)
	EmptyScopeResource() string // resource name to show when no scope is given
}

// DirectScopedResource is a ScopedResource that's reached by looking its
// scope up directly by ID (e.g. a task group ID pasted from a URL or log)
// rather than by drilling down from a parent list. The shell prompts for an
// ID exactly like DirectLookup, but treats what's entered as this resource's
// scope — pushing its List view — instead of an entity to Describe.
type DirectScopedResource interface {
	ScopedResource
	IDPromptLabel() string
}

// ScopeSubtitle is implemented by a ScopedResource whose List view wants
// extra static context about its scope surfaced in the title bar (e.g. a
// task group's sealed status) — fetched once alongside the scoped list
// itself, in addition to (not instead of) its normal rows.
type ScopeSubtitle interface {
	ScopedResource
	Subtitle(scope string) (string, error)
}

// DirectLookup is implemented by resources with no meaningful list at all —
// every entity is addressed directly by its own ID (e.g. a single task or a
// task group; Taskcluster's Queue API has no "list all tasks"/"list all task
// groups" endpoint). The shell treats a `:name <id>` command's argument as
// the ID to Describe immediately, skipping the List view entirely. With no
// id given, the shell opens an inline prompt for one instead of attempting a
// fetch.
//
// IDPromptLabel is DirectLookup's only method beyond Resource — required so
// the interface isn't structurally identical to Resource (which would make
// every Resource satisfy it). It doubles as that prompt's label, e.g. "task
// id".
type DirectLookup interface {
	Resource
	IDPromptLabel() string
}

// Faceted is implemented by resources whose list can be narrowed to a
// secondary tabs-style filter over one column's value, filtered client-side
// over rows the shell has already fetched — appropriate only when the full
// unfiltered list is cheap to load (e.g. worker pools: a few hundred total).
// The shell renders a tab bar, derives per-tab row counts, and filters to
// the selected tab, with no extra API calls.
type Faceted interface {
	Resource
	FacetColumn() int                 // index into Columns()/Row.Cells to filter on
	FacetOptions(rows []Row) []string // ordered tab values (excluding "All"); resource
	                                   // decides whether to hardcode or derive from rows
}

// ServerFaceted is implemented by resources whose facet tabs must be
// filtered at the API level rather than over an already-loaded row set,
// because the combined/unfiltered list can be too large to load or render
// reasonably (e.g. a worker pool with hundreds of running workers but tens
// of thousands of stopped ones). Unlike Faceted, the shell does not derive
// or filter tabs itself — FacetOptions() is the complete, authoritative tab
// list. A resource may include its own "show everything" value if its data
// volume makes that safe; WorkersResource does not.
type ServerFaceted interface {
	Resource
	FacetOptions() []string                          // ordered; first is the default/initial tab
	FacetList(scope, value string) ([]Row, error)     // fetches only rows matching this tab
	FacetCounts(scope string) (map[string]int, error) // cheap aggregate counts per tab value; no row fetch
}

// ScopeActions is implemented by a ScopedResource whose scope has sibling
// resources reachable via quick jump keys (e.g. a worker pool's
// workers/pending/claimed/launchconfigs/errors) — it lets a List view expose
// those keys without navigating back up to the parent's own Detail view.
type ScopeActions interface {
	ScopedResource
	ScopeActions(scope string) []DetailAction
}

// Augmentable is implemented by a Resource whose List()-produced rows can
// be rendered immediately and then enriched by slower, per-row API calls
// (e.g. worker pools' Pending/Claimed/Errors columns). The shell renders
// the base rows as usual, then calls Augment to receive
// progressively-updated row sets and a completed/total count for a
// progress indicator.
type Augmentable interface {
	Resource
	// Augment enriches rows (the just-rendered base rows), calling onUpdate
	// with a fully-independent row snapshot each time new data arrives,
	// plus a completed/total count. onUpdate may be called from any
	// goroutine — the shell handles its own synchronization with the UI
	// thread. Augment blocks until no more updates will come.
	Augment(rows []Row, onUpdate func(rows []Row, completed, total int))
}

// WebLinkable is implemented by resources that have a corresponding page in
// Taskcluster's web UI (a different app sharing the same root URL), letting
// the shell open that page in a browser. DetailWebURL builds the link for a
// Detail view (id is the entity's own ID); ListWebURL builds it for a List
// view (scope is whatever scope that list is currently showing, "" for an
// unscoped list). Either may return "" if that particular id/scope has no
// corresponding page in the web UI, in which case the shell shows a warning
// instead of opening a browser.
type WebLinkable interface {
	Resource
	DetailWebURL(rootURL, id string) string
	ListWebURL(rootURL, scope string) string
}

// Downloadable is implemented by a Resource whose Detail view can save its
// entity's raw content to a local file (e.g. an artifact's actual bytes) —
// distinct from WebLinkable's DetailWebURL, which downloads via an external
// browser instead of directly from within the TUI.
type Downloadable interface {
	Resource
	// DownloadFilename returns the suggested local filename for id, and
	// whether id supports being downloaded at all. Derived from id alone,
	// with no fetch, so it's cheap enough to call before prompting for a
	// save path.
	DownloadFilename(id string) (filename string, ok bool)
	// DownloadContent fetches id's raw content to save to disk — the
	// original bytes, not whatever transformed preview a Detail view's Body
	// might render (e.g. syntax-highlighted or ANSI-translated). truncated
	// reports whether the underlying fetch was capped before reaching the
	// content's actual end.
	DownloadContent(id string) (content []byte, truncated bool, err error)
}
