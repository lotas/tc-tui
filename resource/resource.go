package resource

import "time"

// Column describes one column of a Resource's list view.
type Column struct {
	Title string
	Width int // 0 = flexible
}

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
