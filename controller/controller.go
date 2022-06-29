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
	Roles       taskcluster.RolesList
	WorkerPools taskcluster.WorkerPoolList
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
	case ui.ListWorkerPools:
		c.ShowWorkerPools()
		return
	case ui.ShowWorkerPool:
		c.ShowWorkerPool(payload)
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

		c.ui.ListPage("Roles", rows, false, ui.ShowRole)
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

func (c *Controller) ShowWorkerPools() {
	c.ui.SetTitle("Loading worker pools")

	go func() {
		var err error
		workerPools := c.entities.WorkerPools

		if len(workerPools) == 0 {
			workerPools, err = c.tc.GetWorkerPools()
			if err != nil {
				c.ui.ShowInfo("Error loading worker pools", fmt.Sprintf("%s", err))
				return
			}
		}
		c.entities.WorkerPools = workerPools

		rows := make([]ui.UIListRow, 0)
		for _, pool := range workerPools {
			rows = append(rows, ui.UIListRow{
				PrimaryText:   pool.ProviderID + " :: " + pool.WorkerPoolID,
				SecondaryText: fmt.Sprintf("%d / %d", pool.CurrentCapacity, pool.RequestedCapacity),
			})
		}

		c.ui.ListPage("Worker Pools", rows, true, ui.ShowWorkerPool)
		c.ui.Redraw()
	}()
}

func (c *Controller) ShowWorkerPool(payload ui.EventPayload) {
	pool := c.entities.WorkerPools[payload.Index]

	info := fmt.Sprintf(
		"[green]Description:[white] %s\n\n"+
			"[green]Created:[white] %s\n"+
			"[green]Owner:[white] %s\n\n"+
			"[green]Requested capacity:[blue] %d\n"+
			"[green]Running capacity:[blue] %d\n"+
			"[green]Stopped capacity:[blue] %d\n"+
			"[green]Running count:[blue] %d\n"+
			"[green]Stopped count:[blue] %d\n\n"+
			"[green]Config:[white] %s\n\n",
		pool.Description,
		pool.Created,
		pool.Owner,
		pool.RequestedCapacity,
		pool.RunningCapacity,
		pool.StoppedCapacity,
		pool.RunningCount,
		pool.StoppedCount,
		pool.Config,
	)

	c.ui.ShowInfo(fmt.Sprintf("Worker Pool :: %s", payload.Title), info)
}

func (c *Controller) StartUI() error {
	c.ShowInfo()

	return c.ui.Start()
}
