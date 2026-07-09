package controller

import (
	"github.com/taskcluster/tc-tui/resource"
	"github.com/taskcluster/tc-tui/shell"
	"github.com/taskcluster/tc-tui/state"
	"github.com/taskcluster/tc-tui/taskcluster"
)

const rootResource = "workerpools"

type TcController interface {
	StartUI() error
}

type Controller struct {
	tc    taskcluster.Taskcluster
	shell *shell.Shell

	// statePath is "" if persisting navigation state is unavailable (e.g. the
	// OS user cache directory can't be resolved), in which case load/save are
	// both skipped and the app behaves as if this feature didn't exist.
	statePath string
}

func NewController() TcController {
	tc := taskcluster.NewTaskcluster()

	registry := resource.NewRegistry()
	registry.Register(resource.NewRolesResource(tc))
	registry.Register(resource.NewWorkerPoolsResource(tc))
	registry.Register(resource.NewWorkersResource(tc))
	registry.Register(resource.NewTaskResource(tc))
	registry.Register(resource.NewTaskGroupResource(tc))
	registry.Register(resource.NewTasksResource(tc))
	registry.Register(resource.NewPendingTasksResource(tc))
	registry.Register(resource.NewClaimedTasksResource(tc))

	sh := shell.New(registry)

	statePath, err := state.Path(tc.GetRoot())
	if err == nil {
		sh.RestoreState(state.Load(statePath))
	}

	return &Controller{
		tc:        tc,
		shell:     sh,
		statePath: statePath,
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

	err := c.shell.Start(rootResource)

	if c.statePath != "" {
		_ = state.Save(c.statePath, c.shell.ExportState())
	}

	return err
}
