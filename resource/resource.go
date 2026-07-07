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
}

// NavTarget identifies a resource/id pair to navigate to. The shell executes
// the navigation without interpreting what it means.
type NavTarget struct {
	ResourceName string
	ID           string
}

// DetailAction is a keybinding shown in a Detail view's footer hints that
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
	Name() string      // e.g. "workerpools" — matched in the command bar
	Aliases() []string // e.g. ["wp", "pools"]
	Columns() []Column
	List() ([]Row, error)
	Describe(id string) (Detail, error)
	RefreshInterval() time.Duration // 0 disables auto-refresh for this resource
}
