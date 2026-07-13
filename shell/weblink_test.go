package shell

import (
	"errors"
	"testing"

	"github.com/taskcluster/tc-tui/resource"
)

type fakeWebLinkableResource struct {
	fakeResource
	detailURL string
	listURL   string
}

func (f fakeWebLinkableResource) DetailWebURL(rootURL, id string) string  { return f.detailURL }
func (f fakeWebLinkableResource) ListWebURL(rootURL, scope string) string { return f.listURL }

func TestOpenInBrowserDetailView(t *testing.T) {
	s := New(resource.NewRegistry())
	s.registry.Register(fakeWebLinkableResource{
		fakeResource: fakeResource{name: "task"},
		detailURL:    "https://tc.example.com/tasks/TASK1",
	})
	s.rootURL = "https://tc.example.com"
	s.stack.Push(View{ResourceName: "task", Kind: DetailKind, SelectedID: "TASK1"})

	var opened string
	s.openBrowser = func(url string) error {
		opened = url
		return nil
	}

	s.openInBrowser()

	if opened != "https://tc.example.com/tasks/TASK1" {
		t.Errorf("openBrowser called with %q, want the task's detail URL", opened)
	}
}

func TestOpenInBrowserListView(t *testing.T) {
	s := New(resource.NewRegistry())
	s.registry.Register(fakeWebLinkableResource{
		fakeResource: fakeResource{name: "workerpools"},
		listURL:      "https://tc.example.com/worker-manager",
	})
	s.stack.Push(View{ResourceName: "workerpools", Kind: ListKind})

	var opened string
	s.openBrowser = func(url string) error {
		opened = url
		return nil
	}

	s.openInBrowser()

	if opened != "https://tc.example.com/worker-manager" {
		t.Errorf("openBrowser called with %q, want the list URL", opened)
	}
}

func TestOpenInBrowserNoLinkShowsWarningInsteadOfOpening(t *testing.T) {
	s := New(resource.NewRegistry())
	s.registry.Register(fakeResource{name: "history"}) // does not implement WebLinkable
	s.stack.Push(View{ResourceName: "history", Kind: ListKind})

	called := false
	s.openBrowser = func(url string) error {
		called = true
		return nil
	}

	s.openInBrowser()

	if called {
		t.Error("openBrowser should not be called when the resource has no web UI link")
	}
}

func TestOpenInBrowserPropagatesOpenError(t *testing.T) {
	s := New(resource.NewRegistry())
	s.registry.Register(fakeWebLinkableResource{
		fakeResource: fakeResource{name: "task"},
		detailURL:    "https://tc.example.com/tasks/TASK1",
	})
	s.stack.Push(View{ResourceName: "task", Kind: DetailKind, SelectedID: "TASK1"})

	s.openBrowser = func(url string) error { return errors.New("no browser found") }

	s.openInBrowser() // must not panic; failure surfaces as a transient warning
}

func TestCurrentWebURLEmptyStack(t *testing.T) {
	s := New(resource.NewRegistry())

	if _, ok := s.currentWebURL(); ok {
		t.Error("currentWebURL() should report unavailable with an empty stack")
	}
}
