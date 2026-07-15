package shell

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/taskcluster/tc-tui/resource"
)

func TestSplitCommandNameOnly(t *testing.T) {
	name, scope := splitCommand("workerpools")
	if name != "workerpools" || scope != "" {
		t.Fatalf("unexpected split: name=%q scope=%q", name, scope)
	}
}

func TestSplitCommandNameAndScope(t *testing.T) {
	name, scope := splitCommand("workers gcp/linux-b-gpu")
	if name != "workers" || scope != "gcp/linux-b-gpu" {
		t.Fatalf("unexpected split: name=%q scope=%q", name, scope)
	}
}

func TestSplitCommandCollapsesExtraWhitespace(t *testing.T) {
	name, scope := splitCommand("  workers   gcp/linux-b-gpu  ")
	if name != "workers" || scope != "gcp/linux-b-gpu" {
		t.Fatalf("unexpected split: name=%q scope=%q", name, scope)
	}
}

func TestSplitCommandEmptyInput(t *testing.T) {
	name, scope := splitCommand("")
	if name != "" || scope != "" {
		t.Fatalf("unexpected split: name=%q scope=%q", name, scope)
	}
}

func TestSplitCommandWhitespaceOnlyInput(t *testing.T) {
	name, scope := splitCommand("   ")
	if name != "" || scope != "" {
		t.Fatalf("unexpected split: name=%q scope=%q", name, scope)
	}
}

func TestRenderHeaderHintsShowsTabHintWhenFaceted(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentFaceted = fakeFacetedColumn{column: 0, options: []string{"aws"}}

	s.renderHeaderHints()

	text := s.headerHint.GetText(false)
	if !strings.Contains(text, "Tab") || !strings.Contains(text, "Shift+Tab") {
		t.Fatalf("expected Tab/Shift+Tab hint, got %q", text)
	}
}

func TestRenderHeaderHintsOmitsTabHintWhenNotFaceted(t *testing.T) {
	s := New(resource.NewRegistry())

	s.renderHeaderHints()

	text := s.headerHint.GetText(false)
	if strings.Contains(text, "Shift+Tab") {
		t.Fatalf("expected no Tab hint for a non-faceted resource, got %q", text)
	}
}

func TestHeaderRowsNeededNeverShrinksBelowThree(t *testing.T) {
	for _, hintCount := range []int{0, 1, 5, 9} {
		if got := headerRowsNeeded(hintCount); got != 3 {
			t.Errorf("headerRowsNeeded(%d) = %d, want 3", hintCount, got)
		}
	}
}

func TestHeaderRowsNeededGrowsWithHintCount(t *testing.T) {
	cases := []struct {
		hintCount int
		want      int
	}{
		{10, 4}, // one hint past the 3x3 grid needs a 4th row
		{12, 4},
		{13, 5},
	}
	for _, c := range cases {
		if got := headerRowsNeeded(c.hintCount); got != c.want {
			t.Errorf("headerRowsNeeded(%d) = %d, want %d", c.hintCount, got, c.want)
		}
	}
}

func TestRenderHeaderHintsShowsFilterOnListView(t *testing.T) {
	s := New(resource.NewRegistry())
	s.stack.Push(View{ResourceName: "workerpools", Kind: ListKind})

	s.renderHeaderHints()

	text := s.headerHint.GetText(false)
	if !strings.Contains(text, "filter") {
		t.Fatalf("expected filter hint on a list view, got %q", text)
	}
}

func TestRenderHeaderHintsOmitsFilterOnDetailView(t *testing.T) {
	s := New(resource.NewRegistry())
	s.stack.Push(View{ResourceName: "workerpools", Kind: DetailKind, SelectedID: "gcp/pool-a"})

	s.renderHeaderHints()

	text := s.headerHint.GetText(false)
	if strings.Contains(text, "filter") {
		t.Fatalf("expected no filter hint on a detail view, got %q", text)
	}
}

func TestRenderHeaderHintsRendersDetailActionsWithoutBrackets(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentDetailActions = []resource.DetailAction{
		{Key: 'w', Label: "workers"},
	}

	s.renderHeaderHints()

	text := s.headerHint.GetText(false)
	if !strings.Contains(text, "[yellow]w[white] workers") {
		t.Fatalf("expected bracket-free detail action hint, got %q", text)
	}
	if strings.Contains(text, "<w>") {
		t.Fatalf("expected no angle-bracket detail action hint, got %q", text)
	}
}

func TestRenderBreadcrumbsEmptyStack(t *testing.T) {
	s := New(resource.NewRegistry())

	s.renderBreadcrumbs()

	text := strings.TrimSpace(s.footerBreadcrumb.GetText(false))
	if text != "" {
		t.Fatalf("expected empty breadcrumb for empty stack, got %q", text)
	}
}

func TestRenderBreadcrumbsListView(t *testing.T) {
	s := New(resource.NewRegistry())
	s.stack.Push(View{ResourceName: "workerpools", Kind: ListKind})

	s.renderBreadcrumbs()

	text := strings.TrimSpace(s.footerBreadcrumb.GetText(false))
	if text != "workerpools" {
		t.Fatalf("unexpected breadcrumb: %q", text)
	}
}

func TestRenderBreadcrumbsScopedListView(t *testing.T) {
	s := New(resource.NewRegistry())
	s.stack.Push(View{ResourceName: "workers", Kind: ListKind, Scope: "gecko-3/b-linux"})

	s.renderBreadcrumbs()

	text := strings.TrimSpace(s.footerBreadcrumb.GetText(false))
	if text != "workers (gecko-3/b-linux)" {
		t.Fatalf("unexpected breadcrumb: %q", text)
	}
}

func TestRenderBreadcrumbsMultiLevelStack(t *testing.T) {
	s := New(resource.NewRegistry())
	s.stack.Push(View{ResourceName: "workerpools", Kind: ListKind})
	s.stack.Push(View{ResourceName: "workers", Kind: ListKind, Scope: "gecko-3/b-linux"})
	s.stack.Push(View{ResourceName: "worker", Kind: DetailKind, SelectedID: "worker-1"})

	s.renderBreadcrumbs()

	text := strings.TrimSpace(s.footerBreadcrumb.GetText(false))
	want := "workerpools › workers (gecko-3/b-linux) › worker:worker-1"
	if text != want {
		t.Fatalf("unexpected breadcrumb: got %q, want %q", text, want)
	}
}

func TestHandleFooterInputDonePromptWithIDNavigatesAndCloses(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeDirectLookupResource{fakeResource: fakeResource{name: "task"}, label: "task id"}
	s.footerMode = footerPrompt
	s.pendingLookup = res
	s.footerInput.SetText("task-1")

	s.handleFooterInputDone(tcell.KeyEnter)

	if s.footerMode != footerIdle {
		t.Fatalf("expected footer to close, got mode %v", s.footerMode)
	}
	if s.pendingLookup != nil {
		t.Fatalf("expected pendingLookup cleared, got %+v", s.pendingLookup)
	}
	top, ok := s.stack.Top()
	if !ok || top.Kind != DetailKind || top.SelectedID != "task-1" || top.ResourceName != "task" {
		t.Fatalf("unexpected top view: %+v (ok=%v)", top, ok)
	}
}

func TestHandleFooterInputDonePromptWithEmptyTextStaysOpen(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeDirectLookupResource{fakeResource: fakeResource{name: "task"}, label: "task id"}
	s.footerMode = footerPrompt
	s.pendingLookup = res
	s.footerInput.SetText("   ")

	s.handleFooterInputDone(tcell.KeyEnter)

	if s.footerMode != footerPrompt {
		t.Fatalf("expected prompt to stay open, got mode %v", s.footerMode)
	}
	if s.pendingLookup == nil {
		t.Fatalf("expected pendingLookup to remain set")
	}
}

func TestHandleFooterInputDonePromptEscapeCancels(t *testing.T) {
	s := New(resource.NewRegistry())
	res := fakeDirectLookupResource{fakeResource: fakeResource{name: "task"}, label: "task id"}
	s.footerMode = footerPrompt
	s.pendingLookup = res

	s.handleFooterInputDone(tcell.KeyEscape)

	if s.footerMode != footerIdle {
		t.Fatalf("expected footer to close, got mode %v", s.footerMode)
	}
	if s.pendingLookup != nil {
		t.Fatalf("expected pendingLookup cleared, got %+v", s.pendingLookup)
	}
}

func TestHandleFooterInputDoneFilterEnterPersistsPerResource(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentListResource = "workerpools"
	s.footerMode = footerFilter
	s.footerInput.SetText("proj-task")

	s.handleFooterInputDone(tcell.KeyEnter)

	if s.filterQuery != "proj-task" {
		t.Fatalf("expected filterQuery %q, got %q", "proj-task", s.filterQuery)
	}
	if got := s.filterByResource["workerpools"]; got != "proj-task" {
		t.Fatalf("expected filterByResource[workerpools] = %q, got %q", "proj-task", got)
	}
}

func TestHandleFooterInputDoneFilterEscapeClearsPersistedValue(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentListResource = "workerpools"
	s.filterByResource["workerpools"] = "proj-task"
	s.filterQuery = "proj-task"
	s.footerMode = footerFilter

	s.handleFooterInputDone(tcell.KeyEscape)

	if s.filterQuery != "" {
		t.Fatalf("expected filterQuery cleared, got %q", s.filterQuery)
	}
	if got := s.filterByResource["workerpools"]; got != "" {
		t.Fatalf("expected filterByResource[workerpools] cleared, got %q", got)
	}
}
