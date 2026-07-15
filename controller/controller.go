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
	// StartUIAt behaves like StartUI, but jumps straight to a resource
	// name/alias (and optional scope/id), the way `:name scope` in the
	// command bar would — used for the CLI's positional arguments.
	StartUIAt(name, scope string) error
	// HelpText returns the same resource-derived help content shown by the
	// in-app '?' overlay, formatted for a plain terminal — used by the
	// CLI's --help.
	HelpText() string
}

type Controller struct {
	tc       taskcluster.Taskcluster
	shell    *shell.Shell
	registry *resource.Registry

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
	registry.Register(resource.NewWorkerRecentTasksResource(tc))
	registry.Register(resource.NewLaunchConfigsResource(tc))
	registry.Register(resource.NewErrorsResource(tc))
	registry.Register(resource.NewTaskResource(tc))
	registry.Register(resource.NewTaskGroupResource(tc))
	registry.Register(resource.NewTasksResource(tc))
	registry.Register(resource.NewTaskDependenciesResource(tc))
	registry.Register(resource.NewTaskDependentsResource(tc))
	registry.Register(resource.NewTaskRunsResource(tc))
	registry.Register(resource.NewTaskArtifactsResource(tc))
	registry.Register(resource.NewPendingTasksResource(tc))
	registry.Register(resource.NewClaimedTasksResource(tc))
	registry.Register(resource.NewHistoryResource())

	sh := shell.New(registry)

	statePath, err := state.Path(tc.GetRoot())
	if err == nil {
		sh.RestoreState(state.Load(statePath))
	}

	return &Controller{
		tc:        tc,
		shell:     sh,
		registry:  registry,
		statePath: statePath,
	}
}

func (c *Controller) StartUI() error {
	return c.run(func() error { return c.shell.Start(rootResource) })
}

func (c *Controller) StartUIAt(name, scope string) error {
	return c.run(func() error { return c.shell.StartAt(name, scope) })
}

func (c *Controller) HelpText() string {
	return shell.PlainHelpText(c.registry)
}

// run wires up header info and persists navigation state around whichever
// of Shell's blocking entry points (Start/StartAt) start renders the app.
func (c *Controller) run(start func() error) error {
	c.shell.SetInfo(c.tc.GetRoot(), "..", "..", false)

	go func() {
		c.shell.SetInfo(
			c.tc.GetRoot(),
			c.tc.GetVersion().Version,
			c.tc.GetClientID(),
			c.tc.IsAuthenticated(),
		)
	}()

	err := start()

	if c.statePath != "" {
		_ = state.Save(c.statePath, c.shell.ExportState())
	}

	return err
}
