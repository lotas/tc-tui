package shell

import (
	"testing"

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

	table.SetData(columns, rows)

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
	table.SetData(columns, reordered)

	row, _ := table.GetSelection()
	cell := table.GetCell(row, 0)
	id, ok := cell.GetReference().(string)
	if !ok || id != "pool-b" {
		t.Fatalf("expected selection to follow pool-b, got row %d (id=%q, ok=%v)", row, id, ok)
	}
}

func TestTableViewSetDataFallsBackWhenIDMissing(t *testing.T) {
	table := NewTableView()

	columns := []resource.Column{{Title: "WORKER POOL ID"}}
	rows := []resource.Row{
		{ID: "pool-a", Cells: []string{"pool-a"}},
		{ID: "pool-b", Cells: []string{"pool-b"}},
	}

	table.SetData(columns, rows)
	table.Select(2, 0) // lastSelectedID = "pool-b"

	// Switch to a completely different resource - none of the old IDs are
	// present, so the selection should fall back to row 1 rather than
	// erroring or matching the wrong row.
	newRows := []resource.Row{
		{ID: "role-x", Cells: []string{"role-x"}},
		{ID: "role-y", Cells: []string{"role-y"}},
	}
	table.SetData(columns, newRows)

	row, _ := table.GetSelection()
	if row != 1 {
		t.Fatalf("expected fallback to row 1, got row %d", row)
	}

	// And with an empty list, fall back to row 0.
	table.SetData(columns, nil)
	row, _ = table.GetSelection()
	if row != 0 {
		t.Fatalf("expected fallback to row 0 for empty list, got row %d", row)
	}
}
