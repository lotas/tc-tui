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
