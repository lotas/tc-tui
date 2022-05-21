package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TcUI interface {
	SetTitle(string)
	ShowInfo(string)

	SetTaskclusterInfo(string, string, string, bool)

	Start() error
	Stop()
}

type UI struct {
	app       *tview.Application
	pages     *tview.Pages
	root      *tview.Grid
	menu      *tview.List
	infoLeft  *tview.TextView
	infoRight *tview.TextView
	infoPage  *tview.TextView

	title    string
	info     string
	lastPage string
}

func NewTcUI() TcUI {
	ui := &UI{
		app: tview.NewApplication(),
	}

	ui.init()

	return ui
}

func (ui *UI) SetTitle(title string) {
	formatted := "[ Taskcluster"
	if title != "" {
		formatted += " :: " + title
	}
	formatted += " ]"
	ui.pages.SetTitle(formatted)
}

func (ui *UI) ShowInfo(info string) {
	ui.info = info
}

func (ui *UI) Start() error {
	return ui.app.Run()
}

func (ui *UI) Stop() {
	ui.app.Stop()
}

func (ui *UI) SetTaskclusterInfo(root string, version string, clientID string, isAuthenticated bool) {
	clientColor := "yellow"
	clientExtra := ""
	if !isAuthenticated {
		clientColor = "red"
		clientExtra = " (not authenticated)"
	}

	ui.infoLeft.SetText(fmt.Sprintf(
		" Taskcluster version: [yellow]%s[white]\n Client ID: [%s]%s[gray]%s[white]",
		version,
		clientColor,
		clientID,
		clientExtra,
	))
 
  ui.infoRight.SetText(fmt.Sprintf(" [green]%s[white] ", root))
}

func (ui *UI) init() {
	ui.menu = tview.NewList().
		AddItem("Authenticate", "Signin", 0, nil).
		AddItem("Workers", "List workers", 'w', nil).
		AddItem("Worker Pools", "List pools", 'p', nil). // setViewCallback("worker-pools", renderWorkerPools)).
		AddItem("Roles", "List roles", 'r', nil).        // setViewCallback("roles", renderRoles)).
		AddItem("Scopes", "List scopes", 's', nil).
		AddItem("Quit", "Press to exit", 'q', func() {
			ui.app.Stop()
		})

	ui.infoPage = tview.NewTextView().SetDynamicColors(true).SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			if ui.lastPage != "" {
				ui.pages.SwitchToPage(ui.lastPage)
			} else {
				ui.backToMenu()
			}
		}
	})

	ui.pages = tview.NewPages().
		AddPage("info", ui.infoPage, true, false).
		AddPage("menu", ui.menu, true, true)

	ui.pages.SetBorder(true)

	ui.infoLeft = tview.NewTextView().SetDynamicColors(true).
		SetChangedFunc(func() {
			ui.app.Draw()
		}).
		SetText(" Taskcluster version: [yellow]fetching[white]")
	ui.infoRight = tview.NewTextView().SetDynamicColors(true).
		SetTextAlign(tview.AlignRight).
		SetChangedFunc(func() {
			ui.app.Draw()
		}).
		SetText(fmt.Sprintf(" Root: [green]%s[white]", ".."))

	ui.root = tview.NewGrid().SetRows(2, 0).SetColumns(0, 0)
	ui.root.AddItem(ui.infoLeft, 0, 0, 1, 1, 0, 0, false)
	ui.root.AddItem(ui.infoRight, 0, 1, 1, 1, 0, 0, false)
	ui.root.AddItem(ui.pages, 1, 0, 1, 2, 0, 0, true)

	ui.app.SetRoot(ui.root, true).SetFocus(ui.pages)

}

func (ui *UI) backToMenu() {
	ui.SetTitle("")
	ui.pages.SwitchToPage("menu")
	ui.app.SetFocus(ui.menu)
}
