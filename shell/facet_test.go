package shell

import (
	"testing"
	"time"

	"github.com/taskcluster/tc-tui/resource"
)

type fakeFacetedColumn struct {
	column  int
	options []string
}

func (f fakeFacetedColumn) Name() string                            { return "test" }
func (f fakeFacetedColumn) Aliases() []string                       { return nil }
func (f fakeFacetedColumn) Description() string                     { return "" }
func (f fakeFacetedColumn) Columns() []resource.Column              { return nil }
func (f fakeFacetedColumn) List() ([]resource.Row, error)           { return nil, nil }
func (f fakeFacetedColumn) Describe(id string) (resource.Detail, error) { return resource.Detail{}, nil }
func (f fakeFacetedColumn) RefreshInterval() time.Duration          { return 0 }
func (f fakeFacetedColumn) FacetColumn() int                         { return f.column }
func (f fakeFacetedColumn) FacetOptions(rows []resource.Row) []string { return f.options }

func TestFilterByFacetNilFacetedReturnsUnchanged(t *testing.T) {
	rows := []resource.Row{{ID: "a", Cells: []string{"gcp"}}}

	filtered := FilterByFacet(rows, nil, "gcp")
	if len(filtered) != 1 {
		t.Fatalf("expected unchanged rows, got %+v", filtered)
	}
}

func TestFilterByFacetEmptyValueReturnsUnchanged(t *testing.T) {
	rows := []resource.Row{
		{ID: "a", Cells: []string{"gcp"}},
		{ID: "b", Cells: []string{"aws"}},
	}
	faceted := fakeFacetedColumn{column: 0}

	filtered := FilterByFacet(rows, faceted, "")
	if len(filtered) != 2 {
		t.Fatalf("expected unchanged rows for \"All\", got %+v", filtered)
	}
}

func TestFilterByFacetMatchesColumnValue(t *testing.T) {
	rows := []resource.Row{
		{ID: "a", Cells: []string{"gcp"}},
		{ID: "b", Cells: []string{"aws"}},
		{ID: "c", Cells: []string{"gcp"}},
	}
	faceted := fakeFacetedColumn{column: 0}

	filtered := FilterByFacet(rows, faceted, "gcp")
	if len(filtered) != 2 || filtered[0].ID != "a" || filtered[1].ID != "c" {
		t.Fatalf("unexpected filtered rows: %+v", filtered)
	}
}

func TestFilterByFacetOutOfRangeColumnReturnsNoRows(t *testing.T) {
	rows := []resource.Row{{ID: "a", Cells: []string{"gcp"}}}
	faceted := fakeFacetedColumn{column: 5}

	filtered := FilterByFacet(rows, faceted, "gcp")
	if len(filtered) != 0 {
		t.Fatalf("expected no rows for out-of-range column, got %+v", filtered)
	}
}

func TestClientFacetTabsIncludesAllAndCounts(t *testing.T) {
	rows := []resource.Row{
		{ID: "a", Cells: []string{"gcp"}},
		{ID: "b", Cells: []string{"aws"}},
		{ID: "c", Cells: []string{"gcp"}},
	}
	faceted := fakeFacetedColumn{column: 0, options: []string{"aws", "gcp"}}

	tabs := ClientFacetTabs(faceted, rows)

	want := []FacetTab{{Value: "", Count: 3}, {Value: "aws", Count: 1}, {Value: "gcp", Count: 2}}
	if len(tabs) != len(want) {
		t.Fatalf("expected %d tabs, got %+v", len(want), tabs)
	}
	for i, w := range want {
		if tabs[i] != w {
			t.Fatalf("tab %d: expected %+v, got %+v", i, w, tabs[i])
		}
	}
}

func TestClientFacetTabsEmptyRows(t *testing.T) {
	faceted := fakeFacetedColumn{column: 0, options: []string{"aws", "gcp"}}

	tabs := ClientFacetTabs(faceted, nil)

	want := []FacetTab{{Value: "", Count: 0}, {Value: "aws", Count: 0}, {Value: "gcp", Count: 0}}
	if len(tabs) != len(want) {
		t.Fatalf("expected %d tabs, got %+v", len(want), tabs)
	}
	for i, w := range want {
		if tabs[i] != w {
			t.Fatalf("tab %d: expected %+v, got %+v", i, w, tabs[i])
		}
	}
}

type fakeServerFaceted struct {
	options []string
}

func (f fakeServerFaceted) Name() string                            { return "test" }
func (f fakeServerFaceted) Aliases() []string                       { return nil }
func (f fakeServerFaceted) Description() string                     { return "" }
func (f fakeServerFaceted) Columns() []resource.Column              { return nil }
func (f fakeServerFaceted) List() ([]resource.Row, error)           { return nil, nil }
func (f fakeServerFaceted) Describe(id string) (resource.Detail, error) { return resource.Detail{}, nil }
func (f fakeServerFaceted) RefreshInterval() time.Duration          { return 0 }
func (f fakeServerFaceted) FacetOptions() []string { return f.options }
func (f fakeServerFaceted) FacetList(scope, value string) ([]resource.Row, error) {
	return nil, nil
}
func (f fakeServerFaceted) FacetCounts(scope string) (map[string]int, error) { return nil, nil }

func TestServerFacetTabsUsesCountsMapNoAllTab(t *testing.T) {
	faceted := fakeServerFaceted{options: []string{"running", "stopped"}}
	counts := map[string]int{"running": 12, "stopped": 14302}

	tabs := ServerFacetTabs(faceted, counts)

	want := []FacetTab{{Value: "running", Count: 12}, {Value: "stopped", Count: 14302}}
	if len(tabs) != len(want) || tabs[0] != want[0] || tabs[1] != want[1] {
		t.Fatalf("unexpected tabs: %+v", tabs)
	}
}

func TestServerFacetTabsMissingCountDefaultsToZero(t *testing.T) {
	faceted := fakeServerFaceted{options: []string{"running", "requested"}}
	counts := map[string]int{"running": 5}

	tabs := ServerFacetTabs(faceted, counts)

	if tabs[1].Value != "requested" || tabs[1].Count != 0 {
		t.Fatalf("expected requested count 0, got %+v", tabs[1])
	}
}

func TestCycleFacetValueForwardWraps(t *testing.T) {
	tabs := []FacetTab{{Value: "a"}, {Value: "b"}, {Value: "c"}}

	if got := cycleFacetValue(tabs, "c", 1); got != "a" {
		t.Fatalf("expected wrap to \"a\", got %q", got)
	}
	if got := cycleFacetValue(tabs, "a", 1); got != "b" {
		t.Fatalf("expected \"b\", got %q", got)
	}
}

func TestCycleFacetValueBackwardWraps(t *testing.T) {
	tabs := []FacetTab{{Value: "a"}, {Value: "b"}, {Value: "c"}}

	if got := cycleFacetValue(tabs, "a", -1); got != "c" {
		t.Fatalf("expected wrap to \"c\", got %q", got)
	}
}

func TestCycleFacetValueUnknownCurrentTreatsAsFirst(t *testing.T) {
	tabs := []FacetTab{{Value: "a"}, {Value: "b"}}

	if got := cycleFacetValue(tabs, "unknown", 1); got != "b" {
		t.Fatalf("expected \"b\", got %q", got)
	}
}

func TestCycleFacetValueEmptyTabsReturnsCurrentUnchanged(t *testing.T) {
	if got := cycleFacetValue(nil, "x", 1); got != "x" {
		t.Fatalf("expected unchanged \"x\", got %q", got)
	}
}
