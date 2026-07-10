package shell

import (
	"testing"
	"time"

	"github.com/taskcluster/tc-tui/resource"
	"github.com/taskcluster/tc-tui/state"
)

func TestExportRestoreStateRoundTrips(t *testing.T) {
	s := New(resource.NewRegistry())
	s.stack.Push(View{ResourceName: "workerpools", Kind: ListKind})
	s.stack.Push(View{ResourceName: "workers", Kind: DetailKind, SelectedID: "abc", Scope: "poolId"})
	s.sortByResource["workerpools"] = SortState{Column: 1, Direction: SortDesc}
	s.facetByResource["tasks"] = "pending"
	s.filterByResource["workerpools"] = "proj-task"

	exported := s.ExportState()

	restored := New(resource.NewRegistry())
	restored.RestoreState(exported)

	if got := restored.stack.Views(); len(got) != 2 ||
		got[0] != (View{ResourceName: "workerpools", Kind: ListKind}) ||
		got[1] != (View{ResourceName: "workers", Kind: DetailKind, SelectedID: "abc", Scope: "poolId"}) {
		t.Fatalf("stack did not round-trip, got %+v", got)
	}
	if got := restored.sortByResource["workerpools"]; got != (SortState{Column: 1, Direction: SortDesc}) {
		t.Fatalf("sortByResource did not round-trip, got %+v", got)
	}
	if got := restored.facetByResource["tasks"]; got != "pending" {
		t.Fatalf("facetByResource did not round-trip, got %q", got)
	}
	if got := restored.filterByResource["workerpools"]; got != "proj-task" {
		t.Fatalf("filterByResource did not round-trip, got %q", got)
	}
}

func TestRestoreStateThenExportIsStable(t *testing.T) {
	st := state.State{
		Stack: []state.ViewState{
			{ResourceName: "roles", Kind: 0},
		},
		SortByResource:   map[string]state.SortState{"roles": {Column: 2, Direction: 1}},
		FacetByResource:  map[string]string{"roles": "aws"},
		FilterByResource: map[string]string{"roles": "proj-task"},
	}

	s := New(resource.NewRegistry())
	s.RestoreState(st)

	if got := s.ExportState(); len(got.Stack) != 1 || got.Stack[0] != st.Stack[0] ||
		got.SortByResource["roles"] != st.SortByResource["roles"] ||
		got.FacetByResource["roles"] != st.FacetByResource["roles"] ||
		got.FilterByResource["roles"] != st.FilterByResource["roles"] {
		t.Fatalf("ExportState after RestoreState = %+v, want %+v", got, st)
	}
}

func TestExportRestoreStateRoundTripsHistory(t *testing.T) {
	s := New(resource.NewRegistry())
	rec := &fakeHistoryRecorder{}
	s.historyRecorder = rec
	rec.entries = []resource.HistoryEntry{
		{
			ResourceName: "workers", Kind: 1, SelectedID: "worker-1",
			Title:     "Worker :: worker-1",
			VisitedAt: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
		},
	}

	exported := s.ExportState()

	restored := New(resource.NewRegistry())
	restoredRec := &fakeHistoryRecorder{}
	restored.historyRecorder = restoredRec

	restored.RestoreState(exported)

	got := restoredRec.Entries()
	if len(got) != 1 || got[0].ResourceName != "workers" || got[0].SelectedID != "worker-1" || got[0].Title != "Worker :: worker-1" {
		t.Fatalf("expected history to round-trip through Export/RestoreState, got %+v", got)
	}
	if got[0].Kind != 1 || got[0].Scope != "" || !got[0].VisitedAt.Equal(time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected Kind/Scope/VisitedAt to round-trip too, got %+v", got[0])
	}
}
