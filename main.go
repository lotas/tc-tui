package main

import (
	"github.com/taskcluster/tc-tui/controller"
)

func main() {
	ctrl := controller.NewController()

	if err := ctrl.StartUI(); err != nil {
		panic(err)
	}
}

//
// func setViewCallback(name string, pageRenderer func() tview.Primitive) func() {
// 	return func() {
// 		lastPage = "main." + name
// 		if !pages.HasPage(lastPage) {
// 			pages.AddPage(lastPage, pageRenderer(), true, true)
// 		}
// 		pages.HidePage("info")
// 		pages.SwitchToPage(lastPage)
// 	}
// }
//
// func displayInfoPage(info string) {
// 	pages.SwitchToPage("info")
// 	infoPage.SetText(info)
// }
//
// func setAppTitle(title string) {
// 	formatted := "[ Taskcluster"
// 	if title != "" {
// 		formatted += " :: " + title
// 	}
// 	formatted += " ]"
// 	pages.SetTitle(formatted)
// }
//
// func backToMenu() {
// 	setAppTitle("")
// 	pages.SwitchToPage("menu")
// 	app.SetFocus(menu)
// }
//
// func renderWorkerPools() tview.Primitive {
// 	setAppTitle("Worker Pools")
//
// 	pools := tview.NewList()
// 	pools.AddItem("", "loading..", 0, nil)
//
// 	go func() {
// 		workerPools, err := wm.ListWorkerPools("", "100")
// 		if err != nil {
// 			panic(err)
// 		}
//
// 		pools.RemoveItem(0)
//
// 		for _, pool := range workerPools.WorkerPools {
// 			pools.AddItem(pool.ProviderID+" :: "+pool.WorkerPoolID, fmt.Sprintf("%d / %d", pool.CurrentCapacity, pool.RequestedCapacity), 0, nil)
// 		}
//
// 		pools.SetDoneFunc(backToMenu)
// 		app.Draw()
// 	}()
//
// 	return pools
// }
//
// func renderRoles() tview.Primitive {
// 	setAppTitle("Roles (0)")
// 	displayInfoPage("[gray] loading..[white]")
//
// 	rolesView := tview.NewList()
// 	rolesView.AddItem("", "loading..", 0, nil)
// 	rolesArr := make([]tcauth.GetRoleResponse, 0)
//
// 	go func() {
// 		cont := ""
//
// 		for {
// 			rolesResponse, err := auth.ListRoles2(cont, "150")
//
// 			if err != nil {
// 				displayInfoPage(fmt.Sprintf(" [red]Error:[white] %s", s.Replace(err.Error(), "\\n", "\n", -1)))
// 				break
// 			} else {
// 				rolesArr = append(rolesArr, rolesResponse.Roles...)
// 			}
//
// 			displayInfoPage(fmt.Sprintf(" [gray]loading.. [green]%d[white] roles", len(rolesArr)))
//
// 			if len(rolesArr) == 0 {
// 				displayInfoPage("[gray]No roles found[white]")
// 				break
// 			}
//
// 			if cont = rolesResponse.ContinuationToken; cont == "" {
// 				break
// 			}
//
// 			app.Draw()
// 		}
//
// 		setAppTitle(fmt.Sprintf("Roles (%d)", len(rolesArr)))
// 		rolesView.RemoveItem(0)
//
// 		for _, role := range rolesArr {
// 			rolesView.AddItem(role.RoleID, fmt.Sprintf("%s", role.Scopes), 0, nil)
// 		}
// 		rolesView.SetDoneFunc(backToMenu)
// 		pages.SwitchToPage("main.roles")
// 		app.Draw()
// 	}()
//
// 	return rolesView
// }
//
// func initTc() {
// 	auth = tcauth.NewFromEnv()
// 	wm = tcworkermanager.NewFromEnv()
//
// 	tcRoot = auth.RootURL
// 	if tcRoot == "" {
// 		panic("Root URL not defined. export TASKCLUSTER_ROOT_URL=x")
// 	}
//
// 	tcClientId = auth.Credentials.ClientID
// 	if tcClientId == "" {
// 		tcClientId = "anonymous"
// 	}
// 	tcVersion = getVersion()
//
// 	// check authentication
// 	authenticated = true
// 	clientColor := "yellow"
// 	clientExtra := ""
// 	_, err := auth.CurrentScopes()
// 	if err != nil {
// 		clientColor = "red"
// 		authenticated = false
// 		clientExtra = " (not authenticated)"
// 	}
//
// 	infoText := fmt.Sprintf(
// 		" Taskcluster version: [yellow]%s[white]\n Client ID: [%s]%s[gray]%s[white]",
// 		tcVersion.Version,
// 		clientColor,
// 		tcClientId,
// 		clientExtra)
// 	infoLeft.SetText(infoText)
//
// 	infoRight.SetText(fmt.Sprintf(" [green]%s[white] ", tcRoot))
//
// }
//
// func getVersion() Version {
// 	versionJson, err := getHttpResponse(tcRoot + "/__version__")
// 	if err != nil {
// 		panic(err)
// 	}
// 	ver := Version{}
// 	if err := json.Unmarshal([]byte(versionJson), &ver); err != nil {
// 		panic(err)
// 	}
//
// 	return ver
// }
//
// func getHttpResponse(url string) (string, error) {
// 	response, err := http.Get(url)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer response.Body.Close()
//
// 	contents, err := ioutil.ReadAll(response.Body)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	return string(contents), nil
// }
