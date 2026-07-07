package shell

import (
	"fmt"
	"strings"

	"github.com/taskcluster/tc-tui/resource"
)

// switchResource replaces the entire navigation stack with a fresh List
// view for the given resource name/alias — the `:` command bar's behavior.
// If the resolved resource is a ScopedResource and no scope was given, it
// redirects to that resource's EmptyScopeResource instead of attempting an
// unscoped fetch.
func (s *Shell) switchResource(nameOrAlias, scope string) {
	res, ok := s.registry.Resolve(nameOrAlias)
	if !ok {
		s.showError(nameOrAlias, fmt.Errorf(
			"unknown resource %q (available: %s)", nameOrAlias, strings.Join(s.registry.Names(), ", "),
		), func() {})
		return
	}

	if scoped, isScoped := res.(resource.ScopedResource); isScoped && scope == "" {
		s.switchResource(scoped.EmptyScopeResource(), "")
		return
	}

	s.stack.ResetTo(View{ResourceName: res.Name(), Kind: ListKind, Scope: scope})
	s.renderList(res, scope)
}

// showDetail pushes a Detail view for id onto the stack.
func (s *Shell) showDetail(resourceName, id string) {
	res, ok := s.registry.Resolve(resourceName)
	if !ok {
		s.showError(resourceName, fmt.Errorf("unknown resource %q", resourceName), func() {})
		return
	}

	s.stack.Push(View{ResourceName: res.Name(), Kind: DetailKind, SelectedID: id})
	s.renderDetail(res, id)
}

// pushScopedList pushes a List view scoped to scope onto the stack — what a
// DetailAction with Kind: NavScopedList triggers. Unlike switchResource,
// this pushes rather than resets, so Esc returns to the view that launched
// it.
func (s *Shell) pushScopedList(resourceName, scope string) {
	res, ok := s.registry.Resolve(resourceName)
	if !ok {
		s.showError(resourceName, fmt.Errorf("unknown resource %q", resourceName), func() {})
		return
	}

	s.stack.Push(View{ResourceName: res.Name(), Kind: ListKind, Scope: scope})
	s.renderList(res, scope)
}

// goBack pops the top view and re-renders the new top, or quits if the
// stack is now empty.
func (s *Shell) goBack() {
	if _, ok := s.stack.Pop(); !ok {
		s.Stop()
		return
	}

	top, ok := s.stack.Top()
	if !ok {
		s.Stop()
		return
	}

	res, ok := s.registry.Resolve(top.ResourceName)
	if !ok {
		s.showError(top.ResourceName, fmt.Errorf("unknown resource %q", top.ResourceName), func() {})
		return
	}

	switch top.Kind {
	case ListKind:
		s.renderList(res, top.Scope)
	case DetailKind:
		s.renderDetail(res, top.SelectedID)
	}
}

func (s *Shell) renderList(res resource.Resource, scope string) {
	s.closeFooterInput()
	s.filterQuery = ""
	s.currentListResource = res.Name()
	s.currentColumns = res.Columns()

	s.setTitle("Loading " + res.Name() + "...")
	s.table.SetData(s.currentColumns, nil)
	s.content.SwitchToPage(pageTable)
	s.app.SetFocus(s.table)

	s.startRefreshLoop(View{ResourceName: res.Name(), Kind: ListKind, Scope: scope}, res.RefreshInterval())
	s.loadList(res, scope, true)
}

// loadList fetches List() (or, if scope is non-empty, ScopedList(scope)) in
// the background. isInitial distinguishes a first/navigation load (failure
// shows a full-screen error with retry) from a background refresh tick
// (failure shows a transient warning and keeps the last-good render).
func (s *Shell) loadList(res resource.Resource, scope string, isInitial bool) {
	go func() {
		var rows []resource.Row
		var err error

		if scope != "" {
			scoped, ok := res.(resource.ScopedResource)
			if !ok {
				err = fmt.Errorf("%s does not support a scoped list", res.Name())
			} else {
				rows, err = scoped.ScopedList(scope)
			}
		} else {
			rows, err = res.List()
		}

		s.app.QueueUpdateDraw(func() {
			if !s.isTopView(View{ResourceName: res.Name(), Kind: ListKind, Scope: scope}) {
				return
			}

			if err != nil {
				if isInitial {
					s.showError(res.Name(), err, func() { s.renderList(res, scope) })
				} else {
					s.showTransientWarning(fmt.Sprintf("refresh failed: %s", err))
				}
				return
			}

			s.lastRows = rows
			s.table.SetData(s.currentColumns, FilterRows(rows, s.filterQuery))
			s.activeContent = s.table
			s.setTitle(res.Name())
			if s.footerMode == footerHints {
				s.renderFooterHints()
			}
		})
	}()
}

func (s *Shell) renderDetail(res resource.Resource, id string) {
	s.closeFooterInput()

	s.setTitle("Loading " + res.Name() + "...")
	s.detail.SetData(resource.Detail{})
	s.content.SwitchToPage(pageDetail)
	s.app.SetFocus(s.detail)

	s.startRefreshLoop(View{ResourceName: res.Name(), Kind: DetailKind, SelectedID: id}, res.RefreshInterval())
	s.loadDetail(res, id, true)
}

func (s *Shell) loadDetail(res resource.Resource, id string, isInitial bool) {
	go func() {
		detail, err := res.Describe(id)
		s.app.QueueUpdateDraw(func() {
			if !s.isTopView(View{ResourceName: res.Name(), Kind: DetailKind, SelectedID: id}) {
				return
			}

			if err != nil {
				if isInitial {
					s.showError(fmt.Sprintf("%s %s", res.Name(), id), err, func() { s.renderDetail(res, id) })
				} else {
					s.showTransientWarning(fmt.Sprintf("refresh failed: %s", err))
				}
				return
			}

			s.detail.SetData(detail)
			s.activeContent = s.detail
			s.setTitle(detail.Title)
			if s.footerMode == footerHints {
				s.renderFooterHints()
			}
		})
	}()
}

func (s *Shell) showError(title string, err error, retry func()) {
	s.stopRefreshLoop()

	s.errorView.SetError(title, err)
	s.errorView.SetOnRetry(retry)
	s.activeContent = s.errorView

	s.setTitle(fmt.Sprintf("Error :: %s", title))
	s.content.SwitchToPage(pageError)
	s.app.SetFocus(s.errorView)
}

func (s *Shell) showTransientWarning(msg string) {
	s.footerHint.SetText(fmt.Sprintf("[red]%s[white]", msg))
}
