package controller

import (
	"fmt"
	"strings"

	"github.com/taskcluster/tc-tui/taskcluster"
	"github.com/taskcluster/tc-tui/ui"
)

type TcController interface {
	ShowInfo()

	StartUI() error
}

type Controller struct {
	ui ui.TcUI
	tc taskcluster.Taskcluster
}

func NewController() TcController {
	ctrl := &Controller{
		ui: ui.NewTcUI(),
		tc: taskcluster.NewTaskcluster(),
	}
	ctrl.ui.SetEventCallback(ctrl.eventHandler)

	return ctrl
}

func (c *Controller) eventHandler(evt ui.UIEvent, payload interface{}) {
	switch evt {
	case ui.Quit:
		c.ui.Stop()
		return
	case ui.ListRoles:
		c.ShowRoles()
		return
	}

	fmt.Printf("Unknown event %v", evt)
}

func (c *Controller) ShowInfo() {
	c.ui.SetTaskclusterInfo("..", "..", "..", false)

	go func() {
		c.ui.SetTaskclusterInfo(
			c.tc.GetRoot(),
			c.tc.GetVersion().Version,
			c.tc.GetClientID(),
			c.tc.IsAuthenticated(),
		)
	}()
}

func (c *Controller) ShowRoles() {
	c.ui.SetTitle("Loading roles")

	go func() {
		rolesArr, err := c.tc.GetRoles()

		if err != nil {
			c.ui.ShowInfo("Error loading roles", fmt.Sprintf("%s", err))
			return
		}

		rows := make([]ui.UIListRow, 0)
		for _, role := range rolesArr {
			rows = append(rows, ui.UIListRow{
				PrimaryText:   role.RoleID,
				SecondaryText: "",
			})
		}

		c.ui.ListPage("Roles", rows, false, func(i int, s1, s2 string, r rune) {
			role := rolesArr[i]
			info := fmt.Sprintf(
				"[green]Description:[white] %s\n\n[green]Created:[white] %s\n"+
					"[green]Last Modified:[white] %s\n\n[green]Scopes (%d):[white]\n\n%s"+
					"\n\n[green]Expanded Scopes (%d):[white]\n\n%s",
				role.Description,
				role.Created,
				role.LastModified,
				len(role.Scopes),
				strings.Join(role.Scopes[:], "\n"),
				len(role.ExpandedScopes),
				strings.Join(role.ExpandedScopes[:], "\n"),
			)
			c.ui.ShowInfo(s1, info)
		})
	}()
}

func (c *Controller) StartUI() error {
	c.ShowInfo()

	return c.ui.Start()
}
