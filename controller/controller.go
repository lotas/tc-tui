package controller

import (
	"fmt"

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
	rolesArr, err := c.tc.GetRoles()

	if err != nil {
		fmt.Printf("Error loading roles: %v", err)
		return
	}

	rows := make([]ui.UIListRow, 0)
	for _, role := range rolesArr {
		rows = append(rows, ui.UIListRow{
			PrimaryText:   role.RoleID,
			SecondaryText: fmt.Sprintf("%s", role.Scopes),
		})
	}

	c.ui.ListPage("Roles", rows)
}

func (c *Controller) StartUI() error {
	c.ShowInfo()

	return c.ui.Start()
}
