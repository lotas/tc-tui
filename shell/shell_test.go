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
