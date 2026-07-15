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
// unscoped fetch. The "history" resource is the one exception: it's pushed
// instead (see below), so it doesn't discard the current screen.
func (s *Shell) switchResource(nameOrAlias, scope string) {
	res, ok := s.registry.Resolve(nameOrAlias)
	if !ok {
		s.showError(nameOrAlias, fmt.Errorf(
			"unknown resource %q (available: %s)", nameOrAlias, strings.Join(s.registry.Names(), ", "),
		), func() {})
		return
	}

	// history is a navigational aid, not a destination in its own right —
	// opening it should feel like a peek, not a fresh root. Pushed rather
	// than reset so Esc returns to whatever screen was open before `:history`
	// was run, instead of that screen being discarded from the stack.
	if res.Name() == "history" {
		s.stack.Push(View{ResourceName: res.Name(), Kind: ListKind})
		s.renderList(res, "", false)
		return
	}

	// Checked before DirectLookup: a DirectScopedResource's method set is a
	// superset of DirectLookup's (it also embeds ScopedResource), so it
	// would match that broader assertion too — the more specific check must
	// win, or a resource like TaskGroupResource would be routed into
	// switchToDetail (Describe) instead of a scoped List view.
	if dsr, isDirectScoped := res.(resource.DirectScopedResource); isDirectScoped {
		if scope == "" {
			s.openIDPrompt(dsr.IDPromptLabel(), func(id string) {
				s.switchResource(dsr.Name(), id)
			})
			return
		}
		s.stack.ResetTo(View{ResourceName: res.Name(), Kind: ListKind, Scope: scope})
		s.renderList(res, scope, false)
		return
	}

	if direct, isDirect := res.(resource.DirectLookup); isDirect {
		if scope == "" {
			s.openIDPrompt(direct.IDPromptLabel(), func(id string) {
				s.switchToDetail(res, id)
			})
			return
		}
		s.switchToDetail(res, scope)
		return
	}

	if scoped, isScoped := res.(resource.ScopedResource); isScoped {
		if scope == "" {
			s.switchResource(scoped.EmptyScopeResource(), "")
			return
		}
		s.stack.ResetTo(View{ResourceName: res.Name(), Kind: ListKind, Scope: scope})
		s.renderList(res, scope, false)
		return
	}

	// res is a plain Resource: no ScopedList, no DirectLookup. A second
	// argument here can only mean "open this id directly" (e.g. `:workerpools
	// proj-taskcluster/ci`) — res.List() takes no scope, so there's no scoped
	// list this could otherwise mean.
	if scope != "" {
		s.switchToDetail(res, scope)
		return
	}

	s.stack.ResetTo(View{ResourceName: res.Name(), Kind: ListKind})
	s.renderList(res, "", false)
}

// switchToDetail resets the navigation stack to a Detail view for res/id —
// the DirectLookup counterpart of switchResource's List-view reset. Used by
// a `:name <id>` command and by the id-prompt handleFooterInputDone opens
// when <id> is omitted.
func (s *Shell) switchToDetail(res resource.Resource, id string) {
	s.stack.ResetTo(View{ResourceName: res.Name(), Kind: DetailKind, SelectedID: id})
	s.renderDetail(res, id, false)
}

// showDetail pushes a Detail view for id onto the stack.
func (s *Shell) showDetail(resourceName, id string) {
	res, ok := s.registry.Resolve(resourceName)
	if !ok {
		s.showError(resourceName, fmt.Errorf("unknown resource %q", resourceName), func() {})
		return
	}

	s.stack.Push(View{ResourceName: res.Name(), Kind: DetailKind, SelectedID: id})
	s.renderDetail(res, id, false)
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
	s.renderList(res, scope, false)
}

// navigateTo executes a NavTarget the same way regardless of where it came
// from — a Detail view's action keybinding or a list row's NavTarget
// override (e.g. HistoryResource's rows).
//
// If history is the view being navigated away from, it's popped first so
// the target replaces it on the stack instead of stacking on top of it —
// otherwise Esc from the target would land back on history (itself pushed,
// not reset, specifically so Esc could skip past it) rather than on
// whatever screen was open before `:history` was run.
func (s *Shell) navigateTo(target resource.NavTarget) {
	if top, ok := s.stack.Top(); ok && top.ResourceName == "history" {
		s.stack.Pop()
	}

	switch target.Kind {
	case resource.NavScopedList:
		s.pushScopedList(target.ResourceName, target.ID)
	default:
		s.showDetail(target.ResourceName, target.ID)
	}
}

// renderRestoredTop renders the current stack's top view — called once from
// Start with a stack populated by RestoreState (if any). It renders with
// isRestore=true, so a resolve/fetch failure for that view pops it and
// retries the next one down instead of showing the error screen (see
// loadList/loadDetail's isRestore-failure branch), so a stale restored view
// (e.g. an entity deleted since the last session) falls back through the
// breadcrumb trail silently. Once the stack empties out, it falls back to
// s.restoreFallback exactly as a fresh launch would.
func (s *Shell) renderRestoredTop() {
	top, ok := s.stack.Top()
	if !ok {
		s.switchResource(s.restoreFallback, "")
		return
	}

	res, ok := s.registry.Resolve(top.ResourceName)
	if !ok {
		s.stack.Pop()
		s.renderRestoredTop()
		return
	}

	switch top.Kind {
	case ListKind:
		s.renderList(res, top.Scope, true)
	case DetailKind:
		s.renderDetail(res, top.SelectedID, true)
	}
}

// goBack pops the top view and re-renders the new top. Esc never quits —
// only q does — so this is a no-op once only the root view is left.
func (s *Shell) goBack() {
	if s.stack.Len() <= 1 {
		return
	}

	s.stack.Pop()

	top, ok := s.stack.Top()
	if !ok {
		return
	}

	res, ok := s.registry.Resolve(top.ResourceName)
	if !ok {
		s.showError(top.ResourceName, fmt.Errorf("unknown resource %q", top.ResourceName), func() {})
		return
	}

	switch top.Kind {
	case ListKind:
		s.renderList(res, top.Scope, false)
	case DetailKind:
		s.renderDetail(res, top.SelectedID, false)
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
		s.loadList(res, top.Scope, s.currentFacetValue, true, false, false)

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

// toggleExpandColumns flips whether the table's columns honor their Width
// cap — the 'x' key's behavior. Columns that no longer fit the terminal once
// expanded are reachable with the Left/Right arrow keys (tview.Table's
// built-in horizontal scrolling).
func (s *Shell) toggleExpandColumns() {
	s.table.SetExpandColumns(!s.table.ExpandColumns())
	s.refreshTable()
}

// refreshTable recomputes the table's displayed rows from s.lastRows by
// applying the current filter, facet, then sort, and re-renders; it also
// updates the title to show the active filter, if any. This is the single
// place list-view rows get filtered/sorted — call it any time s.lastRows,
// s.filterQuery, s.currentFacetValue, or s.currentSort changes.
func (s *Shell) refreshTable() {
	title := s.currentListResource
	if s.currentListScope != "" {
		title += " (" + s.currentListScope + ")"
	}
	if s.currentScopeSubtitle != "" {
		title += " [" + s.currentScopeSubtitle + "]"
	}
	if s.filterQuery != "" {
		title += " (" + s.filterQuery + ")"
	}
	if s.augmentTotal > 0 && s.augmentCompleted < s.augmentTotal {
		title += fmt.Sprintf(" [%d/%d]", s.augmentCompleted, s.augmentTotal)
	}
	if s.table.ExpandColumns() {
		title += " [no truncation]"
	}
	s.setTitle(title)

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

func (s *Shell) renderList(res resource.Resource, scope string, isRestore bool) {
	s.currentDetailActions = nil
	if sa, ok := res.(resource.ScopeActions); ok {
		s.currentDetailActions = sa.ScopeActions(scope)
	}
	s.closeFooterInput()
	s.filterQuery = s.filterByResource[res.Name()] // "" if never set
	s.currentListResource = res.Name()
	s.currentListScope = scope
	s.currentScopeSubtitle = "" // repopulated by loadList if res implements ScopeSubtitle
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

	s.renderHeaderHints()
	s.renderBreadcrumbs()
	s.setTitle("Loading " + res.Name() + "...")
	s.table.SetData(s.currentColumns, nil, s.currentSort)
	s.renderTabsBar(nil)
	s.content.SwitchToPage(pageTable)
	s.app.SetFocus(s.table)

	s.startRefreshLoop(View{ResourceName: res.Name(), Kind: ListKind, Scope: scope}, res.RefreshInterval())
	s.loadList(res, scope, s.currentFacetValue, true, false, isRestore)
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
// path. subtitle is whatever a ScopeSubtitle resource returned alongside
// rows ("" if the resource doesn't implement it, or the scope is empty).
func (s *Shell) applyListResult(res resource.Resource, rows []resource.Row, counts map[string]int, subtitle string) {
	s.lastRows = rows
	s.currentScopeSubtitle = subtitle
	s.augmentCompleted, s.augmentTotal = 0, 0
	s.augmentEpoch++
	if counts != nil {
		s.currentFacetCounts = counts
	}
	s.refreshTable()
	s.activeContent = s.table
	s.renderHeaderHints()
	s.renderBreadcrumbs()
}

// nextLoadGeneration returns the generation the caller's dispatch should
// capture. A genuine navigation dispatch (isInitial=true — a real
// navigation, a restore-replay, or a facet-tab switch) starts a NEW
// generation (increments first). A background refresh tick (isInitial=false,
// used exclusively by Invalidate) instead just returns whichever generation
// is already current, inheriting the epoch of the view it's refreshing
// rather than starting its own or being exempt from the mechanism entirely.
// This is the single place that decides capture behavior for both loadList
// and loadDetail — see loadGeneration's doc comment in shell.go for how the
// captured value is later used.
func (s *Shell) nextLoadGeneration(isInitial bool) int {
	if isInitial {
		s.loadGeneration++
	}
	return s.loadGeneration
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
// action. isRestore is true only for the one call renderRestoredTop itself
// issues for the view it's replaying — every other caller passes false.
//
// Unless forceRefresh is set, this first checks the cache for this
// (resource, scope, facetValue) key and, on a hit, applies it synchronously
// — no goroutine, no network call. forceRefresh is set only by the
// auto-refresh ticker (Invalidate), whose whole purpose is to get a
// genuinely fresh result for the view currently on screen.
func (s *Shell) loadList(res resource.Resource, scope, facetValue string, isInitial, forceRefresh, isRestore bool) {
	gen := s.nextLoadGeneration(isInitial)

	key := cacheKeyFor(res, scope, facetValue)

	recordVisit := func() {
		if isInitial && !isRestore && scope != "" && res.Name() != "history" && s.historyRecorder != nil {
			s.historyRecorder.Record(resource.HistoryEntry{
				ResourceName: res.Name(),
				Kind:         int(ListKind),
				Scope:        scope,
				VisitedAt:    time.Now(),
			})
		}
	}

	if !forceRefresh {
		if entry, ok := s.cache.get(key, res.RefreshInterval()); ok {
			s.applyListResult(res, entry.rows, entry.counts, entry.subtitle)
			recordVisit() // <-- new
			return
		}
	}

	go func() {
		var rows []resource.Row
		var counts map[string]int
		var err error
		var epoch int
		var subtitle string

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

		// A ScopeSubtitle failure is non-fatal — sealed/expiry-style context
		// is supplementary, not worth failing the whole list load over.
		if err == nil && scope != "" {
			if ss, ok := res.(resource.ScopeSubtitle); ok {
				subtitle, _ = ss.Subtitle(scope)
			}
		}

		s.app.QueueUpdateDraw(func() {
			if s.isStaleLoad(gen) {
				return // a newer navigation dispatch has started since — even for the same View
			}
			if !s.isTopView(View{ResourceName: res.Name(), Kind: ListKind, Scope: scope}) {
				return
			}
			if facetValue != s.currentFacetValue {
				return // a newer tab switch has already superseded this fetch
			}

			if err != nil {
				// isRestore must be checked before isInitial: renderRestoredTop
				// always dispatches with isInitial=true, so swapping this order
				// would route a restore-chain failure into showError instead of
				// the silent pop-and-retry.
				if isRestore {
					s.stack.Pop()
					s.renderRestoredTop()
					return
				}
				if isInitial {
					s.showError(res.Name(), err, func() { s.renderList(res, scope, false) })
				} else {
					s.showTransientWarning(fmt.Sprintf("refresh failed: %s", err))
				}
				return
			}

			s.cache.set(key, cacheEntry{rows: rows, counts: counts, subtitle: subtitle, fetchedAt: time.Now()})
			s.applyListResult(res, rows, counts, subtitle) // bumps s.augmentEpoch
			epoch = s.augmentEpoch
			recordVisit() // <-- new
		})

		if err == nil {
			if a, ok := res.(resource.Augmentable); ok {
				view := View{ResourceName: res.Name(), Kind: ListKind, Scope: scope}
				a.Augment(rows, func(updated []resource.Row, completed, total int) {
					s.app.QueueUpdateDraw(func() {
						if s.isStaleLoad(gen) || !s.isTopView(view) || facetValue != s.currentFacetValue {
							return // the same staleness checks the base-rows success branch above uses
						}
						if s.augmentEpoch != epoch {
							return // a newer base-row render for this view has since started — drop this now-stale tick rather than clobbering fresher rows
						}
						s.lastRows = updated
						s.augmentCompleted, s.augmentTotal = completed, total
						s.refreshTable()
						s.cache.set(key, cacheEntry{rows: updated, counts: counts, subtitle: subtitle, fetchedAt: time.Now()})
					})
				})
			}
		}
	}()
}

func (s *Shell) renderDetail(res resource.Resource, id string, isRestore bool) {
	s.currentDetailActions = nil
	s.closeFooterInput()

	s.renderHeaderHints()
	s.renderBreadcrumbs()
	s.setTitle("Loading " + res.Name() + "...")
	s.detail.SetData(resource.Detail{})
	s.content.SwitchToPage(pageDetail)
	s.app.SetFocus(s.detail)

	s.startRefreshLoop(View{ResourceName: res.Name(), Kind: DetailKind, SelectedID: id}, res.RefreshInterval())
	s.loadDetail(res, id, true, isRestore)
}

func (s *Shell) loadDetail(res resource.Resource, id string, isInitial, isRestore bool) {
	gen := s.nextLoadGeneration(isInitial)

	go func() {
		detail, err := res.Describe(id)
		s.app.QueueUpdateDraw(func() {
			if s.isStaleLoad(gen) {
				return // a newer navigation dispatch has started since — even for the same View
			}
			if !s.isTopView(View{ResourceName: res.Name(), Kind: DetailKind, SelectedID: id}) {
				return
			}

			if err != nil {
				// isRestore must be checked before isInitial: renderRestoredTop
				// always dispatches with isInitial=true, so swapping this order
				// would route a restore-chain failure into showError instead of
				// the silent pop-and-retry.
				if isRestore {
					s.stack.Pop()
					s.renderRestoredTop()
					return
				}
				if isInitial {
					s.showError(fmt.Sprintf("%s %s", res.Name(), id), err, func() { s.renderDetail(res, id, false) })
				} else {
					s.showTransientWarning(fmt.Sprintf("refresh failed: %s", err))
				}
				return
			}

			if isInitial {
				s.detail.SetData(detail)
			} else {
				s.detail.UpdateData(detail)
			}
			s.currentDetailActions = detail.Actions
			s.activeContent = s.detail
			s.setTitle(detail.Title)
			s.renderHeaderHints()
			s.renderBreadcrumbs()

			if isInitial && !isRestore && res.Name() != "history" && s.historyRecorder != nil {
				s.historyRecorder.Record(resource.HistoryEntry{
					ResourceName: res.Name(),
					Kind:         int(DetailKind),
					SelectedID:   id,
					Title:        detail.Title,
					VisitedAt:    time.Now(),
				})
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
	s.footerBreadcrumb.SetText(fmt.Sprintf("[red]%s[white]", msg))
}
