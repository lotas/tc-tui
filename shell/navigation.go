package shell

import (
	"fmt"
	"strings"
	"time"

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

	if direct, isDirect := res.(resource.DirectLookup); isDirect {
		if scope == "" {
			s.openIDPrompt(direct)
			return
		}
		s.switchToDetail(res, scope)
		return
	}

	if scoped, isScoped := res.(resource.ScopedResource); isScoped && scope == "" {
		s.switchResource(scoped.EmptyScopeResource(), "")
		return
	}

	s.stack.ResetTo(View{ResourceName: res.Name(), Kind: ListKind, Scope: scope})
	s.renderList(res, scope)
}

// switchToDetail resets the navigation stack to a Detail view for res/id —
// the DirectLookup counterpart of switchResource's List-view reset. Used by
// a `:name <id>` command and by the id-prompt handleFooterInputDone opens
// when <id> is omitted.
func (s *Shell) switchToDetail(res resource.Resource, id string) {
	s.stack.ResetTo(View{ResourceName: res.Name(), Kind: DetailKind, SelectedID: id})
	s.renderDetail(res, id)
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

// cycleFacet moves the current list view to the next (direction 1) or
// previous (direction -1) facet tab, wrapping around. For a client-side
// Faceted resource this is instant (refreshTable() re-filters s.lastRows).
// For a ServerFaceted resource it triggers a fresh fetch, since a different
// tab means different rows entirely.
func (s *Shell) cycleFacet(direction int) {
	switch {
	case s.currentServerFaceted != nil:
		tabs := ServerFacetTabs(s.currentServerFaceted, s.currentFacetCounts)
		next := cycleFacetValue(tabs, s.currentFacetValue, direction)
		if next == s.currentFacetValue {
			return
		}

		s.currentFacetValue = next
		s.facetByResource[s.currentListResource] = next
		s.table.ResetSelection()

		top, ok := s.stack.Top()
		if !ok {
			return
		}

		// Use the already-held ServerFaceted value directly (it embeds
		// resource.Resource) rather than re-resolving via the registry — a
		// tab switch doesn't change which resource is active, just its
		// facet value, so there's no need to look it up again.
		res := s.currentServerFaceted
		s.setTitle("Loading " + res.Name() + "...")
		s.table.SetData(s.currentColumns, nil, s.currentSort)
		s.loadList(res, top.Scope, s.currentFacetValue, true, false)

	case s.currentFaceted != nil:
		rows := FilterRows(s.lastRows, s.filterQuery)
		tabs := ClientFacetTabs(s.currentFaceted, rows)
		next := cycleFacetValue(tabs, s.currentFacetValue, direction)

		s.currentFacetValue = next
		s.facetByResource[s.currentListResource] = next
		s.table.ResetSelection()
		s.refreshTable()
	}
}

// toggleSort updates the active sort for the current list view: pressing
// the same column again reverses direction; any other column starts fresh
// at ascending. The result is remembered per-resource in sortByResource so
// it's restored the next time this resource is switched to.
func (s *Shell) toggleSort(column int) {
	if column < 0 || column >= len(s.currentColumns) {
		return
	}

	if s.currentSort.Direction != SortNone && s.currentSort.Column == column {
		if s.currentSort.Direction == SortAsc {
			s.currentSort.Direction = SortDesc
		} else {
			s.currentSort.Direction = SortAsc
		}
	} else {
		s.currentSort = SortState{Column: column, Direction: SortAsc}
	}

	s.sortByResource[s.currentListResource] = s.currentSort
	s.table.ResetSelection()
	s.refreshTable()
}

// refreshTable recomputes the table's displayed rows from s.lastRows by
// applying the current filter, facet, then sort, and re-renders. This is
// the single place list-view rows get filtered/sorted — call it any time
// s.lastRows, s.filterQuery, s.currentFacetValue, or s.currentSort changes.
func (s *Shell) refreshTable() {
	rows := FilterRows(s.lastRows, s.filterQuery)
	s.renderTabsBar(rows)

	// For a ServerFaceted resource, s.lastRows is already exactly the
	// selected tab's rows (server-filtered) — no client-side facet filter
	// applies. For a Faceted resource, filter client-side.
	if s.currentFaceted != nil {
		rows = FilterByFacet(rows, s.currentFaceted, s.currentFacetValue)
	}

	rows = SortRows(rows, s.currentSort)
	s.table.SetData(s.currentColumns, rows, s.currentSort)
}

// renderTabsBar recomputes and redraws the facet tab bar from rows (used
// only by the client-side Faceted case; ServerFaceted reads
// s.currentFacetCounts instead, ignoring rows), collapsing the bar (and its
// separator line) entirely when the current resource has no facets. Each
// tab is rendered as its own colored "pill" — a bright highlight for the
// active tab, a muted one for the rest — with a horizontal rule underneath
// to separate the bar from the table.
func (s *Shell) renderTabsBar(rows []resource.Row) {
	var tabs []FacetTab
	switch {
	case s.currentServerFaceted != nil:
		tabs = ServerFacetTabs(s.currentServerFaceted, s.currentFacetCounts)
	case s.currentFaceted != nil:
		tabs = ClientFacetTabs(s.currentFaceted, rows)
	default:
		s.tableContainer.ResizeItem(s.tabsBar, 0, 0)
		s.tableContainer.ResizeItem(s.tabsSeparator, 0, 0)
		s.tabsBar.SetText("")
		s.tabsSeparator.SetText("")
		return
	}

	s.tableContainer.ResizeItem(s.tabsBar, 1, 0)
	s.tableContainer.ResizeItem(s.tabsSeparator, 1, 0)
	s.tabsSeparator.SetText(strings.Repeat("─", 300)) // clipped to whatever width is available

	var b strings.Builder
	for i, tab := range tabs {
		if i > 0 {
			b.WriteString(" ")
		}

		label := tab.Value
		if label == "" {
			label = "All"
		}
		text := fmt.Sprintf("%s (%d)", label, tab.Count)

		if tab.Value == s.currentFacetValue {
			// White-on-blue rather than black-on-yellow: named colors like
			// "black"/"gray" get remapped unpredictably by terminal themes
			// and can end up low-contrast or oddly tinted; white text on a
			// solid blue fill reads reliably across terminals.
			b.WriteString(fmt.Sprintf("[white:blue:b] %s [-:-:-]", text))
		} else {
			b.WriteString(fmt.Sprintf("[white] %s ", text))
		}
	}

	s.tabsBar.SetText(" " + b.String())
}

func (s *Shell) renderList(res resource.Resource, scope string) {
	s.currentDetailActions = nil
	s.closeFooterInput()
	s.filterQuery = ""
	s.currentListResource = res.Name()
	s.currentColumns = res.Columns()
	s.currentSort = s.sortByResource[res.Name()] // zero value (SortNone) if not yet sorted

	s.currentFaceted = nil
	s.currentServerFaceted = nil
	s.currentFacetValue = ""
	s.currentFacetCounts = nil

	if sf, ok := res.(resource.ServerFaceted); ok {
		s.currentServerFaceted = sf
		s.currentFacetValue = s.restoreFacetValue(sf, res.Name())
	} else if f, ok := res.(resource.Faceted); ok {
		s.currentFaceted = f
		s.currentFacetValue = s.facetByResource[res.Name()] // "" (All) if never set
	}

	s.setTitle("Loading " + res.Name() + "...")
	s.table.SetData(s.currentColumns, nil, s.currentSort)
	s.renderTabsBar(nil)
	s.content.SwitchToPage(pageTable)
	s.app.SetFocus(s.table)

	s.startRefreshLoop(View{ResourceName: res.Name(), Kind: ListKind, Scope: scope}, res.RefreshInterval())
	s.loadList(res, scope, s.currentFacetValue, true, false)
}

// restoreFacetValue returns the remembered facet value for name if it's
// still a valid option for sf, otherwise sf's first (default) option.
func (s *Shell) restoreFacetValue(sf resource.ServerFaceted, name string) string {
	options := sf.FacetOptions()
	if len(options) == 0 {
		return ""
	}

	saved := s.facetByResource[name]
	for _, opt := range options {
		if opt == saved {
			return saved
		}
	}

	return options[0]
}

// applyListResult renders a successfully fetched (or cache-hit) list result:
// shared by loadList's synchronous cache-hit path and its async fetch-success
// path.
func (s *Shell) applyListResult(res resource.Resource, rows []resource.Row, counts map[string]int) {
	s.lastRows = rows
	if counts != nil {
		s.currentFacetCounts = counts
	}
	s.refreshTable()
	s.activeContent = s.table
	s.setTitle(res.Name())
	if s.footerMode == footerHints {
		s.renderFooterHints()
	}
}

// loadList fetches this view's rows: via ServerFaceted.FacetList(scope,
// facetValue) if the resource implements it, otherwise via ScopedList(scope)
// (if scope is non-empty) or List(). Passing facetValue explicitly (rather
// than reading s.currentFacetValue from the fetch goroutine) avoids a data
// race with a UI-thread tab switch, and re-checking it on completion
// discards a slow fetch that a newer tab switch has since superseded.
// isInitial distinguishes a first/navigation load (failure shows a
// full-screen error with retry) from a background refresh tick that hasn't
// (failure shows a transient warning and keeps the last-good render) — tab
// switches themselves pass isInitial=true, since they're a deliberate user
// action.
//
// Unless forceRefresh is set, this first checks the cache for this
// (resource, scope, facetValue) key and, on a hit, applies it synchronously
// — no goroutine, no network call. forceRefresh is set only by the
// auto-refresh ticker (Invalidate), whose whole purpose is to get a
// genuinely fresh result for the view currently on screen.
func (s *Shell) loadList(res resource.Resource, scope, facetValue string, isInitial, forceRefresh bool) {
	key := cacheKeyFor(res, scope, facetValue)

	if !forceRefresh {
		if entry, ok := s.cache.get(key, res.RefreshInterval()); ok {
			s.applyListResult(res, entry.rows, entry.counts)
			return
		}
	}

	go func() {
		var rows []resource.Row
		var counts map[string]int
		var err error

		if sf, ok := res.(resource.ServerFaceted); ok {
			rows, err = sf.FacetList(scope, facetValue)
			if err == nil {
				counts, err = sf.FacetCounts(scope)
			}
		} else if scope != "" {
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
			if facetValue != s.currentFacetValue {
				return // a newer tab switch has already superseded this fetch
			}

			if err != nil {
				if isInitial {
					s.showError(res.Name(), err, func() { s.renderList(res, scope) })
				} else {
					s.showTransientWarning(fmt.Sprintf("refresh failed: %s", err))
				}
				return
			}

			s.cache.set(key, cacheEntry{rows: rows, counts: counts, fetchedAt: time.Now()})
			s.applyListResult(res, rows, counts)
		})
	}()
}

func (s *Shell) renderDetail(res resource.Resource, id string) {
	s.currentDetailActions = nil
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
			s.currentDetailActions = detail.Actions
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
