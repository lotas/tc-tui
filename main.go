package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/rivo/tview"

	"net/http"

	tcauth "github.com/taskcluster/taskcluster/v44/clients/client-go/tcauth"
	tcworkermanager "github.com/taskcluster/taskcluster/v44/clients/client-go/tcworkermanager"
)

type Version struct {
	Source  string `json:"source"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Build   string `json:"build"`
}

var (
	app   *tview.Application
	pages *tview.Pages
	auth  *tcauth.Auth
	root  *tview.Grid
	menu  *tview.List
	info  *tview.TextView

	wm        *tcworkermanager.WorkerManager
	tcVersion Version
	tcRoot    string
)

func main() {
	app = tview.NewApplication()

	initUI()
	go initTc()

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func initUI() {
	// TODO: add "/" shortcut to search/filter items on the page

	menu = tview.NewList().
		AddItem("Workers", "List workers", 'w', nil).
		AddItem("Worker Pools", "List pools", 'p', setViewCallback("worker-pools", renderWorkerPools)).
		AddItem("Roles", "List roles", 'r', setViewCallback("roles", renderRoles)).
		AddItem("Scopes", "List scopes", 's', nil).
		AddItem("Quit", "Press to exit", 'q', func() {
			app.Stop()
		})

	pages = tview.NewPages().AddPage("main.menu", menu, true, true)
	pages.SetBorder(true)
	pages.SetTitle("[ Taskcluster ]")

	info = tview.NewTextView().SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetText(" Taskcluster version: [green]fetching[white]")

	root = tview.NewGrid().SetRows(3, 0).SetColumns(0, 0)
	root.AddItem(info, 0, 0, 1, 1, 0, 0, false)
	// root.AddItem(menu, 0, 1, 1, 1, 0, 0, false)
	root.AddItem(pages, 1, 0, 1, 2, 0, 0, true)

	app.SetRoot(root, true).SetFocus(pages)
}

func setViewCallback(name string, pageRenderer func() tview.Primitive) func() {
	return func() {
		if pages.HasPage("main." + name) {
			pages.SwitchToPage("main." + name)
		}
		pages.AddPage("main."+name, pageRenderer(), true, true)
	}
}

func renderWorkerPools() tview.Primitive {
	pages.SetTitle("[ Taskcluster :: Worker Pools ]")

	pools := tview.NewList()
	pools.AddItem("", "loading..", 0, nil)

	go func() {
		workerPools, err := wm.ListWorkerPools("", "100")
		if err != nil {
			panic(err)
		}

		pools.RemoveItem(0)

		for _, pool := range workerPools.WorkerPools {
			pools.AddItem(pool.ProviderID+" :: "+pool.WorkerPoolID, fmt.Sprintf("%d / %d", pool.CurrentCapacity, pool.RequestedCapacity), 0, nil)
		}

		pools.SetDoneFunc(func() {
			pages.SwitchToPage("menu")
			app.SetFocus(menu)
		})

		app.Draw()
	}()

	return pools
}

func renderRoles() tview.Primitive {
	pages.SetTitle("[ Taskcluster :: Roles ]")

	rolesView := tview.NewList()
	rolesView.AddItem("", "loading..", 0, nil)

	go func() {
		rolesResponse, err := auth.ListRoles2("", "500")
		if err != nil {
			panic(err)
		}

		rolesView.RemoveItem(0)

		for _, role := range rolesResponse.Roles {
			rolesView.AddItem(role.RoleID, fmt.Sprintf("%s", role.Scopes), 0, nil)
		}

		rolesView.SetDoneFunc(func() {
			pages.SwitchToPage("menu")
			app.SetFocus(menu)
		})

		app.Draw()
	}()

	return rolesView
}

func initTc() {
	auth = tcauth.NewFromEnv()
	wm = tcworkermanager.NewFromEnv()

	tcRoot = auth.RootURL
	tcVersion = getVersion()

	infoText := fmt.Sprintf(" Taskluster version: [green]%s[white]\n Root: [yellow]%s[white]\n ", tcVersion.Version, tcRoot)
	info.SetText(infoText)
	// menu.SetItemText(0, "Taskcluster info", tcVersion.Version+" :: "+tcRoot)
	// app.Draw()
}

func getVersion() Version {
	versionJson, err := getHttpResponse(tcRoot + "/__version__")
	if err != nil {
		panic(err)
	}
	ver := Version{}
	if err := json.Unmarshal([]byte(versionJson), &ver); err != nil {
		panic(err)
	}

	return ver
}

func getHttpResponse(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(contents), nil
}
