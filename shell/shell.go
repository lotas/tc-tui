package shell

import (
	"fmt"

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

	footer           *tview.Pages
	footerBreadcrumb *tview.TextView
	footerInput      *tview.InputField
	footerMode       footerMode
	pendingLookup    resource.Resource // set while footerMode == footerPrompt; the resource awaiting an id

	currentListResource  string
	currentColumns       []resource.Column
	lastRows             []resource.Row
	filterQuery          string
	filterByResource     map[string]string
	currentDetailActions []resource.DetailAction

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

	activeContent tview.Primitive

	stopRefresh chan struct{}

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
}

func New(registry *resource.Registry) *Shell {
	s := &Shell{
		app:              tview.NewApplication(),
		registry:         registry,
		stack:            NewViewStack(),
		sortByResource:   make(map[string]SortState),
		facetByResource:  make(map[string]string),
		filterByResource: make(map[string]string),
		cache:            newListCache(),
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
		if event.Rune() == '?' {
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

// SetInfo renders the persistent header bar's left block (Taskcluster
// root/version/client info), replacing the old ui.UI.SetTaskclusterInfo.
func (s *Shell) SetInfo(root, version, clientID string, authenticated bool) {
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
	s.app.SetFocus(s.helpView)
}

// closeHelp restores whatever content page was showing before openHelp.
func (s *Shell) closeHelp() {
	if !s.helpOpen {
		return
	}

	s.helpOpen = false
	s.content.SwitchToPage(s.preHelpPage)
	s.app.SetFocus(s.activeContent)
}
