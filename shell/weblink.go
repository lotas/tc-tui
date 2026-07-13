package shell

import (
	"fmt"

	"github.com/taskcluster/tc-tui/resource"
)

// currentWebURL returns the web UI URL for whatever view is currently on
// top of the stack, and whether one is available at all — "" is a possible,
// valid URL result (e.g. an unscoped list a WebLinkable resource declines to
// link), which is why availability is reported separately.
func (s *Shell) currentWebURL() (url string, ok bool) {
	top, hasTop := s.stack.Top()
	if !hasTop {
		return "", false
	}

	res, ok := s.registry.Resolve(top.ResourceName)
	if !ok {
		return "", false
	}

	linkable, ok := res.(resource.WebLinkable)
	if !ok {
		return "", false
	}

	switch top.Kind {
	case DetailKind:
		return linkable.DetailWebURL(s.rootURL, top.SelectedID), true
	default:
		return linkable.ListWebURL(s.rootURL, top.Scope), true
	}
}

// openInBrowser opens the current view's web UI page (the 'o' key), or shows
// a transient warning if this resource/view has none.
func (s *Shell) openInBrowser() {
	url, ok := s.currentWebURL()
	if !ok || url == "" {
		s.showTransientWarning("no web UI page for this view")
		return
	}

	if err := s.openBrowser(url); err != nil {
		s.showTransientWarning(fmt.Sprintf("failed to open browser: %s", err))
	}
}
