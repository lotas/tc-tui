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
	ttl     time.Duration // overrides fakeResource's default RefreshInterval() of 0

	mu    sync.Mutex
	calls []string // records the `value` FacetList was called with, in order
}

func (f *fakeServerFacetedListResource) RefreshInterval() time.Duration { return f.ttl }
func (f *fakeServerFacetedListResource) FacetOptions() []string         { return f.options }
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

	s.renderList(res, "", false)

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

	s.renderList(res, "gcp/pool-a", false)

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

	s.renderList(res, "gcp/pool-a", false)

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

	s.renderList(res, "gcp/pool-a", false)

	if s.currentFacetValue != "running" {
		t.Fatalf("expected fallback to first option %q, got %q", "running", s.currentFacetValue)
	}
}

func TestRenderListRestoresRememberedFilterQuery(t *testing.T) {
	s := New(resource.NewRegistry())
	s.filterByResource["workerpools"] = "proj-task"
	res := fakeResource{name: "workerpools"}

	s.renderList(res, "", false)

	if s.filterQuery != "proj-task" {
		t.Fatalf("expected remembered filter query %q, got %q", "proj-task", s.filterQuery)
	}
}

func TestRenderListRecordsScope(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeScopedResource{fakeResource: fakeResource{name: "runs"}}

	s.renderList(res, "task-1", false)

	if s.currentListScope != "task-1" {
		t.Fatalf("expected currentListScope %q, got %q", "task-1", s.currentListScope)
	}
}

func TestRenderListClearsScopeForUnscopedList(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentListScope = "stale"
	res := fakeResource{name: "workerpools"}

	s.renderList(res, "", false)

	if s.currentListScope != "" {
		t.Fatalf("expected empty currentListScope, got %q", s.currentListScope)
	}
}

func TestRenderListDefaultsToEmptyFilterQueryWhenNeverSet(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeResource{name: "workerpools"}

	s.renderList(res, "", false)

	if s.filterQuery != "" {
		t.Fatalf("expected empty filter query, got %q", s.filterQuery)
	}
}

func TestApplyListResultBumpsAugmentEpoch(t *testing.T) {
	s := newTestShellForSort()
	before := s.augmentEpoch

	s.applyListResult(fakeResource{name: "widgets"}, s.lastRows, nil)

	if s.augmentEpoch != before+1 {
		t.Fatalf("expected augmentEpoch to increment by exactly 1, got %d (was %d)", s.augmentEpoch, before)
	}
}

func TestApplyListResultResetsAugmentProgress(t *testing.T) {
	s := newTestShellForSort()
	s.augmentCompleted, s.augmentTotal = 3, 10

	s.applyListResult(fakeResource{name: "widgets"}, s.lastRows, nil)

	if s.augmentCompleted != 0 || s.augmentTotal != 0 {
		t.Fatalf("expected progress reset to 0/0, got %d/%d", s.augmentCompleted, s.augmentTotal)
	}
}

func TestRefreshTableShowsActiveFilterInTitle(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentListResource = "workerpools"
	s.currentColumns = []resource.Column{{Title: "NAME"}}
	s.lastRows = []resource.Row{{ID: "a", Cells: []string{"aws-1"}}}

	s.filterQuery = "aws"
	s.refreshTable()
	if got, want := s.content.GetTitle(), "[ Taskcluster :: workerpools (aws) ]"; got != want {
		t.Fatalf("title with active filter = %q, want %q", got, want)
	}

	s.filterQuery = ""
	s.refreshTable()
	if got, want := s.content.GetTitle(), "[ Taskcluster :: workerpools ]"; got != want {
		t.Fatalf("title with cleared filter = %q, want %q", got, want)
	}
}

func TestRefreshTableShowsScopeInTitle(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentListResource = "runs"
	s.currentListScope = "task-1"
	s.currentColumns = []resource.Column{{Title: "RUN"}}
	s.lastRows = []resource.Row{{ID: "task-1/0", Cells: []string{"0"}}}

	s.refreshTable()

	if got, want := s.content.GetTitle(), "[ Taskcluster :: runs (task-1) ]"; got != want {
		t.Fatalf("title with scope = %q, want %q", got, want)
	}
}

func TestRefreshTableShowsAugmentProgressInTitle(t *testing.T) {
	s := newTestShellForSort() // currentListResource: "widgets", two rows already set up
	s.augmentCompleted, s.augmentTotal = 2, 5

	s.refreshTable()

	if got, want := s.content.GetTitle(), "[ Taskcluster :: widgets [2/5] ]"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRefreshTableHidesAugmentSuffixOnceComplete(t *testing.T) {
	s := newTestShellForSort()
	s.augmentCompleted, s.augmentTotal = 5, 5

	s.refreshTable()

	if got, want := s.content.GetTitle(), "[ Taskcluster :: widgets ]"; got != want {
		t.Fatalf("got %q, want %q", got, want)
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

func TestLoadListServesFromCacheWithoutFetching(t *testing.T) {
	s := New(resource.NewRegistry())
	res := &fakeServerFacetedListResource{
		fakeResource: fakeResource{name: "workers"},
		ttl:          time.Minute,
		options:      []string{"running"},
	}
	s.currentListResource = "workers"
	s.currentColumns = []resource.Column{{Title: "STATE"}}
	s.currentServerFaceted = res
	s.currentFacetValue = "running"
	s.cache.set(cacheKeyFor(res, "pool-a", "running"), cacheEntry{
		rows:      []resource.Row{{ID: "cached", Cells: []string{"running"}}},
		fetchedAt: time.Now(),
	})

	s.loadList(res, "pool-a", "running", true, false, false)

	if len(s.lastRows) != 1 || s.lastRows[0].ID != "cached" {
		t.Fatalf("expected cache-hit rows to be applied, got %+v", s.lastRows)
	}
	if _, called := res.lastCall(); called {
		t.Fatalf("expected no fetch on a cache hit, but FacetList was called")
	}
}

func TestLoadListForceRefreshBypassesCache(t *testing.T) {
	s := New(resource.NewRegistry())
	res := &fakeServerFacetedListResource{
		fakeResource: fakeResource{name: "workers"},
		ttl:          time.Minute,
		options:      []string{"running"},
		rows: map[string][]resource.Row{
			"running": {{ID: "fresh", Cells: []string{"running"}}},
		},
	}
	s.currentListResource = "workers"
	s.currentColumns = []resource.Column{{Title: "STATE"}}
	s.currentServerFaceted = res
	s.currentFacetValue = "running"
	s.stack.Push(View{ResourceName: "workers", Kind: ListKind, Scope: "pool-a"})
	s.cache.set(cacheKeyFor(res, "pool-a", "running"), cacheEntry{
		rows:      []resource.Row{{ID: "stale", Cells: []string{"running"}}},
		fetchedAt: time.Now(),
	})

	s.loadList(res, "pool-a", "running", false, true, false)

	waitFor(t, func() bool {
		_, called := res.lastCall()
		return called
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

// fakeHistoryRecorder is a minimal resource.HistoryRecorder for tests that
// need to observe what Shell recorded, without going through the real
// HistoryResource (whose own behavior is covered by resource/history_test.go).
type fakeHistoryRecorder struct {
	mu      sync.Mutex
	entries []resource.HistoryEntry
}

func (f *fakeHistoryRecorder) Record(e resource.HistoryEntry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, e)
}

func (f *fakeHistoryRecorder) Entries() []resource.HistoryEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]resource.HistoryEntry(nil), f.entries...)
}

func (f *fakeHistoryRecorder) Restore(entries []resource.HistoryEntry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = entries
}

// fakeScopedTTLResource is a ScopedResource with a settable RefreshInterval,
// needed to exercise loadList's cache-hit branch for a scoped (not
// ServerFaceted) resource — fakeResource's RefreshInterval() is hardcoded to
// 0, which the cache always treats as a miss (see cache.go's get).
type fakeScopedTTLResource struct {
	fakeResource
	ttl time.Duration
}

func (f fakeScopedTTLResource) RefreshInterval() time.Duration                  { return f.ttl }
func (f fakeScopedTTLResource) ScopedList(scope string) ([]resource.Row, error) { return nil, nil }
func (f fakeScopedTTLResource) EmptyScopeResource() string                      { return "" }

func TestLoadListRecordsScopedVisitOnSuccessfulCacheHit(t *testing.T) {
	s := New(resource.NewRegistry())
	rec := &fakeHistoryRecorder{}
	s.historyRecorder = rec
	s.currentColumns = []resource.Column{{Title: "ID"}}

	res := fakeScopedTTLResource{fakeResource: fakeResource{name: "workers"}, ttl: time.Minute}
	s.cache.set(cacheKeyFor(res, "pool-a", ""), cacheEntry{
		rows:      []resource.Row{{ID: "worker-1", Cells: []string{"worker-1"}}},
		fetchedAt: time.Now(),
	})

	s.loadList(res, "pool-a", "", true, false, false)

	entries := rec.Entries()
	if len(entries) != 1 || entries[0].ResourceName != "workers" || entries[0].Scope != "pool-a" || entries[0].Kind != 0 {
		t.Fatalf("expected one recorded scoped-list visit, got %+v", entries)
	}
}

func TestLoadListDoesNotRecordUnscopedVisit(t *testing.T) {
	s := New(resource.NewRegistry())
	rec := &fakeHistoryRecorder{}
	s.historyRecorder = rec
	s.currentColumns = []resource.Column{{Title: "ID"}}

	res := fakeScopedTTLResource{fakeResource: fakeResource{name: "workerpools"}, ttl: time.Minute}
	s.cache.set(cacheKeyFor(res, "", ""), cacheEntry{
		rows:      []resource.Row{{ID: "pool-a", Cells: []string{"pool-a"}}},
		fetchedAt: time.Now(),
	})

	s.loadList(res, "", "", true, false, false)

	if len(rec.Entries()) != 0 {
		t.Fatalf("expected no recorded visit for an unscoped list, got %+v", rec.Entries())
	}
}

func TestLoadListDoesNotRecordDuringRestore(t *testing.T) {
	s := New(resource.NewRegistry())
	rec := &fakeHistoryRecorder{}
	s.historyRecorder = rec
	s.currentColumns = []resource.Column{{Title: "ID"}}

	res := fakeScopedTTLResource{fakeResource: fakeResource{name: "workers"}, ttl: time.Minute}
	s.cache.set(cacheKeyFor(res, "pool-a", ""), cacheEntry{
		rows:      []resource.Row{{ID: "worker-1", Cells: []string{"worker-1"}}},
		fetchedAt: time.Now(),
	})

	s.loadList(res, "pool-a", "", true, false, true) // isRestore=true

	if len(rec.Entries()) != 0 {
		t.Fatalf("expected no recorded visit while isRestore=true, got %+v", rec.Entries())
	}
}

func TestLoadListDoesNotRecordForHistoryResourceItself(t *testing.T) {
	s := New(resource.NewRegistry())
	rec := &fakeHistoryRecorder{}
	s.historyRecorder = rec
	s.currentColumns = []resource.Column{{Title: "ID"}}

	res := fakeScopedTTLResource{fakeResource: fakeResource{name: "history"}, ttl: time.Minute}
	s.cache.set(cacheKeyFor(res, "pool-a", ""), cacheEntry{
		rows:      []resource.Row{{ID: "x", Cells: []string{"x"}}},
		fetchedAt: time.Now(),
	})

	s.loadList(res, "pool-a", "", true, false, false)

	if len(rec.Entries()) != 0 {
		t.Fatalf("expected no recorded visit for the history resource itself, got %+v", rec.Entries())
	}
}

func TestSwitchResourceDirectLookupWithIDGoesStraightToDetail(t *testing.T) {
	registry := resource.NewRegistry()
	registry.Register(fakeDirectLookupResource{
		fakeResource: fakeResource{name: "task"},
		label:        "task id",
	})
	s := New(registry)

	s.switchResource("task", "task-1")

	top, ok := s.stack.Top()
	if !ok || top.Kind != DetailKind || top.SelectedID != "task-1" || top.ResourceName != "task" {
		t.Fatalf("unexpected top view: %+v (ok=%v)", top, ok)
	}
}

func TestSwitchResourceDirectLookupWithoutIDOpensPrompt(t *testing.T) {
	registry := resource.NewRegistry()
	registry.Register(fakeDirectLookupResource{
		fakeResource: fakeResource{name: "task"},
		label:        "task id",
	})
	s := New(registry)

	s.switchResource("task", "")

	if s.footerMode != footerPrompt {
		t.Fatalf("expected footerMode footerPrompt, got %v", s.footerMode)
	}
	if s.pendingLookup == nil || s.pendingLookup.Name() != "task" {
		t.Fatalf("expected pendingLookup set to the task resource, got %+v", s.pendingLookup)
	}
}

func TestRenderRestoredTopFallsBackToRootWhenStackIsEmpty(t *testing.T) {
	registry := resource.NewRegistry()
	registry.Register(fakeResource{name: "workerpools"})
	s := New(registry)
	s.restoreFallback = "workerpools"

	s.renderRestoredTop()

	top, ok := s.stack.Top()
	if !ok || top.ResourceName != "workerpools" || top.Kind != ListKind {
		t.Fatalf("expected root view on top, got %+v (ok=%v)", top, ok)
	}
}

func TestRenderRestoredTopDropsUnresolvableEntriesThenFallsBackToRoot(t *testing.T) {
	registry := resource.NewRegistry()
	registry.Register(fakeResource{name: "workerpools"})
	s := New(registry)
	s.restoreFallback = "workerpools"
	s.stack.Push(View{ResourceName: "long-gone-resource", Kind: ListKind})

	s.renderRestoredTop()

	top, ok := s.stack.Top()
	if !ok || top.ResourceName != "workerpools" || top.Kind != ListKind {
		t.Fatalf("expected root view on top after dropping the unresolvable entry, got %+v (ok=%v)", top, ok)
	}
}

// TestRenderRestoredTopRendersResolvableTopView confirms a resolvable
// restored view stays on top of the stack while its fetch is in flight. It
// no longer asserts on an "isRestore" flag directly: isRestore is now a
// plain function argument private to loadDetail's goroutine (see Task 4 of
// the implementation plan / the design doc), not an externally observable
// field — and the argument's effect (recording/pop-and-retry decisions)
// lives entirely inside a QueueUpdateDraw callback, which never runs in
// these tests since s.app.Run() is never called (see the waitFor helper's
// comment above for the same, pre-existing limitation on the async
// success/failure paths in general).
func TestRenderRestoredTopRendersResolvableTopView(t *testing.T) {
	registry := resource.NewRegistry()
	registry.Register(fakeResource{name: "workerpools"})
	s := New(registry)
	s.restoreFallback = "workerpools"
	s.stack.Push(View{ResourceName: "workerpools", Kind: DetailKind, SelectedID: "abc"})

	s.renderRestoredTop()

	top, ok := s.stack.Top()
	if !ok || top.ResourceName != "workerpools" || top.Kind != DetailKind || top.SelectedID != "abc" {
		t.Fatalf("expected the restored detail view to remain on top, got %+v (ok=%v)", top, ok)
	}
}

func TestIsStaleLoadDetectsSupersededDispatchToIdenticalTarget(t *testing.T) {
	s := New(resource.NewRegistry())

	// Simulate two dispatches to the IDENTICAL target — e.g. a pending
	// restore-replay for (task, "A") and a manual `:task A` navigation
	// racing it, per the bug this guards against. Each dispatch increments
	// loadGeneration and captures its own value, exactly as loadList/
	// loadDetail do internally.
	s.loadGeneration++
	gen1 := s.loadGeneration // the older, soon-to-be-superseded dispatch

	s.loadGeneration++
	gen2 := s.loadGeneration // the newer dispatch, targeting the same View

	if !s.isStaleLoad(gen1) {
		t.Fatalf("expected the older dispatch (gen1=%d) to be stale once a newer one (gen2=%d) started", gen1, gen2)
	}
	if s.isStaleLoad(gen2) {
		t.Fatalf("expected the newer dispatch (gen2=%d) to still be current", gen2)
	}
}

func TestLoadDetailIncrementsGenerationOnEachNavigationDispatch(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeResource{name: "task"}

	before := s.loadGeneration
	s.loadDetail(res, "A", true, true) // e.g. a restore-replay dispatch

	if s.loadGeneration != before+1 {
		t.Fatalf("expected loadGeneration to increment by 1, got %d -> %d", before, s.loadGeneration)
	}
	firstGen := s.loadGeneration

	s.loadDetail(res, "A", true, false) // e.g. a manual re-navigation to the identical target

	if s.loadGeneration != firstGen+1 {
		t.Fatalf("expected loadGeneration to increment again, got %d -> %d", firstGen, s.loadGeneration)
	}
	// The first dispatch's captured generation (firstGen) is now stale,
	// even though it targeted the exact same (resource, id) as the second —
	// this is precisely the case isTopView's View-equality check cannot
	// distinguish on its own.
	if !s.isStaleLoad(firstGen) {
		t.Fatalf("expected the first dispatch's generation to be stale after a second dispatch to the identical target")
	}
}

func TestLoadListIncrementsGenerationOnEachNavigationDispatch(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeResource{name: "workerpools"}

	before := s.loadGeneration
	s.loadList(res, "", "", true, false, true) // e.g. a restore-replay dispatch

	if s.loadGeneration != before+1 {
		t.Fatalf("expected loadGeneration to increment by 1, got %d -> %d", before, s.loadGeneration)
	}
	firstGen := s.loadGeneration

	s.loadList(res, "", "", true, false, false) // e.g. a manual re-navigation to the identical target

	if s.loadGeneration != firstGen+1 {
		t.Fatalf("expected loadGeneration to increment again, got %d -> %d", firstGen, s.loadGeneration)
	}
	if !s.isStaleLoad(firstGen) {
		t.Fatalf("expected the first dispatch's generation to be stale after a second dispatch to the identical target")
	}
}

func TestLoadDetailBackgroundRefreshDoesNotBumpGeneration(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeResource{name: "task"}

	s.loadDetail(res, "A", true, false) // a genuine navigation dispatch (isInitial=true)
	genAfterInitial := s.loadGeneration

	s.loadDetail(res, "A", false, false) // a background refresh tick (isInitial=false) — must NOT bump generation

	if s.loadGeneration != genAfterInitial {
		t.Fatalf("expected a background refresh (isInitial=false) not to bump loadGeneration, got %d -> %d", genAfterInitial, s.loadGeneration)
	}
	// Critically: the initial dispatch's captured generation must still be
	// considered current after the refresh tick, since nothing has
	// actually superseded it as a navigation target.
	if s.isStaleLoad(genAfterInitial) {
		t.Fatalf("expected the initial dispatch's generation to remain current after a mere background refresh tick")
	}
}

func TestLoadListBackgroundRefreshDoesNotBumpGeneration(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeResource{name: "workerpools"}

	s.loadList(res, "", "", true, false, false) // a genuine navigation dispatch (isInitial=true)
	genAfterInitial := s.loadGeneration

	s.loadList(res, "", "", false, true, false) // a background refresh tick (isInitial=false, forceRefresh=true, as Invalidate dispatches it)

	if s.loadGeneration != genAfterInitial {
		t.Fatalf("expected a background refresh (isInitial=false) not to bump loadGeneration, got %d -> %d", genAfterInitial, s.loadGeneration)
	}
	if s.isStaleLoad(genAfterInitial) {
		t.Fatalf("expected the initial dispatch's generation to remain current after a mere background refresh tick")
	}
}

func TestNextLoadGenerationInheritsForRefreshButBumpsForNavigation(t *testing.T) {
	s := New(resource.NewRegistry())

	gen1 := s.nextLoadGeneration(true) // initial navigation to A

	genRefresh := s.nextLoadGeneration(false) // a same-target refresh tick
	if genRefresh != gen1 {
		t.Fatalf("expected a refresh dispatch to inherit the current generation (%d), got %d", gen1, genRefresh)
	}

	gen2 := s.nextLoadGeneration(true) // navigate away to B
	gen3 := s.nextLoadGeneration(true) // navigate back to A — a fresh re-visit

	if gen2 == gen1 || gen3 == gen2 {
		t.Fatalf("expected each navigation dispatch to bump the generation, got gen1=%d gen2=%d gen3=%d", gen1, gen2, gen3)
	}

	// The old refresh's captured generation must now be stale relative to
	// the fresh re-visit to A — even though isTopView would match again —
	// since two navigation dispatches have incremented the generation since
	// it was captured.
	if !s.isStaleLoad(genRefresh) {
		t.Fatalf("expected the old refresh's generation (%d) to be stale after two subsequent navigation dispatches (current %d)", genRefresh, s.loadGeneration)
	}
}

func TestNextLoadGenerationRefreshRemainsCurrentWithoutInterveningNavigation(t *testing.T) {
	s := New(resource.NewRegistry())

	s.nextLoadGeneration(true)                // initial navigation to A
	genRefresh := s.nextLoadGeneration(false) // a same-target refresh tick, nothing navigated away in between

	if s.isStaleLoad(genRefresh) {
		t.Fatalf("expected a refresh's captured generation to remain current when nothing has navigated away in the meantime")
	}
}

func TestGoBackAtRootIsNoOp(t *testing.T) {
	registry := resource.NewRegistry()
	registry.Register(fakeResource{name: "workerpools"})
	s := New(registry)
	s.stack.Push(View{ResourceName: "workerpools", Kind: ListKind})

	s.goBack()

	if s.stack.Len() != 1 {
		t.Fatalf("expected the root view to remain on the stack, got length %d", s.stack.Len())
	}
	top, ok := s.stack.Top()
	if !ok || top.ResourceName != "workerpools" {
		t.Fatalf("expected root view to remain on top, got %+v (ok=%v)", top, ok)
	}
}

func TestGoBackPopsOneLevelWhenNotAtRoot(t *testing.T) {
	registry := resource.NewRegistry()
	registry.Register(fakeResource{name: "workerpools"})
	registry.Register(fakeResource{name: "workers"})
	s := New(registry)
	s.stack.Push(View{ResourceName: "workerpools", Kind: ListKind})
	s.stack.Push(View{ResourceName: "workers", Kind: ListKind, Scope: "pool-a"})

	s.goBack()

	if s.stack.Len() != 1 {
		t.Fatalf("expected exactly one view left on the stack, got %d", s.stack.Len())
	}
	top, ok := s.stack.Top()
	if !ok || top.ResourceName != "workerpools" {
		t.Fatalf("expected to be back at workerpools, got %+v (ok=%v)", top, ok)
	}
}

type fakeScopeActionsResource struct {
	fakeScopedResource
	actions []resource.DetailAction
}

func (f fakeScopeActionsResource) ScopeActions(scope string) []resource.DetailAction { return f.actions }

func TestRenderListPopulatesDetailActionsFromScopeActions(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeScopeActionsResource{
		fakeScopedResource: fakeScopedResource{fakeResource: fakeResource{name: "workers"}},
		actions: []resource.DetailAction{
			{Key: 'e', Label: "errors", Target: resource.NavTarget{ResourceName: "errors", ID: "pool-a"}},
		},
	}

	s.renderList(res, "pool-a", false)

	if len(s.currentDetailActions) != 1 || s.currentDetailActions[0].Key != 'e' {
		t.Fatalf("expected ScopeActions to populate currentDetailActions, got %+v", s.currentDetailActions)
	}
}

func TestRenderListLeavesDetailActionsNilWithoutScopeActions(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeResource{name: "workerpools"}

	s.renderList(res, "", false)

	if s.currentDetailActions != nil {
		t.Fatalf("expected nil currentDetailActions for a resource without ScopeActions, got %+v", s.currentDetailActions)
	}
}
