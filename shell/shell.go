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

	pageFooterHints = "hints"
	pageFooterInput = "input"
)

type footerMode int

const (
	footerHints footerMode = iota
	footerCommand
	footerFilter
)

// Shell is the generic navigation engine: registry, view stack, table/detail
// views, command bar, filter, refresh loop. It knows nothing about roles or
// worker pools specifically — only the Resource interface.
type Shell struct {
	app      *tview.Application
	registry *resource.Registry
	stack    *ViewStack

	root        *tview.Grid
	headerLeft  *tview.TextView
	headerRight *tview.TextView

	content   *tview.Pages
	table     *TableView
	detail    *DetailView
	errorView *ErrorView
	helpView  *HelpView

	helpOpen    bool
	preHelpPage string

	footer      *tview.Pages
	footerHint  *tview.TextView
	footerInput *tview.InputField
	footerMode  footerMode

	currentListResource  string
	currentColumns       []resource.Column
	lastRows             []resource.Row
	filterQuery          string
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
}

func New(registry *resource.Registry) *Shell {
	s := &Shell{
		app:             tview.NewApplication(),
		registry:        registry,
		stack:           NewViewStack(),
		sortByResource:  make(map[string]SortState),
		facetByResource: make(map[string]string),
	}
	s.init()

	return s
}

func (s *Shell) init() {
	s.headerLeft = tview.NewTextView().SetDynamicColors(true).
		SetChangedFunc(func() { s.app.Draw() })
	s.headerRight = tview.NewTextView().SetDynamicColors(true).
		SetTextAlign(tview.AlignRight).
		SetChangedFunc(func() { s.app.Draw() })

	s.table = NewTableView()
	s.table.SetOnSelect(func(id string) {
		s.showDetail(s.currentListResource, id)
	})

	s.tabsBar = tview.NewTextView().SetDynamicColors(true)
	s.tabsSeparator = tview.NewTextView().SetDynamicColors(true).
		SetTextColor(tview.Styles.SecondaryTextColor)
	s.tableContainer = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.tabsBar, 0, 0, false).
		AddItem(s.tabsSeparator, 0, 0, false).
		AddItem(s.table, 0, 1, true)

	s.detail = NewDetailView()
	s.detail.SetOnAction(func(target resource.NavTarget) {
		switch target.Kind {
		case resource.NavScopedList:
			s.pushScopedList(target.ResourceName, target.ID)
		default:
			s.showDetail(target.ResourceName, target.ID)
		}
	})

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

	s.root = tview.NewGrid().SetRows(2, 0, 1).SetColumns(0, 0)
	s.root.AddItem(s.headerLeft, 0, 0, 1, 1, 0, 0, false)
	s.root.AddItem(s.headerRight, 0, 1, 1, 1, 0, 0, false)
	s.root.AddItem(s.content, 1, 0, 1, 2, 0, 0, true)
	s.root.AddItem(s.footer, 2, 0, 1, 2, 0, 0, false)

	s.app.SetRoot(s.root, true).SetFocus(s.content)
	s.app.SetInputCapture(s.globalInputCapture)
}

// globalInputCapture handles keys that apply regardless of which content
// view has focus: `q` quits from navigable views, `:` opens the command bar,
// `/` opens the filter (list views only), `?` toggles the help overlay, and
// Esc pops the view stack (or quits at the root, or closes help if open).
// While the footer input is active, every key passes through untouched so it
// can be typed into the input field. While help is open, every key is
// swallowed except q, Esc/`?`, and the scroll keys.
func (s *Shell) globalInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if s.helpOpen {
		if s.footerMode == footerHints && isQuitKey(event) {
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

	if s.footerMode != footerHints {
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

// SetInfo renders the persistent header bar (Taskcluster root/version/client
// info), replacing the old ui.UI.SetTaskclusterInfo.
func (s *Shell) SetInfo(root, version, clientID string, authenticated bool) {
	clientColor := "yellow"
	clientExtra := ""
	if !authenticated {
		clientColor = "red"
		clientExtra = " (not authenticated)"
	}

	s.headerLeft.SetText(fmt.Sprintf(
		" Taskcluster version: [yellow]%s[white]\n Client ID: [%s]%s[gray]%s[white]",
		version, clientColor, clientID, clientExtra,
	))
	s.headerRight.SetText(fmt.Sprintf(" [green]%s[white] ", root))
}

// Start pushes the given resource as the root view and runs the tview event
// loop. It blocks until Stop() is called.
func (s *Shell) Start(rootResource string) error {
	s.switchResource(rootResource, "")
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
