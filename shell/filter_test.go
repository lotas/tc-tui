package shell

import (
	"testing"

	"github.com/taskcluster/tc-tui/resource"
)

func TestFilterRowsEmptyQueryReturnsAll(t *testing.T) {
	rows := []resource.Row{
		{ID: "a", Cells: []string{"alpha"}},
		{ID: "b", Cells: []string{"beta"}},
	}

	filtered := FilterRows(rows, "")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(filtered))
	}
}

func TestFilterRowsMatchesAnyCellCaseInsensitive(t *testing.T) {
	rows := []resource.Row{
		{ID: "a", Cells: []string{"gcp", "proj/pool-a"}},
		{ID: "b", Cells: []string{"aws", "proj/pool-b"}},
	}

	filtered := FilterRows(rows, "GCP")
	if len(filtered) != 1 || filtered[0].ID != "a" {
		t.Fatalf("unexpected filtered rows: %+v", filtered)
	}
}

func TestFilterRowsNoMatches(t *testing.T) {
	rows := []resource.Row{
		{ID: "a", Cells: []string{"gcp"}},
	}

	filtered := FilterRows(rows, "azure")
	if len(filtered) != 0 {
		t.Fatalf("expected no rows, got %+v", filtered)
	}
}

func TestMergeRowsByIDReplacesMatchingRowsOnly(t *testing.T) {
	base := []resource.Row{
		{ID: "a", Cells: []string{"alpha", "..."}},
		{ID: "b", Cells: []string{"beta", "..."}},
		{ID: "c", Cells: []string{"gamma", "..."}},
	}
	updates := []resource.Row{
		{ID: "a", Cells: []string{"alpha", "42"}},
		{ID: "c", Cells: []string{"gamma", "7"}},
	}

	merged := mergeRowsByID(base, updates)

	if len(merged) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(merged))
	}
	if merged[0].Cells[1] != "42" {
		t.Fatalf("expected row a to be updated, got %+v", merged[0])
	}
	if merged[1].Cells[1] != "..." {
		t.Fatalf("expected row b (not in updates) to be untouched, got %+v", merged[1])
	}
	if merged[2].Cells[1] != "7" {
		t.Fatalf("expected row c to be updated, got %+v", merged[2])
	}
}

func TestMergeRowsByIDPreservesBaseOrder(t *testing.T) {
	base := []resource.Row{
		{ID: "a", Cells: []string{"1"}},
		{ID: "b", Cells: []string{"2"}},
	}
	updates := []resource.Row{
		{ID: "b", Cells: []string{"2-updated"}},
		{ID: "a", Cells: []string{"1-updated"}},
	}

	merged := mergeRowsByID(base, updates)

	if merged[0].ID != "a" || merged[1].ID != "b" {
		t.Fatalf("expected base order preserved, got %+v", merged)
	}
}
