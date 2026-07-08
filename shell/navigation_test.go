package shell

import (
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/taskcluster/tc-tui/resource"
)

// newTestShellForSort builds a Shell as if renderList had just populated a
// two-column, two-row list view for a resource named "widgets" — without
// actually calling renderList (see note above).
func newTestShellForSort() *Shell {
	s := New(resource.NewRegistry())
	s.currentListResource = "widgets"
	s.currentColumns = []resource.Column{{Title: "ID"}, {Title: "VALUE"}}
	s.lastRows = []resource.Row{
		{ID: "b", Cells: []string{"b", "2"}},
		{ID: "a", Cells: []string{"a", "1"}},
	}
	return s
}

type fakeClientFacetedResource struct {
	fakeResource
	column  int
	options []string
}

func (f fakeClientFacetedResource) FacetColumn() int                          { return f.column }
func (f fakeClientFacetedResource) FacetOptions(rows []resource.Row) []string { return f.options }

type fakeServerFacetedListResource struct {
	fakeResource
	options []string
	rows    map[string][]resource.Row
	counts  map[string]int
	err     error

	mu    sync.Mutex
	calls []string // records the `value` FacetList was called with, in order
}

func (f *fakeServerFacetedListResource) FacetOptions() []string { return f.options }
func (f *fakeServerFacetedListResource) FacetList(scope, value string) ([]resource.Row, error) {
	f.mu.Lock()
	f.calls = append(f.calls, value)
	f.mu.Unlock()

	if f.err != nil {
		return nil, f.err
	}
	return f.rows[value], nil
}
func (f *fakeServerFacetedListResource) FacetCounts(scope string) (map[string]int, error) {
	return f.counts, nil
}

// lastCall returns the most recent value FacetList was called with, and
// whether it's been called at all — synchronized since FacetList runs on a
// background goroutine (loadList) while the test polls from the main one.
func (f *fakeServerFacetedListResource) lastCall() (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.calls) == 0 {
		return "", false
	}
	return f.calls[len(f.calls)-1], true
}

func TestRenderListSetsUpClientFaceted(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeClientFacetedResource{
		fakeResource: fakeResource{name: "workerpools"},
		column:       1,
		options:      []string{"aws", "gcp"},
	}

	s.renderList(res, "")

	if s.currentFaceted == nil {
		t.Fatalf("expected currentFaceted to be set")
	}
	if s.currentServerFaceted != nil {
		t.Fatalf("expected currentServerFaceted to remain nil")
	}
	if s.currentFacetValue != "" {
		t.Fatalf("expected default facet value \"\" (All), got %q", s.currentFacetValue)
	}
}

func TestRenderListSetsUpServerFacetedWithDefaultTab(t *testing.T) {
	s := New(resource.NewRegistry())
	res := &fakeServerFacetedListResource{
		fakeResource: fakeResource{name: "workers"},
		options:      []string{"running", "stopped"},
	}

	s.renderList(res, "gcp/pool-a")

	if s.currentServerFaceted == nil {
		t.Fatalf("expected currentServerFaceted to be set")
	}
	if s.currentFaceted != nil {
		t.Fatalf("expected currentFaceted to remain nil")
	}
	if s.currentFacetValue != "running" {
		t.Fatalf("expected default facet value %q, got %q", "running", s.currentFacetValue)
	}
}

func TestRenderListRestoresRememberedFacetValue(t *testing.T) {
	s := New(resource.NewRegistry())
	s.facetByResource["workers"] = "stopped"
	res := &fakeServerFacetedListResource{
		fakeResource: fakeResource{name: "workers"},
		options:      []string{"running", "stopped"},
	}

	s.renderList(res, "gcp/pool-a")

	if s.currentFacetValue != "stopped" {
		t.Fatalf("expected remembered facet value %q, got %q", "stopped", s.currentFacetValue)
	}
}

func TestRenderListFallsBackToDefaultForStaleRememberedValue(t *testing.T) {
	s := New(resource.NewRegistry())
	s.facetByResource["workers"] = "no-longer-valid"
	res := &fakeServerFacetedListResource{
		fakeResource: fakeResource{name: "workers"},
		options:      []string{"running", "stopped"},
	}

	s.renderList(res, "gcp/pool-a")

	if s.currentFacetValue != "running" {
		t.Fatalf("expected fallback to first option %q, got %q", "running", s.currentFacetValue)
	}
}

func TestRefreshTableSkipsClientFilterForServerFaceted(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentListResource = "workers"
	s.currentColumns = []resource.Column{{Title: "STATE"}}
	s.currentServerFaceted = &fakeServerFacetedListResource{options: []string{"running"}}
	s.currentFacetValue = "running"
	// lastRows already server-filtered to "running" — refreshTable must not
	// drop any of them via a (nonexistent) client-side facet filter.
	s.lastRows = []resource.Row{
		{ID: "a", Cells: []string{"running"}},
		{ID: "b", Cells: []string{"running"}},
	}

	s.refreshTable()

	if s.table.GetRowCount() != 3 { // 1 header row + 2 data rows
		t.Fatalf("expected 2 data rows to survive refreshTable, got %d total rows", s.table.GetRowCount())
	}
}

func TestCycleFacetClientSideIsInstantAndPersists(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentListResource = "workerpools"
	s.currentColumns = []resource.Column{{Title: "PROVIDER"}}
	s.currentFaceted = fakeClientFacetedResource{column: 0, options: []string{"aws", "gcp"}}
	s.currentFacetValue = ""
	s.lastRows = []resource.Row{
		{ID: "a", Cells: []string{"aws"}},
		{ID: "b", Cells: []string{"gcp"}},
	}

	s.cycleFacet(1) // All -> aws

	if s.currentFacetValue != "aws" {
		t.Fatalf("expected \"aws\", got %q", s.currentFacetValue)
	}
	if s.facetByResource["workerpools"] != "aws" {
		t.Fatalf("expected facet remembered for workerpools, got %+v", s.facetByResource)
	}
	if s.table.GetRowCount() != 2 { // 1 header + 1 matching row
		t.Fatalf("expected table filtered to 1 matching row, got %d total rows", s.table.GetRowCount())
	}
}

func TestCycleFacetServerSideTriggersRefetch(t *testing.T) {
	s := New(resource.NewRegistry())
	res := &fakeServerFacetedListResource{
		fakeResource: fakeResource{name: "workers"},
		options:      []string{"running", "stopped"},
		rows: map[string][]resource.Row{
			"stopped": {{ID: "a", Cells: []string{"stopped"}}},
		},
	}
	s.stack.Push(View{ResourceName: "workers", Kind: ListKind, Scope: "gcp/pool-a"})
	s.currentListResource = "workers"
	s.currentColumns = []resource.Column{{Title: "STATE"}}
	s.currentServerFaceted = res
	s.currentFacetValue = "running"

	s.cycleFacet(1) // running -> stopped, triggers loadList in a goroutine

	if s.currentFacetValue != "stopped" {
		t.Fatalf("expected \"stopped\", got %q", s.currentFacetValue)
	}
	if s.facetByResource["workers"] != "stopped" {
		t.Fatalf("expected facet remembered for workers, got %+v", s.facetByResource)
	}

	waitFor(t, func() bool {
		last, ok := res.lastCall()
		return ok && last == "stopped"
	})
}

// waitFor polls cond briefly and fails the test if it never becomes true.
// It's used to observe a background goroutine's synchronous side effects
// (e.g. FacetList being called) without needing the queued QueueUpdateDraw
// callback to actually run — s.app is never Run() in these tests, so that
// callback would never be drained; the assertions here only depend on work
// the goroutine does before reaching QueueUpdateDraw.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("condition never became true")
}

func TestToggleSortFirstPressSortsAscending(t *testing.T) {
	s := newTestShellForSort()

	s.toggleSort(1) // VALUE column

	if s.currentSort.Column != 1 || s.currentSort.Direction != SortAsc {
		t.Fatalf("expected ascending sort on column 1, got %+v", s.currentSort)
	}
}

func TestToggleSortSameColumnReversesDirection(t *testing.T) {
	s := newTestShellForSort()

	s.toggleSort(1)
	s.toggleSort(1)

	if s.currentSort.Direction != SortDesc {
		t.Fatalf("expected second press to reverse to descending, got %+v", s.currentSort)
	}

	s.toggleSort(1)
	if s.currentSort.Direction != SortAsc {
		t.Fatalf("expected third press to reverse back to ascending, got %+v", s.currentSort)
	}
}

func TestToggleSortDifferentColumnStartsAscending(t *testing.T) {
	s := newTestShellForSort()

	s.toggleSort(1)
	s.toggleSort(1) // now descending on column 1
	s.toggleSort(0) // switch to column 0

	if s.currentSort.Column != 0 || s.currentSort.Direction != SortAsc {
		t.Fatalf("expected fresh ascending sort on column 0, got %+v", s.currentSort)
	}
}

func TestToggleSortOutOfRangeColumnIsNoOp(t *testing.T) {
	s := newTestShellForSort() // 2 columns

	s.toggleSort(5)

	if s.currentSort.Direction != SortNone {
		t.Fatalf("expected no-op for out-of-range column, got %+v", s.currentSort)
	}
}

func TestToggleSortRemembersPerResourceInMap(t *testing.T) {
	s := newTestShellForSort()

	s.toggleSort(1)

	got, ok := s.sortByResource["widgets"]
	if !ok || got.Column != 1 || got.Direction != SortAsc {
		t.Fatalf("expected widgets' sort remembered in map, got %+v (ok=%v)", got, ok)
	}

	// renderList's restore-on-switch does `s.currentSort =
	// s.sortByResource[res.Name()]` — for a resource with no memory yet,
	// that's the zero value (unsorted), same as this lookup.
	restored := s.sortByResource["gadgets"]
	if restored.Direction != SortNone {
		t.Fatalf("expected zero-value (unsorted) for a resource with no memory, got %+v", restored)
	}
}

func TestToggleSortResetsSelectionToTopRow(t *testing.T) {
	s := newTestShellForSort()
	s.refreshTable() // populate the table, as renderList/loadList would

	// Simulate the user having moved the cursor down to the second row
	// ("a") before triggering a sort.
	s.table.Select(2, 0)
	if row, _ := s.table.GetSelection(); row != 2 {
		t.Fatalf("test setup: expected cursor on row 2, got row %d", row)
	}

	s.toggleSort(0) // sort by ID ascending

	row, _ := s.table.GetSelection()
	if row != 1 {
		t.Fatalf("expected sorting to reset the cursor to the top row, got row %d", row)
	}
}

func TestGlobalInputCaptureDigitTriggersSortOnTablePage(t *testing.T) {
	s := newTestShellForSort()

	event := tcell.NewEventKey(tcell.KeyRune, '2', tcell.ModNone)
	if got := s.globalInputCapture(event); got != nil {
		t.Fatalf("expected digit key to be swallowed, got %#v", got)
	}

	if s.currentSort.Column != 1 || s.currentSort.Direction != SortAsc {
		t.Fatalf("expected '2' to sort column index 1 ascending, got %+v", s.currentSort)
	}
}

func TestGlobalInputCaptureDigitBeyondColumnCountIsNoOp(t *testing.T) {
	s := newTestShellForSort() // 2 columns

	event := tcell.NewEventKey(tcell.KeyRune, '9', tcell.ModNone)
	s.globalInputCapture(event)

	if s.currentSort.Direction != SortNone {
		t.Fatalf("expected out-of-range digit to be a no-op, got %+v", s.currentSort)
	}
}
