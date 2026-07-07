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

	footer      *tview.Pages
	footerHint  *tview.TextView
	footerInput *tview.InputField
	footerMode  footerMode

	currentListResource string
	currentColumns      []resource.Column
	lastRows            []resource.Row
	filterQuery         string

	activeContent tview.Primitive

	stopRefresh chan struct{}
}

func New(registry *resource.Registry) *Shell {
	s := &Shell{
		app:      tview.NewApplication(),
		registry: registry,
		stack:    NewViewStack(),
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

	s.content = tview.NewPages().
		AddPage(pageTable, s.table, true, true).
		AddPage(pageDetail, s.detail, true, false).
		AddPage(pageError, s.errorView, true, false)
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
// view has focus: `:` opens the command bar, `/` opens the filter (list
// views only), Esc pops the view stack (or quits, at the root). While the
// footer input is active, every key passes through untouched so it can be
// typed into the input field.
func (s *Shell) globalInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if s.footerMode != footerHints {
		return event
	}

	switch {
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
	}

	return event
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
