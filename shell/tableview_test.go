package shell

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/taskcluster/tc-tui/resource"
)

func TestTableViewSetDataRestoresSelectionByID(t *testing.T) {
	table := NewTableView()

	columns := []resource.Column{{Title: "WORKER POOL ID"}}
	rows := []resource.Row{
		{ID: "pool-a", Cells: []string{"pool-a"}},
		{ID: "pool-b", Cells: []string{"pool-b"}},
		{ID: "pool-c", Cells: []string{"pool-c"}},
	}

	table.SetData(columns, rows, SortState{})

	// Simulate the user having navigated to row 2 (pool-b). Table.Select
	// invokes the registered SetSelectionChangedFunc directly, so this has
	// the same effect on lastSelectedID as an arrow-key press would.
	table.Select(2, 0)

	// Re-render with the same rows, reordered, as a background refresh
	// would. The cursor should follow "pool-b" rather than staying pinned
	// to row 2.
	reordered := []resource.Row{
		{ID: "pool-b", Cells: []string{"pool-b"}},
		{ID: "pool-a", Cells: []string{"pool-a"}},
		{ID: "pool-c", Cells: []string{"pool-c"}},
	}
	table.SetData(columns, reordered, SortState{})

	row, _ := table.GetSelection()
	cell := table.GetCell(row, 0)
	got, ok := cell.GetReference().(resource.Row)
	if !ok || got.ID != "pool-b" {
		t.Fatalf("expected selection to follow pool-b, got row %d (id=%q, ok=%v)", row, got.ID, ok)
	}
}

func TestTableViewSetDataFallsBackWhenIDMissing(t *testing.T) {
	table := NewTableView()

	columns := []resource.Column{{Title: "WORKER POOL ID"}}
	rows := []resource.Row{
		{ID: "pool-a", Cells: []string{"pool-a"}},
		{ID: "pool-b", Cells: []string{"pool-b"}},
	}

	table.SetData(columns, rows, SortState{})
	table.Select(2, 0) // lastSelectedID = "pool-b"

	// Switch to a completely different resource - none of the old IDs are
	// present, so the selection should fall back to row 1 rather than
	// erroring or matching the wrong row.
	newRows := []resource.Row{
		{ID: "role-x", Cells: []string{"role-x"}},
		{ID: "role-y", Cells: []string{"role-y"}},
	}
	table.SetData(columns, newRows, SortState{})

	row, _ := table.GetSelection()
	if row != 1 {
		t.Fatalf("expected fallback to row 1, got row %d", row)
	}

	// And with an empty list, fall back to row 0.
	table.SetData(columns, nil, SortState{})
	row, _ = table.GetSelection()
	if row != 0 {
		t.Fatalf("expected fallback to row 0 for empty list, got row %d", row)
	}
}

func TestTableViewResetSelectionForcesFallbackToTop(t *testing.T) {
	table := NewTableView()

	columns := []resource.Column{{Title: "WORKER POOL ID"}}
	rows := []resource.Row{
		{ID: "pool-a", Cells: []string{"pool-a"}},
		{ID: "pool-b", Cells: []string{"pool-b"}},
		{ID: "pool-c", Cells: []string{"pool-c"}},
	}

	table.SetData(columns, rows, SortState{})
	table.Select(3, 0) // lastSelectedID = "pool-c"

	table.ResetSelection()

	// Same rows, same IDs still present - without ResetSelection, SetData
	// would restore the cursor to "pool-c". With it cleared, it should fall
	// back to the top row instead, as if this were a brand new render.
	table.SetData(columns, rows, SortState{Column: 0, Direction: SortAsc})

	row, _ := table.GetSelection()
	if row != 1 {
		t.Fatalf("expected fallback to top row 1 after ResetSelection, got row %d", row)
	}
}

func TestTableViewSetDataShowsSortIndicatorOnActiveColumn(t *testing.T) {
	table := NewTableView()

	columns := []resource.Column{{Title: "WORKER POOL ID"}, {Title: "PROVIDER"}}
	rows := []resource.Row{
		{ID: "pool-a", Cells: []string{"pool-a", "gcp"}},
	}

	table.SetData(columns, rows, SortState{Column: 1, Direction: SortAsc})

	got := table.GetCell(0, 1).Text
	if got != "PROVIDER ▲" {
		t.Fatalf("expected ascending indicator on sorted column, got %q", got)
	}

	unsorted := table.GetCell(0, 0).Text
	if unsorted != "WORKER POOL ID"+columnGap {
		t.Fatalf("expected no indicator on non-sorted column, got %q", unsorted)
	}
}

func TestTableViewSetDataShowsDescendingIndicator(t *testing.T) {
	table := NewTableView()

	columns := []resource.Column{{Title: "CAPACITY"}}
	rows := []resource.Row{
		{ID: "pool-a", Cells: []string{"3"}},
	}

	table.SetData(columns, rows, SortState{Column: 0, Direction: SortDesc})

	got := table.GetCell(0, 0).Text
	if got != "CAPACITY ▼" {
		t.Fatalf("expected descending indicator, got %q", got)
	}
}

func TestTableViewSetExpandColumnsDisablesWidthCap(t *testing.T) {
	table := NewTableView()

	columns := []resource.Column{{Title: "TASK ID", Width: 22}, {Title: "NAME"}}
	rows := []resource.Row{{ID: "task-a", Cells: []string{"task-a", "a very long task name"}}}

	table.SetData(columns, rows, SortState{})

	if table.ExpandColumns() {
		t.Fatalf("expected ExpandColumns to default to false")
	}
	if got := table.GetCell(0, 0).MaxWidth; got == 0 {
		t.Fatalf("expected the fixed-width column's header to have a width cap before expanding, got %d", got)
	}
	if got := table.GetCell(1, 0).MaxWidth; got == 0 {
		t.Fatalf("expected the fixed-width column's cell to have a width cap before expanding, got %d", got)
	}

	table.SetExpandColumns(true)
	if !table.ExpandColumns() {
		t.Fatalf("expected ExpandColumns to report true after SetExpandColumns(true)")
	}

	table.SetData(columns, rows, SortState{})

	header := table.GetCell(0, 0)
	if header.MaxWidth != 0 {
		t.Fatalf("expected no width cap on the header once expanded, got %d", header.MaxWidth)
	}
	if header.Expansion != 1 {
		t.Fatalf("expected the header to expand once truncation is disabled, got expansion %d", header.Expansion)
	}

	cell := table.GetCell(1, 0)
	if cell.MaxWidth != 0 {
		t.Fatalf("expected no width cap on the cell once expanded, got %d", cell.MaxWidth)
	}
	if cell.Expansion != 1 {
		t.Fatalf("expected the cell to expand once truncation is disabled, got expansion %d", cell.Expansion)
	}
}

func TestTableViewSelectingRowWithNavTargetPassesItThrough(t *testing.T) {
	table := NewTableView()

	target := &resource.NavTarget{ResourceName: "workerpools", ID: "pool-a", Kind: resource.NavDetail}
	columns := []resource.Column{{Title: "RESOURCE TYPE"}}
	rows := []resource.Row{
		{ID: "history-key-1", Cells: []string{"workerpools"}, NavTarget: target},
	}
	table.SetData(columns, rows, SortState{})

	var got resource.Row
	table.SetOnSelect(func(row resource.Row) {
		got = row
	})

	table.Select(1, 0)
	handler := table.InputHandler()
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})

	if got.NavTarget != target {
		t.Fatalf("expected onSelect to receive the row's NavTarget, got %+v", got.NavTarget)
	}
	if got.ID != "history-key-1" {
		t.Fatalf("expected onSelect to receive the row's ID, got %q", got.ID)
	}
}

func TestTableViewSelectingRowWithoutNavTargetLeavesItNil(t *testing.T) {
	table := NewTableView()

	columns := []resource.Column{{Title: "WORKER POOL ID"}}
	rows := []resource.Row{{ID: "pool-a", Cells: []string{"pool-a"}}}
	table.SetData(columns, rows, SortState{})

	var got resource.Row
	table.SetOnSelect(func(row resource.Row) {
		got = row
	})

	table.Select(1, 0)
	handler := table.InputHandler()
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})

	if got.NavTarget != nil {
		t.Fatalf("expected nil NavTarget for a normal row, got %+v", got.NavTarget)
	}
	if got.ID != "pool-a" {
		t.Fatalf("expected onSelect to receive ID pool-a, got %q", got.ID)
	}
}
