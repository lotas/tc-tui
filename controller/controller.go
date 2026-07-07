package controller

import (
	"github.com/taskcluster/tc-tui/resource"
	"github.com/taskcluster/tc-tui/shell"
	"github.com/taskcluster/tc-tui/taskcluster"
)

const rootResource = "workerpools"

type TcController interface {
	StartUI() error
}

type Controller struct {
	tc    taskcluster.Taskcluster
	shell *shell.Shell
}

func NewController() TcController {
	tc := taskcluster.NewTaskcluster()

	registry := resource.NewRegistry()
	registry.Register(resource.NewRolesResource(tc))
	registry.Register(resource.NewWorkerPoolsResource(tc))

	return &Controller{
		tc:    tc,
		shell: shell.New(registry),
	}
}

func (c *Controller) StartUI() error {
	c.shell.SetInfo(c.tc.GetRoot(), "..", "..", false)

	go func() {
		c.shell.SetInfo(
			c.tc.GetRoot(),
			c.tc.GetVersion().Version,
			c.tc.GetClientID(),
			c.tc.IsAuthenticated(),
		)
	}()

	return c.shell.Start(rootResource)
}
