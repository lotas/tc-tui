package shell

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/taskcluster/tc-tui/resource"
)

func TestGlobalInputCaptureQuitKeyIsHandledInNavigableViews(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*Shell)
	}{
		{
			name:      "content",
			configure: func(*Shell) {},
		},
		{
			name: "help",
			configure: func(s *Shell) {
				s.helpOpen = true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(resource.NewRegistry())
			tt.configure(s)

			event := tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)
			if got := s.globalInputCapture(event); got != nil {
				t.Fatalf("expected quit key to be swallowed, got %#v", got)
			}
		})
	}
}

func TestGlobalInputCaptureTranslatesVimKeysToScrollWhenHelpIsOpen(t *testing.T) {
	tests := []struct {
		rune rune
		want tcell.Key
	}{
		{'j', tcell.KeyDown},
		{'k', tcell.KeyUp},
	}

	for _, tt := range tests {
		s := New(resource.NewRegistry())
		s.helpOpen = true

		event := tcell.NewEventKey(tcell.KeyRune, tt.rune, tcell.ModNone)
		got := s.globalInputCapture(event)
		if got == nil || got.Key() != tt.want {
			t.Fatalf("expected %q to translate to %v, got %#v", tt.rune, tt.want, got)
		}
	}
}

type fakeServerFacetedResource struct {
	fakeResource
	options []string
}

func (f fakeServerFacetedResource) FacetOptions() []string { return f.options }
func (f fakeServerFacetedResource) FacetList(scope, value string) ([]resource.Row, error) {
	return nil, nil
}
func (f fakeServerFacetedResource) FacetCounts(scope string) (map[string]int, error) {
	return nil, nil
}

func TestGlobalInputCaptureTabCyclesFacetOnTablePage(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentListResource = "workers"
	s.currentServerFaceted = fakeServerFacetedResource{options: []string{"running", "stopped"}}
	s.currentFacetCounts = map[string]int{"running": 1, "stopped": 2}
	s.currentFacetValue = "running"

	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	if got := s.globalInputCapture(event); got != nil {
		t.Fatalf("expected Tab to be swallowed, got %#v", got)
	}

	if s.currentFacetValue != "stopped" {
		t.Fatalf("expected Tab to advance to \"stopped\", got %q", s.currentFacetValue)
	}
}

func TestGlobalInputCaptureTabIsNoOpWithoutFacets(t *testing.T) {
	s := New(resource.NewRegistry())

	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	if got := s.globalInputCapture(event); got != nil {
		t.Fatalf("expected Tab to be swallowed even as a no-op, got %#v", got)
	}
}

func TestGlobalInputCapturePassesQuitKeyToFooterInput(t *testing.T) {
	tests := []struct {
		name string
		mode footerMode
	}{
		{name: "command input", mode: footerCommand},
		{name: "filter input", mode: footerFilter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(resource.NewRegistry())
			s.footerMode = tt.mode

			event := tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)
			if got := s.globalInputCapture(event); got != event {
				t.Fatalf("expected quit key to pass through footer input, got %#v", got)
			}
		})
	}
}

func TestGlobalInputCaptureDispatchesDetailActionKey(t *testing.T) {
	registry := resource.NewRegistry()
	registry.Register(fakeResource{name: "errors"})
	s := New(registry)
	s.currentDetailActions = []resource.DetailAction{
		{Key: 'e', Label: "errors", Target: resource.NavTarget{ResourceName: "errors", ID: "pool-a", Kind: resource.NavScopedList}},
	}

	event := tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone)
	if got := s.globalInputCapture(event); got != nil {
		t.Fatalf("expected the action key to be swallowed, got %#v", got)
	}

	top, ok := s.stack.Top()
	if !ok || top.ResourceName != "errors" || top.Scope != "pool-a" || top.Kind != ListKind {
		t.Fatalf("expected navigation to errors scoped to pool-a, got %+v (ok=%v)", top, ok)
	}
}

func TestGlobalInputCaptureUnmatchedRuneIsPassedThrough(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentDetailActions = []resource.DetailAction{
		{Key: 'e', Label: "errors", Target: resource.NavTarget{ResourceName: "errors", ID: "pool-a"}},
	}

	event := tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone)
	if got := s.globalInputCapture(event); got != event {
		t.Fatalf("expected an unmatched rune to pass through unchanged, got %#v", got)
	}
}
