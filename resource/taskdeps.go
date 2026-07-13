package resource

import (
	"fmt"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

// TaskDepsResource lists a task's dependencies, scoped by that task's own
// ID. Every row overrides navigation via NavTarget straight to the
// dependency's own Detail view (see Row.NavTarget), so Describe is never
// called and this resource is never registered under a command-bar-visible
// name of its own — it's reached only via the task detail's 'd' action.
type TaskDepsResource struct {
	tc taskcluster.Taskcluster
}

func NewTaskDepsResource(tc taskcluster.Taskcluster) *TaskDepsResource {
	return &TaskDepsResource{tc: tc}
}

func (r *TaskDepsResource) Name() string      { return "taskdeps" }
func (r *TaskDepsResource) Aliases() []string { return nil }
func (r *TaskDepsResource) Description() string {
	return "A task's dependencies (scoped list) — select one to jump to it"
}

func (r *TaskDepsResource) Columns() []Column {
	return []Column{{Title: "DEPENDENCY TASK ID"}}
}

// List is never expected to be called via normal navigation — the shell
// always either has a scope, or redirects to EmptyScopeResource() first.
func (r *TaskDepsResource) List() ([]Row, error) {
	return nil, fmt.Errorf("taskdeps requires a task scope")
}

func (r *TaskDepsResource) ScopedList(taskID string) ([]Row, error) {
	task, err := r.tc.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(task.Dependencies))
	for _, depID := range task.Dependencies {
		rows = append(rows, Row{
			ID:        depID,
			Cells:     []string{depID},
			NavTarget: &NavTarget{ResourceName: "task", ID: depID, Kind: NavDetail},
		})
	}

	return rows, nil
}

func (r *TaskDepsResource) EmptyScopeResource() string {
	return "task"
}

// Describe is unreachable in normal use — see the type doc comment.
// Implemented only to satisfy the Resource interface.
func (r *TaskDepsResource) Describe(id string) (Detail, error) {
	return Detail{}, fmt.Errorf("taskdeps entries are not viewable directly")
}

func (r *TaskDepsResource) RefreshInterval() time.Duration { return 0 }
