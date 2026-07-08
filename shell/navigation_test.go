package shell

import (
	"testing"

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
