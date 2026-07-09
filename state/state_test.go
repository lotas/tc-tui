package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitize(t *testing.T) {
	cases := map[string]string{
		"https://stage.taskcluster.nonprod.cloudops.mozgcp.net": "stage.taskcluster.nonprod.cloudops.mozgcp.net",
		"https://taskcluster.example.com:443":                   "taskcluster.example.com-443",
		"http://taskcluster.example.com/":                       "taskcluster.example.com",
		"":                                                      "default",
	}

	for input, want := range cases {
		if got := sanitize(input); got != want {
			t.Errorf("sanitize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "state.json")

	want := State{
		Stack: []ViewState{
			{ResourceName: "workerpools", Kind: 0, Scope: ""},
			{ResourceName: "workers", Kind: 1, SelectedID: "abc", Scope: "poolId"},
		},
		SortByResource:   map[string]SortState{"workerpools": {Column: 1, Direction: 2}},
		FacetByResource:  map[string]string{"tasks": "pending"},
		FilterByResource: map[string]string{"workerpools": "proj-task"},
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got := Load(path)

	if len(got.Stack) != len(want.Stack) {
		t.Fatalf("Stack length = %d, want %d", len(got.Stack), len(want.Stack))
	}
	for i := range want.Stack {
		if got.Stack[i] != want.Stack[i] {
			t.Errorf("Stack[%d] = %+v, want %+v", i, got.Stack[i], want.Stack[i])
		}
	}
	if got.SortByResource["workerpools"] != want.SortByResource["workerpools"] {
		t.Errorf("SortByResource = %+v, want %+v", got.SortByResource, want.SortByResource)
	}
	if got.FacetByResource["tasks"] != want.FacetByResource["tasks"] {
		t.Errorf("FacetByResource = %+v, want %+v", got.FacetByResource, want.FacetByResource)
	}
	if got.FilterByResource["workerpools"] != want.FilterByResource["workerpools"] {
		t.Errorf("FilterByResource = %+v, want %+v", got.FilterByResource, want.FilterByResource)
	}
}

func TestLoadMissingFile(t *testing.T) {
	got := Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if len(got.Stack) != 0 || len(got.SortByResource) != 0 || len(got.FacetByResource) != 0 {
		t.Errorf("Load of missing file = %+v, want zero value", got)
	}
}

func TestLoadCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got := Load(path)
	if len(got.Stack) != 0 || len(got.SortByResource) != 0 || len(got.FacetByResource) != 0 {
		t.Errorf("Load of corrupt file = %+v, want zero value", got)
	}
}
