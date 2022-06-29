package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SelectedCallback func(int, string, string, rune)

type UIListRow struct {
	PrimaryText   string
	SecondaryText string
}

type TcUI interface {
	SetTitle(string)
	ShowInfo(string, string)

	SetTaskclusterInfo(string, string, string, bool)
	ListPage(string, []UIListRow, bool, UIEvent)

	Start() error
	Stop()
	Redraw()

	SetEventCallback(EventCallback)
}

type UI struct {
	app       *tview.Application
	pages     *tview.Pages
	root      *tview.Grid
	menu      *tview.List
	infoLeft  *tview.TextView
	infoRight *tview.TextView
	infoPage  *tview.TextView

	evtCb    EventCallback
	lastPage UIPage
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

func (ui *UI) ShowInfo(title string, info string) {
	ui.SetTitle(title)
	ui.infoPage.Clear().SetText(info).SetWordWrap(true)
	ui.pages.SwitchToPage(string(Info))
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
		AddItem("Authenticate", "Signin", 0, ui.eventHandler(Signin)).
		AddItem("Workers", "List workers", 'w', ui.eventHandler(ListWorkers)).
		AddItem("Worker Pools", "List pools", 'p', ui.eventHandler(ListWorkerPools)).
		AddItem("Roles", "List roles", 'r', ui.eventHandler(ListRoles)).
		AddItem("Scopes", "List scopes", 's', ui.eventHandler(ListScopes)).
		AddItem("Quit", "Press to exit", 'q', ui.eventHandler(Quit))

	ui.infoPage = tview.NewTextView().SetDynamicColors(true).SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			// TODO add event?
			if ui.lastPage != "" && ui.lastPage != Info {
				ui.SwitchPage(ui.lastPage)
			} else {
				ui.backToMenu()
			}
		}
	})

	ui.pages = tview.NewPages().
		AddPage(string(Info), ui.infoPage, true, false).
		AddPage(string(Menu), ui.menu, true, true)

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

func (ui *UI) ListPage(title string, rows []UIListRow, showSecondaryText bool, showType UIEvent) {
	ui.SetTitle(title)
	pageKey := fmt.Sprintf("list.%s", title)

	if ui.pages.HasPage(pageKey) {
		ui.pages.RemovePage(pageKey)
	}

	listView := tview.NewList()
	for _, row := range rows {
		listView.AddItem(row.PrimaryText, row.SecondaryText, 0, nil)
	}
	listView.SetDoneFunc(ui.backToMenu)
	listView.SetSelectedFunc(func(i int, s1, s2 string, r rune) {
		// todo move cb to be passed as arg
		ui.evtCb(showType, EventPayload{
			Index: i,
			Title: s1,
		})
	})
	listView.ShowSecondaryText(showSecondaryText)

	ui.pages.AddPage(pageKey, listView, true, true)
	ui.lastPage = UIPage(pageKey)
}

func (ui *UI) SwitchPage(page UIPage) {
	ui.lastPage = page
	ui.pages.SwitchToPage(string(page))
}

func (ui *UI) Redraw() {
	ui.app.Draw()
}

func (ui *UI) backToMenu() {
	ui.SetTitle("")
	ui.pages.SwitchToPage(string(Menu))
	ui.app.SetFocus(ui.menu)
}

func (ui *UI) SetEventCallback(cb EventCallback) {
	ui.evtCb = cb
}

func (ui *UI) eventHandler(evt UIEvent) func() {
	return func() {
		ui.evtCb(evt, EventPayload{})
	}
}
