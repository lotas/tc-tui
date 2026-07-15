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

func TestSortRowsDescendingWithDuplicateValuesIsStable(t *testing.T) {
	rows := []resource.Row{
		{ID: "first", Cells: []string{"same"}},
		{ID: "second", Cells: []string{"same"}},
		{ID: "third", Cells: []string{"same"}},
		{ID: "fourth", Cells: []string{"zzz"}},
	}

	sorted := SortRows(rows, SortState{Column: 0, Direction: SortDesc})

	// "zzz" > "same", so fourth sorts first; the three tied "same" rows must
	// keep their original relative order (stable), not merely "some order" —
	// the pre-fix comparator violated strict-weak-ordering for equal keys,
	// which sort.SliceStable does not define as safe.
	if len(sorted) != 4 || sorted[0].ID != "fourth" || sorted[1].ID != "first" ||
		sorted[2].ID != "second" || sorted[3].ID != "third" {
		t.Fatalf("expected [fourth,first,second,third], got %+v", sorted)
	}
}

func TestSortRowsByteSizeAscendingSortsByMagnitudeNotText(t *testing.T) {
	// Mirrors resource.formatBytes' rendering (e.g. TaskArtifactsResource's
	// SIZE column) — lexical order would wrongly put "2.0 KiB" before
	// "500 B" and "4.2 MiB" before "800 KiB".
	rows := []resource.Row{
		{ID: "a", Cells: []string{"4.2 MiB"}},
		{ID: "b", Cells: []string{"500 B"}},
		{ID: "c", Cells: []string{"2.0 KiB"}},
		{ID: "d", Cells: []string{"800 KiB"}},
	}

	sorted := SortRows(rows, SortState{Column: 0, Direction: SortAsc})
	if sorted[0].ID != "b" || sorted[1].ID != "c" || sorted[2].ID != "d" || sorted[3].ID != "a" {
		t.Fatalf("expected magnitude order [b,c,d,a] (500B,2.0KiB,800KiB,4.2MiB), got %+v", sorted)
	}
}

func TestSortRowsNumericDescendingWithDuplicateValuesIsStable(t *testing.T) {
	rows := []resource.Row{
		{ID: "first", Cells: []string{"5"}},
		{ID: "second", Cells: []string{"5"}},
		{ID: "third", Cells: []string{"9"}},
	}

	sorted := SortRows(rows, SortState{Column: 0, Direction: SortDesc})

	if len(sorted) != 3 || sorted[0].ID != "third" || sorted[1].ID != "first" || sorted[2].ID != "second" {
		t.Fatalf("expected [third,first,second], got %+v", sorted)
	}
}
