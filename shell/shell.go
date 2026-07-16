package shell

import (
	"fmt"
	"sync/atomic"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/taskcluster/tc-tui/resource"
)

const (
	pageTable  = "table"
	pageDetail = "detail"
	pageError  = "error"
	pageHelp   = "help"

	pageFooterBreadcrumb = "breadcrumb"
	pageFooterInput      = "input"
)

type footerMode int

const (
	footerIdle footerMode = iota
	footerCommand
	footerFilter
	footerPrompt
)

// Shell is the generic navigation engine: registry, view stack, table/detail
// views, command bar, filter, refresh loop. It knows nothing about roles or
// worker pools specifically — only the Resource interface.
type Shell struct {
	app      *tview.Application
	registry *resource.Registry
	stack    *ViewStack

	root       *tview.Grid
	headerLeft *tview.TextView
	headerHint *tview.TextView

	content   *tview.Pages
	table     *TableView
	detail    *DetailView
	errorView *ErrorView
	helpView  *HelpView

	helpOpen    bool
	preHelpPage string

	footer              *tview.Pages
	footerBreadcrumb    *tview.TextView
	footerInput         *tview.InputField
	footerMode          footerMode
	pendingLookupCommit func(id string) // set while footerMode == footerPrompt; called with the entered id

	currentListResource  string
	currentListScope     string // "" for an unscoped list
	currentScopeSubtitle string // set by a ScopeSubtitle resource alongside its rows; "" if none/not applicable
	currentColumns       []resource.Column
	lastRows             []resource.Row
	filterQuery          string
	filterByResource     map[string]string
	currentDetailActions []resource.DetailAction
	currentDetailTitle   string

	// currentListTruncated reports whether the current list view's rows were
	// capped at the safe fetch limit with more left unfetched server-side
	// (see resource.PartialLister) — drives refreshTable's "[N+]" title
	// suffix and the 'L' load-all key and hint.
	currentListTruncated bool

	// loadAllKeys remembers, per list cache key, that the user asked for the
	// full uncapped fetch ('L') — consulted by loadList so later loads of
	// the same view (auto-refresh ticks, navigating back within the cache
	// TTL) stay uncapped instead of silently reverting to the capped first
	// fetch.
	loadAllKeys map[cacheKey]bool

	currentSort    SortState
	sortByResource map[string]SortState

	tabsBar        *tview.TextView
	tabsSeparator  *tview.TextView
	tableContainer *tview.Flex

	currentFaceted       resource.Faceted
	currentServerFaceted resource.ServerFaceted
	currentFacetValue    string
	currentFacetCounts   map[string]int
	facetByResource      map[string]string

	// augmentCompleted/augmentTotal track an in-progress Augmentable.Augment
	// call for the current list view — augmentTotal == 0 means no
	// augmentation is active (or applicable), so refreshTable's title
	// suffix is hidden. Reset to 0,0 by applyListResult on every fresh
	// base-row render (navigation, refresh, or a cache hit — which never
	// gets a follow-up Augment call).
	augmentCompleted int
	augmentTotal     int

	// augmentEpoch is bumped by applyListResult every time base rows are
	// freshly applied for a list view. A refresh reuses the same View and
	// the same loadGeneration as the render it's refreshing (see
	// loadGeneration's doc comment) — isStaleLoad/isTopView alone can't
	// tell a slow, still-in-flight Augment run left over from a PRIOR
	// refresh cycle apart from one belonging to the CURRENT rows on
	// screen. loadList captures the epoch right after applying a given
	// set of base rows and threads it through to that Augment call's
	// onUpdate closure, which drops any tick where the epoch has since
	// moved on — otherwise a slow augmentation's late ticks would
	// overwrite a newer reload's fresh (placeholder) rows with stale
	// computed values.
	augmentEpoch int

	// augmentedRowIDs tracks which of the current view's rows have already
	// been REQUESTED from Augmentable.Augment (dispatched — not necessarily
	// finished yet) for the current augmentEpoch — refreshTable calls
	// triggerAugmentForNewlyVisibleRows on every filter/facet/sort change,
	// which uses this to avoid dispatching a duplicate request for a row
	// already in flight, and to fire Augment only for rows not in this set,
	// so widening or clearing a filter picks up augmentation for the
	// newly-revealed rows instead of leaving them stuck at their
	// placeholder forever. Reset (to an empty, non-nil map) by
	// applyListResult alongside augmentEpoch.
	augmentedRowIDs map[string]bool

	// settledRowIDs tracks which of augmentedRowIDs have actually FINISHED
	// — their owning batch reached its final tick and confirmed they were
	// never rejected by wanted — as opposed to merely requested. This is
	// the set that gets written to (and restored from) the list cache: a
	// row can be marked in augmentedRowIDs the instant its batch is
	// dispatched, well before it has any real data, so caching that set
	// directly would let a still-in-flight (or fallback-batch-dispatched)
	// row's placeholder get treated as settled forever if the user
	// navigates away and back within the cache TTL before that batch
	// finishes. Reset alongside augmentedRowIDs; always a subset of it.
	settledRowIDs map[string]bool

	// visibleRowIDs holds the current view's visible-row-ID set (map[string]bool),
	// updated by refreshTable every time it recomputes what's actually
	// shown (filter/facet/sort/base-row changes). It's an atomic.Value
	// rather than a plain map because Augmentable.Augment's wanted callback
	// reads it from arbitrary background goroutines — each Store swaps in a
	// brand new map, so a concurrent Load always sees a complete, never-
	// mutated-after-publish snapshot, no locking needed.
	visibleRowIDs atomic.Value

	// onAugmentRedrawForTest, if set, is called each time
	// triggerAugmentForNewlyVisibleRows actually performs the throttled
	// refreshTable+cache-write for an Augment tick (i.e. shouldRedrawAugmentTick
	// returned true) — a test-only seam for counting real redraws directly,
	// since wall-clock timing against a SimulationScreen doesn't reflect real
	// terminal draw cost. Always nil in production.
	onAugmentRedrawForTest func()

	activeContent tview.Primitive

	stopRefresh chan struct{}

	// stopStream, when non-nil, is the active detail stream's stop channel
	// (see runDetailStream) — closed by stopDetailStream whenever the
	// streamed view stops being the one on screen (navigation, error, a
	// newer load), which ends the underlying fetch. Like stopRefresh, only
	// ever touched on the UI goroutine.
	stopStream chan struct{}

	cache *listCache

	// historyRecorder is resolved once, in init(), from whatever resource is
	// registered under the name "history" (nil if none is — e.g. a minimal
	// test registry). Every recording call in loadList/loadDetail is a no-op
	// when this is nil.
	historyRecorder resource.HistoryRecorder

	// loadGeneration is incremented once per genuine navigation/render
	// dispatch of loadList/loadDetail (isInitial=true — a real navigation,
	// a restore-replay, or a facet-tab switch). A background refresh tick
	// (isInitial=false, used exclusively by Invalidate) does NOT increment
	// it — it captures whichever generation is already current instead,
	// inheriting the epoch of the view it's refreshing rather than
	// starting a new one. See nextLoadGeneration (navigation.go), the
	// single place that decides this capture behavior for both
	// loadList/loadDetail. Every dispatch's completion (refresh or
	// navigation) checks its captured generation against the current
	// value unconditionally: a captured value that no longer matches means
	// a newer navigation has started since — even one that later returns
	// to the identical View (isTopView would match again in that case, but
	// the generation correctly still doesn't) — and this completion must
	// no-op regardless of success or failure.
	//
	// Only ever mutated/read on tview's single event-loop goroutine (input
	// captures, Start's initial dispatch before app.Run(), and
	// QueueUpdateDraw callbacks are all serialized onto it) — never call
	// loadList/loadDetail from a raw `go` statement, or this increment
	// becomes a data race and two dispatches could capture the same value.
	loadGeneration int

	// restoreFallback is the resource Start falls back to once a restored
	// stack (if any) has been fully drained — either because it was empty to
	// begin with, or because every restored view turned out to be
	// unresolvable/stale. See renderRestoredTop/loadList/loadDetail's
	// isRestore argument for how an in-progress restore is now tracked (a
	// per-call argument, not a field).
	restoreFallback string

	// rootURL is set once via SetInfo and used to build web UI links for the
	// 'o' key (see openInBrowser).
	rootURL string

	// openBrowser is a seam over the package-level openBrowser func so tests
	// can capture what would be opened instead of shelling out for real.
	openBrowser func(url string) error
}

func New(registry *resource.Registry) *Shell {
	s := &Shell{
		app:              tview.NewApplication(),
		registry:         registry,
		stack:            NewViewStack(),
		sortByResource:   make(map[string]SortState),
		facetByResource:  make(map[string]string),
		filterByResource: make(map[string]string),
		loadAllKeys:      make(map[cacheKey]bool),
		cache:            newListCache(),
		openBrowser:      openBrowser,
	}
	s.init()

	return s
}

func (s *Shell) init() {
	if hr, ok := s.registry.Resolve("history"); ok {
		s.historyRecorder, _ = hr.(resource.HistoryRecorder)
	}

	s.headerLeft = tview.NewTextView().SetDynamicColors(true).SetWordWrap(true).
		SetChangedFunc(func() { s.app.Draw() })
	s.headerHint = tview.NewTextView().SetDynamicColors(true).SetWordWrap(true).
		SetTextAlign(tview.AlignLeft).
		SetChangedFunc(func() { s.app.Draw() })

	s.table = NewTableView()
	s.table.SetOnSelect(func(row resource.Row) {
		if row.NavTarget != nil {
			s.navigateTo(*row.NavTarget)
			return
		}
		s.showDetail(s.currentListResource, row.ID)
	})

	s.tabsBar = tview.NewTextView().SetDynamicColors(true)
	s.tabsSeparator = tview.NewTextView().SetDynamicColors(true).
		SetTextColor(tview.Styles.SecondaryTextColor)
	s.tableContainer = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.tabsBar, 0, 0, false).
		AddItem(s.tabsSeparator, 0, 0, false).
		AddItem(s.table, 0, 1, true)

	s.detail = NewDetailView()

	s.errorView = NewErrorView()
	s.helpView = NewHelpView()

	s.content = tview.NewPages().
		AddPage(pageTable, s.tableContainer, true, true).
		AddPage(pageDetail, s.detail, true, false).
		AddPage(pageError, s.errorView, true, false).
		AddPage(pageHelp, s.helpView, true, false)
	s.content.SetBorder(true)
	s.activeContent = s.table

	s.initFooter()

	s.root = tview.NewGrid().SetRows(3, 0, 1).SetColumns(-1, -1)
	s.root.AddItem(s.headerLeft, 0, 0, 1, 1, 0, 0, false)
	s.root.AddItem(s.headerHint, 0, 1, 1, 1, 0, 0, false)
	s.root.AddItem(s.content, 1, 0, 1, 2, 0, 0, true)
	s.root.AddItem(s.footer, 2, 0, 1, 2, 0, 0, false)

	s.app.SetRoot(s.root, true).SetFocus(s.content)
	s.app.SetInputCapture(s.globalInputCapture)
}

// globalInputCapture handles keys that apply regardless of which content
// view has focus: `q` quits from navigable views, `:` opens the command bar,
// `/` opens the filter (list views only), `?` toggles the help overlay, and
// Esc pops the view stack (a no-op at the root, or closes help if open).
// While the footer input is active, every key passes through untouched so it
// can be typed into the input field. While help is open, every key is
// swallowed except q, Esc/`?`, and the scroll keys.
func (s *Shell) globalInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if s.helpOpen {
		if s.footerMode == footerIdle && isQuitKey(event) {
			s.Stop()
			return nil
		}

		switch event.Key() {
		case tcell.KeyEscape:
			s.closeHelp()
			return nil
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyHome, tcell.KeyEnd:
			return event // let the HelpView TextView scroll
		}
		switch event.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone) // vim-style scroll
		case 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone) // vim-style scroll
		case '?':
			s.closeHelp()
		}
		return nil
	}

	if s.footerMode != footerIdle {
		return event
	}

	switch {
	case isQuitKey(event):
		s.Stop()
		return nil
	case event.Key() == tcell.KeyEscape:
		s.goBack()
		return nil
	case event.Rune() == ':':
		s.openCommandBar()
		return nil
	case event.Rune() == '/':
		if name, _ := s.content.GetFrontPage(); name == pageTable {
			s.openFilter()
		}
		return nil
	case event.Rune() >= '1' && event.Rune() <= '9':
		if name, _ := s.content.GetFrontPage(); name == pageTable {
			s.toggleSort(int(event.Rune() - '1'))
		}
		return nil
	case event.Key() == tcell.KeyTab:
		if name, _ := s.content.GetFrontPage(); name == pageTable {
			s.cycleFacet(1)
		}
		return nil
	case event.Key() == tcell.KeyBacktab:
		if name, _ := s.content.GetFrontPage(); name == pageTable {
			s.cycleFacet(-1)
		}
		return nil
	case event.Rune() == '?':
		s.openHelp()
		return nil
	case event.Rune() == 'r':
		s.refreshCurrent()
		return nil
	case event.Rune() == 'o':
		s.openInBrowser()
		return nil
	case event.Rune() == 's':
		s.promptSaveToDisk()
		return nil
	case event.Rune() == 'x':
		switch name, _ := s.content.GetFrontPage(); name {
		case pageTable:
			s.toggleExpandColumns()
		case pageDetail:
			s.toggleDetailWrap()
		}
		return nil
	// The condition is part of the case so that an 'L' pressed anywhere a
	// truncated list ISN'T front falls through to the detail-action keys
	// below instead of being swallowed.
	case event.Rune() == 'L' && s.canLoadAllRows():
		s.loadAllRows()
		return nil
	}

	if event.Key() == tcell.KeyRune {
		for _, action := range s.currentDetailActions {
			if event.Rune() == action.Key {
				s.navigateTo(action.Target)
				return nil
			}
		}
	}

	return event
}

func isQuitKey(event *tcell.EventKey) bool {
	return event.Key() == tcell.KeyRune && event.Rune() == 'q'
}

// hasFacets reports whether the current list view has a facet tab bar —
// either client-side (Faceted) or server-side (ServerFaceted).
func (s *Shell) hasFacets() bool {
	return s.currentFaceted != nil || s.currentServerFaceted != nil
}

func (s *Shell) setTitle(title string) {
	formatted := "[ Taskcluster"
	if title != "" {
		formatted += " :: " + title
	}
	formatted += " ]"
	s.content.SetTitle(formatted)
}

// updateBorderColor tints s.content's border blue while a filter is active
// on the currently visible list, so a filtered view is distinguishable at a
// glance rather than only via the title-bar query suffix. Blue rather than
// yellow, since yellow is already used for shortcut/header highlights
// elsewhere and wouldn't stand out here. The border is shared by every page
// in s.content (table/detail/error/help all live in the same Pages Box), so
// this must be recomputed any time either the front page or s.filterQuery
// changes — not just from refreshTable.
func (s *Shell) updateBorderColor() {
	front, _ := s.content.GetFrontPage()
	if front == pageTable && s.filterQuery != "" {
		s.content.SetBorderColor(tcell.ColorBlue)
	} else {
		s.content.SetBorderColor(tview.Styles.BorderColor)
	}
}

// SetInfo renders the persistent header bar's left block (Taskcluster
// root/version/client info), replacing the old ui.UI.SetTaskclusterInfo.
func (s *Shell) SetInfo(root, version, clientID string, authenticated bool) {
	s.rootURL = root

	clientColor := "yellow"
	clientExtra := ""
	if !authenticated {
		clientColor = "red"
		clientExtra = " (not authenticated)"
	}

	s.headerLeft.SetText(fmt.Sprintf(
		" [green]%s[white]\n Taskcluster version: [yellow]%s[white]\n Client ID: [%s]%s[gray]%s[white]",
		root, version, clientColor, clientID, clientExtra,
	))
}

// Start renders the app's initial view — the top of a stack restored via
// RestoreState, if one was, otherwise rootResource — and runs the tview event
// loop. It blocks until Stop() is called.
func (s *Shell) Start(rootResource string) error {
	s.restoreFallback = rootResource
	s.renderRestoredTop()
	return s.app.Run()
}

// StartAt behaves like Start, but jumps straight to name/scope (resolved the
// same way as the `:` command bar — see switchResource) instead of restored
// state or a fallback root resource, discarding any restored navigation
// stack. Used when the CLI is given a positional resource argument.
func (s *Shell) StartAt(name, scope string) error {
	s.switchResource(name, scope)
	return s.app.Run()
}

func (s *Shell) Stop() {
	s.stopRefreshLoop()
	s.app.Stop()
}

// openHelp swaps the content area to the help overlay, remembering which
// page was showing so closeHelp can restore it exactly. It does not touch
// s.stack or the active refresh loop — help is not a navigable place.
func (s *Shell) openHelp() {
	if s.helpOpen {
		return
	}

	s.preHelpPage, _ = s.content.GetFrontPage()
	s.helpOpen = true
	s.helpView.SetData(buildHelpText(s.registry))
	s.content.SwitchToPage(pageHelp)
	s.updateBorderColor()
	s.app.SetFocus(s.helpView)
}

// closeHelp restores whatever content page was showing before openHelp.
func (s *Shell) closeHelp() {
	if !s.helpOpen {
		return
	}

	s.helpOpen = false
	s.content.SwitchToPage(s.preHelpPage)
	s.updateBorderColor()
	s.app.SetFocus(s.activeContent)
}
