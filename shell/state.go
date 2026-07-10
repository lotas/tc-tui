package shell

import (
	"github.com/taskcluster/tc-tui/resource"
	"github.com/taskcluster/tc-tui/state"
)

// ExportState snapshots the current navigation stack, sort choices,
// facet-tab choices, filter queries, and history log for persistence.
func (s *Shell) ExportState() state.State {
	st := state.State{
		SortByResource:   make(map[string]state.SortState, len(s.sortByResource)),
		FacetByResource:  make(map[string]string, len(s.facetByResource)),
		FilterByResource: make(map[string]string, len(s.filterByResource)),
	}

	for _, v := range s.stack.Views() {
		st.Stack = append(st.Stack, state.ViewState{
			ResourceName: v.ResourceName,
			Kind:         int(v.Kind),
			SelectedID:   v.SelectedID,
			Scope:        v.Scope,
		})
	}

	for name, ss := range s.sortByResource {
		st.SortByResource[name] = state.SortState{Column: ss.Column, Direction: int(ss.Direction)}
	}

	for name, val := range s.facetByResource {
		st.FacetByResource[name] = val
	}

	for name, val := range s.filterByResource {
		st.FilterByResource[name] = val
	}

	if s.historyRecorder != nil {
		for _, e := range s.historyRecorder.Entries() {
			st.History = append(st.History, state.HistoryEntry{
				ResourceName: e.ResourceName,
				Kind:         e.Kind,
				SelectedID:   e.SelectedID,
				Scope:        e.Scope,
				Title:        e.Title,
				VisitedAt:    e.VisitedAt,
			})
		}
	}

	return st
}

// RestoreState loads a previously-exported snapshot into the (still empty)
// stack and maps. Must be called before Start.
func (s *Shell) RestoreState(st state.State) {
	for _, v := range st.Stack {
		s.stack.Push(View{
			ResourceName: v.ResourceName,
			Kind:         ViewKind(v.Kind),
			SelectedID:   v.SelectedID,
			Scope:        v.Scope,
		})
	}

	for name, ss := range st.SortByResource {
		s.sortByResource[name] = SortState{Column: ss.Column, Direction: SortDirection(ss.Direction)}
	}

	for name, val := range st.FacetByResource {
		s.facetByResource[name] = val
	}

	for name, val := range st.FilterByResource {
		s.filterByResource[name] = val
	}

	if s.historyRecorder != nil && len(st.History) > 0 {
		entries := make([]resource.HistoryEntry, len(st.History))
		for i, e := range st.History {
			entries[i] = resource.HistoryEntry{
				ResourceName: e.ResourceName,
				Kind:         e.Kind,
				SelectedID:   e.SelectedID,
				Scope:        e.Scope,
				Title:        e.Title,
				VisitedAt:    e.VisitedAt,
			}
		}
		s.historyRecorder.Restore(entries)
	}
}
