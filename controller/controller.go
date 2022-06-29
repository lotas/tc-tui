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

type EntityCache struct {
	Roles taskcluster.RolesList
}

type Controller struct {
	ui ui.TcUI
	tc taskcluster.Taskcluster

	// Cache is needed to render individual entities from list
	// that can only be referenced by index id
	entities EntityCache
}

func NewController() TcController {
	ctrl := &Controller{
		ui:       ui.NewTcUI(),
		tc:       taskcluster.NewTaskcluster(),
		entities: EntityCache{},
	}
	ctrl.ui.SetEventCallback(ctrl.eventHandler)

	return ctrl
}

func (c *Controller) eventHandler(evt ui.UIEvent, payload ui.EventPayload) {
	switch evt {
	case ui.Quit:
		c.ui.Stop()
		return
	case ui.ListRoles:
		c.ShowRoles()
		return
	case ui.ShowRole:
		c.ShowRole(payload)
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
		var err error
		rolesArr := c.entities.Roles

		if len(rolesArr) == 0 {
			rolesArr, err = c.tc.GetRoles()
			if err != nil {
				c.ui.ShowInfo("Error loading roles", fmt.Sprintf("%s", err))
				return
			}
		}
		c.entities.Roles = rolesArr

		rows := make([]ui.UIListRow, 0)
		for _, role := range rolesArr {
			rows = append(rows, ui.UIListRow{
				PrimaryText:   role.RoleID,
				SecondaryText: "",
			})
		}

		c.ui.ListPage("Roles", rows, false)
		c.ui.Redraw()
	}()
}

func (c *Controller) ShowRole(payload ui.EventPayload) {
	role := c.entities.Roles[payload.Index]

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

	c.ui.ShowInfo(fmt.Sprintf("Role :: %s", payload.Title), info)
}

func (c *Controller) StartUI() error {
	c.ShowInfo()

	return c.ui.Start()
}
