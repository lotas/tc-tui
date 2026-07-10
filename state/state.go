// Package state persists a tc-tui session's navigation stack, sort choices,
// and facet-tab choices to a JSON file under the user's cache directory, keyed
// per Taskcluster root URL, so the app can reopen where it left off.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// State is the full persisted snapshot of a Shell's navigation state. Kind
// and Direction fields mirror shell.ViewKind/shell.SortDirection as plain
// ints, so this package stays dependency-free rather than importing shell.
type State struct {
	Stack            []ViewState          `json:"stack"`
	SortByResource   map[string]SortState `json:"sortByResource"`
	FacetByResource  map[string]string    `json:"facetByResource"`
	FilterByResource map[string]string    `json:"filterByResource"`
	History          []HistoryEntry       `json:"history"`
}

// HistoryEntry mirrors resource.HistoryEntry (this package stays
// dependency-free rather than importing resource, same reasoning as
// ViewState mirroring shell.ViewKind).
type HistoryEntry struct {
	ResourceName string    `json:"resourceName"`
	Kind         int       `json:"kind"`
	SelectedID   string    `json:"selectedID"`
	Scope        string    `json:"scope"`
	Title        string    `json:"title"`
	VisitedAt    time.Time `json:"visitedAt"`
}

type ViewState struct {
	ResourceName string `json:"resourceName"`
	Kind         int    `json:"kind"`
	SelectedID   string `json:"selectedID"`
	Scope        string `json:"scope"`
}

type SortState struct {
	Column    int `json:"column"`
	Direction int `json:"direction"`
}

const appDirName = "tc-tui"

var unsafeRun = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// Path returns the cache file path for the given Taskcluster root URL. It
// fails only if the OS-level user cache directory can't be resolved (e.g.
// $HOME unset) — callers should treat that as "persistence unavailable" and
// skip loading/saving entirely.
func Path(rootURL string) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, appDirName, sanitize(rootURL)+".json"), nil
}

// sanitize turns a root URL into a filesystem-safe, still-readable filename
// component: strips the scheme and collapses any run of characters outside
// [a-zA-Z0-9._-] (a port's ':', a path's '/', ...) into a single '-'.
func sanitize(rootURL string) string {
	trimmed := strings.TrimPrefix(rootURL, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	trimmed = strings.Trim(trimmed, "/")

	slug := unsafeRun.ReplaceAllString(trimmed, "-")
	if slug == "" {
		return "default"
	}

	return slug
}

// Load reads and unmarshals the state file at path. Any failure (missing
// file, unreadable, malformed JSON) is treated as "no prior state" and
// silently returns a zero-value State — a corrupt cache file should never
// block startup.
func Load(path string) State {
	data, err := os.ReadFile(path)
	if err != nil {
		return State{}
	}

	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}
	}

	return st
}

// Save marshals st and writes it to path, creating the parent directory if
// needed.
func Save(path string, st State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(st)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}
