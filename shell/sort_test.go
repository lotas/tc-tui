package shell

import (
	"testing"

	"github.com/taskcluster/tc-tui/resource"
)

func TestSortRowsNoneReturnsUnchanged(t *testing.T) {
	rows := []resource.Row{
		{ID: "b", Cells: []string{"b"}},
		{ID: "a", Cells: []string{"a"}},
	}

	sorted := SortRows(rows, SortState{Column: 0, Direction: SortNone})
	if sorted[0].ID != "b" || sorted[1].ID != "a" {
		t.Fatalf("expected unchanged order, got %+v", sorted)
	}
}

func TestSortRowsTextAscendingCaseInsensitive(t *testing.T) {
	rows := []resource.Row{
		{ID: "a", Cells: []string{"Beta"}},
		{ID: "b", Cells: []string{"alpha"}},
		{ID: "c", Cells: []string{"Gamma"}},
	}

	sorted := SortRows(rows, SortState{Column: 0, Direction: SortAsc})
	if sorted[0].ID != "b" || sorted[1].ID != "a" || sorted[2].ID != "c" {
		t.Fatalf("expected [b,a,c] order, got %+v", sorted)
	}
}

func TestSortRowsTextDescending(t *testing.T) {
	rows := []resource.Row{
		{ID: "a", Cells: []string{"alpha"}},
		{ID: "b", Cells: []string{"beta"}},
	}

	sorted := SortRows(rows, SortState{Column: 0, Direction: SortDesc})
	if sorted[0].ID != "b" || sorted[1].ID != "a" {
		t.Fatalf("expected [b,a] order, got %+v", sorted)
	}
}

func TestSortRowsNumericAscending(t *testing.T) {
	rows := []resource.Row{
		{ID: "a", Cells: []string{"10"}},
		{ID: "b", Cells: []string{"2"}},
		{ID: "c", Cells: []string{"3"}},
	}

	sorted := SortRows(rows, SortState{Column: 0, Direction: SortAsc})
	if sorted[0].ID != "b" || sorted[1].ID != "c" || sorted[2].ID != "a" {
		t.Fatalf("expected numeric order [b,c,a] (2,3,10), got %+v", sorted)
	}
}

func TestSortRowsNumericDescendingWithPadding(t *testing.T) {
	// Mirrors resource.WorkerPoolsResource's fmt.Sprintf("%10d", ...) padding.
	rows := []resource.Row{
		{ID: "a", Cells: []string{"         3"}},
		{ID: "b", Cells: []string{"        10"}},
		{ID: "c", Cells: []string{"         1"}},
	}

	sorted := SortRows(rows, SortState{Column: 0, Direction: SortDesc})
	if sorted[0].ID != "b" || sorted[1].ID != "a" || sorted[2].ID != "c" {
		t.Fatalf("expected descending numeric order [b,a,c] (10,3,1), got %+v", sorted)
	}
}

func TestSortRowsMixedColumnFallsBackToText(t *testing.T) {
	rows := []resource.Row{
		{ID: "a", Cells: []string{"10"}},
		{ID: "b", Cells: []string{"n/a"}},
		{ID: "c", Cells: []string{"2"}},
	}

	sorted := SortRows(rows, SortState{Column: 0, Direction: SortAsc})
	// Text order: "10" < "2" < "n/a"
	if sorted[0].ID != "a" || sorted[1].ID != "c" || sorted[2].ID != "b" {
		t.Fatalf("expected text-order [a,c,b], got %+v", sorted)
	}
}

func TestSortRowsStablePreservesOriginalOrderForTies(t *testing.T) {
	rows := []resource.Row{
		{ID: "first", Cells: []string{"same"}},
		{ID: "second", Cells: []string{"same"}},
		{ID: "third", Cells: []string{"same"}},
	}

	sorted := SortRows(rows, SortState{Column: 0, Direction: SortAsc})
	if sorted[0].ID != "first" || sorted[1].ID != "second" || sorted[2].ID != "third" {
		t.Fatalf("expected stable original order, got %+v", sorted)
	}
}

func TestSortRowsOutOfRangeColumnReturnsUnchanged(t *testing.T) {
	rows := []resource.Row{
		{ID: "a", Cells: []string{"x"}},
	}

	sorted := SortRows(rows, SortState{Column: 5, Direction: SortAsc})
	if len(sorted) != 1 || sorted[0].ID != "a" {
		t.Fatalf("expected unchanged single row, got %+v", sorted)
	}
}

func TestSortRowsEmptyDoesNotPanic(t *testing.T) {
	sorted := SortRows(nil, SortState{Column: 0, Direction: SortAsc})
	if len(sorted) != 0 {
		t.Fatalf("expected empty result, got %+v", sorted)
	}
}
