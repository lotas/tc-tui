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
