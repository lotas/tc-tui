package shell

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/taskcluster/tc-tui/resource"
)

// fakeActionableResource is a minimal resource.Actionable used to exercise
// the shared action flow without any real Taskcluster calls.
type fakeActionableResource struct {
	fakeResource
	actions       []resource.Action
	describeCalls *int32
}

func (f fakeActionableResource) Describe(id string) (resource.Detail, error) {
	if f.describeCalls != nil {
		atomic.AddInt32(f.describeCalls, 1)
	}
	return resource.Detail{Title: f.name + " " + id}, nil
}

func (f fakeActionableResource) Actions(id string) []resource.Action {
	return f.actions
}

// readOnUI runs fn on the tview event loop and returns its result, so a test
// can inspect Shell/tview state a background action goroutine may be mutating
// without racing it — production code only ever touches that state on the UI
// goroutine too.
func readOnUI[T any](s *Shell, fn func() T) T {
	ch := make(chan T, 1)
	s.app.QueueUpdateDraw(func() { ch <- fn() })
	return <-ch
}

func actionableShell(t *testing.T, actions []resource.Action) (*Shell, *fakeActionableResource) {
	t.Helper()
	var describeCalls int32
	res := &fakeActionableResource{
		fakeResource:  fakeResource{name: "widgets"},
		actions:       actions,
		describeCalls: &describeCalls,
	}
	registry := resource.NewRegistry()
	registry.Register(res)

	s := New(registry)
	s.stack.Push(View{ResourceName: "widgets", Kind: DetailKind, SelectedID: "w1"})
	return s, res
}

func TestSubmitActionBlocksOnParseError(t *testing.T) {
	var performed int32
	action := resource.Action{
		Key:        'y',
		Label:      "edit widget",
		Prompt:     "Apply this definition?",
		Input:      resource.InputYAML,
		InputLabel: "definition",
		Perform:    func(resource.ActionInput) error { atomic.AddInt32(&performed, 1); return nil },
	}
	s, _ := actionableShell(t, []resource.Action{action})

	s.startAction(action)
	s.actionView.setFieldText(s.actionView.inputIndex, "name: [unterminated")
	s.submitAction()

	if atomic.LoadInt32(&performed) != 0 {
		t.Fatalf("Perform must not run when the input fails to parse")
	}
	if !s.actionOpen {
		t.Fatalf("dialog should stay open after a validation error")
	}
	if got := s.actionView.status.GetText(true); !strings.Contains(got, "invalid YAML") {
		t.Fatalf("expected an invalid-YAML status, got %q", got)
	}
}

func TestSubmitActionBlocksOnFieldValidationError(t *testing.T) {
	var performed int32
	action := resource.Action{
		Key:        'p',
		Label:      "set priority",
		Prompt:     "Set a new priority?",
		Input:      resource.InputLine,
		InputLabel: "priority",
		Validate: func(in resource.ActionInput) error {
			if in.Raw != "high" {
				return fmt.Errorf("priority must be one of: low, high")
			}
			return nil
		},
		Perform: func(resource.ActionInput) error { atomic.AddInt32(&performed, 1); return nil },
	}
	s, _ := actionableShell(t, []resource.Action{action})

	s.startAction(action)
	s.actionView.setFieldText(s.actionView.inputIndex, "bogus")
	s.submitAction()

	if atomic.LoadInt32(&performed) != 0 {
		t.Fatalf("Perform must not run when field validation fails")
	}
	if got := s.actionView.status.GetText(true); !strings.Contains(got, "priority must be one of") {
		t.Fatalf("expected the validator's message in the status, got %q", got)
	}
}

func TestSubmitActionRequiresConfirmWordForDestructive(t *testing.T) {
	var performed int32
	action := resource.Action{
		Key:         'd',
		Label:       "delete widget",
		Destructive: true,
		ConfirmWord: "DELETE",
		Prompt:      "Delete widget w1?",
		Perform:     func(resource.ActionInput) error { atomic.AddInt32(&performed, 1); return nil },
	}
	s, _ := actionableShell(t, []resource.Action{action})

	s.startAction(action)
	s.actionView.setFieldText(s.actionView.confirmIndex, "delete") // wrong case
	s.submitAction()

	if atomic.LoadInt32(&performed) != 0 {
		t.Fatalf("Perform must not run until the confirm word matches exactly")
	}
	if got := s.actionView.status.GetText(true); !strings.Contains(got, "confirm") {
		t.Fatalf("expected a confirm-word prompt in the status, got %q", got)
	}
}

func TestActionSuccessInvalidatesCacheAndClosesDialog(t *testing.T) {
	var performed int32
	action := resource.Action{
		Key:         'd',
		Label:       "delete widget",
		Destructive: true,
		Prompt:      "Delete widget w1?",
		Invalidates: []string{"secrets"},
		Perform:     func(resource.ActionInput) error { atomic.AddInt32(&performed, 1); return nil },
	}
	s, _ := actionableShell(t, []resource.Action{action})
	startRunning(t, s)

	// Seed a cache entry the mutation should drop.
	s.cache.set(cacheKey{resource: "secrets"}, cacheEntry{fetchedAt: time.Now()})

	s.app.QueueUpdateDraw(func() { s.startAction(action) })
	waitFor(t, func() bool { return readOnUI(s, func() bool { return s.actionOpen }) })

	s.app.QueueUpdateDraw(func() { s.submitAction() })

	waitFor(t, func() bool { return atomic.LoadInt32(&performed) == 1 })
	waitFor(t, func() bool { return readOnUI(s, func() bool { return !s.actionOpen }) })

	cacheHit := readOnUI(s, func() bool {
		_, ok := s.cache.get(cacheKey{resource: "secrets"}, time.Minute)
		return ok
	})
	if cacheHit {
		t.Fatalf("expected the invalidated cache entry to be dropped after success")
	}
	if got := readOnUI(s, func() string { return s.footerBreadcrumb.GetText(true) }); !strings.Contains(got, "done") {
		t.Fatalf("expected a success message in the footer, got %q", got)
	}
}

func TestNonDestructiveActionRefreshesCurrentView(t *testing.T) {
	action := resource.Action{
		Key:        'e',
		Label:      "edit widget",
		Prompt:     "Apply?",
		Input:      resource.InputLine,
		InputLabel: "value",
		Perform:    func(resource.ActionInput) error { return nil },
	}
	s, res := actionableShell(t, []resource.Action{action})
	startRunning(t, s)

	s.app.QueueUpdateDraw(func() {
		s.startAction(action)
		s.actionView.setFieldText(s.actionView.inputIndex, "new-value")
		s.submitAction()
	})

	// A non-destructive success force-refreshes the current detail view, so
	// Describe is re-issued.
	waitFor(t, func() bool { return atomic.LoadInt32(res.describeCalls) >= 1 })
	waitFor(t, func() bool { return readOnUI(s, func() bool { return !s.actionOpen }) })
}

func TestActionPerformErrorKeepsDialogOpen(t *testing.T) {
	action := resource.Action{
		Key:     'g',
		Label:   "do it",
		Prompt:  "Proceed?",
		Perform: func(resource.ActionInput) error { return fmt.Errorf("boom from API") },
	}
	s, _ := actionableShell(t, []resource.Action{action})
	startRunning(t, s)

	s.app.QueueUpdateDraw(func() { s.startAction(action) })
	waitFor(t, func() bool { return readOnUI(s, func() bool { return s.actionOpen }) })

	s.app.QueueUpdateDraw(func() { s.submitAction() })

	waitFor(t, func() bool {
		return strings.Contains(readOnUI(s, func() string { return s.actionView.status.GetText(true) }), "boom from API")
	})

	if !readOnUI(s, func() bool { return s.actionOpen }) {
		t.Fatalf("dialog should stay open so the user can retry after an API failure")
	}
}

func TestActionKeyDispatchOpensDialog(t *testing.T) {
	action := resource.Action{
		Key:     'C',
		Label:   "cancel widget",
		Prompt:  "Cancel widget w1?",
		Perform: func(resource.ActionInput) error { return nil },
	}
	s, _ := actionableShell(t, []resource.Action{action})

	consumed := s.globalInputCapture(tcell.NewEventKey(tcell.KeyRune, 'C', tcell.ModNone))

	if consumed != nil {
		t.Fatalf("expected the action key to be consumed, got %+v", consumed)
	}
	if !s.actionOpen {
		t.Fatalf("expected the action dialog to open on its key")
	}
	if s.currentAction.Label != "cancel widget" {
		t.Fatalf("expected the dispatched action to be current, got %q", s.currentAction.Label)
	}
}

func TestGlobalKeysPassThroughWhileActionOpen(t *testing.T) {
	s, _ := actionableShell(t, nil)
	s.actionOpen = true

	// 'q' would normally quit; while the dialog owns input it must reach the
	// form untouched instead.
	event := tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)
	if got := s.globalInputCapture(event); got != event {
		t.Fatalf("expected keys to pass through to the action form, got %+v", got)
	}
}

func TestRenderHeaderHintsShowsActionKeys(t *testing.T) {
	s := New(resource.NewRegistry())
	s.stack.Push(View{ResourceName: "widgets", Kind: DetailKind, SelectedID: "w1"})
	s.currentActions = []resource.Action{{Key: 'd', Label: "delete widget", Destructive: true}}

	s.renderHeaderHints()

	text := s.headerHint.GetText(false)
	if !strings.Contains(text, "delete widget") || !strings.Contains(text, "d") {
		t.Fatalf("expected the action key hint to be shown, got %q", text)
	}
}
