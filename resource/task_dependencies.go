package resource

import (
	"fmt"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

// TaskDependenciesResource lists a task's dependencies, scoped by that
// task's own ID. Every row overrides navigation via NavTarget straight to
// the dependency's own Detail view (see Row.NavTarget), so Describe is
// never called and this resource is never reached other than via the task
// detail's 'd' action — Taskcluster's Queue API exposes only dependency IDs
// (task.Dependencies), not full task+status rows, so there's nothing richer
// to show per row.
type TaskDependenciesResource struct {
	tc taskcluster.Taskcluster
}

func NewTaskDependenciesResource(tc taskcluster.Taskcluster) *TaskDependenciesResource {
	return &TaskDependenciesResource{tc: tc}
}

func (r *TaskDependenciesResource) Name() string      { return "dependencies" }
func (r *TaskDependenciesResource) Aliases() []string { return nil }
func (r *TaskDependenciesResource) Description() string {
	return "A task's dependencies (scoped list) — select one to jump to it"
}

func (r *TaskDependenciesResource) Columns() []Column {
	return []Column{{Title: "DEPENDENCY TASK ID"}}
}

// List is never expected to be called via normal navigation — the shell
// always either has a scope, or redirects to EmptyScopeResource() first.
func (r *TaskDependenciesResource) List() ([]Row, error) {
	return nil, fmt.Errorf("dependencies requires a task scope")
}

func (r *TaskDependenciesResource) ScopedList(taskID string) ([]Row, error) {
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

func (r *TaskDependenciesResource) EmptyScopeResource() string {
	return "task"
}

// Describe is unreachable in normal use — see the type doc comment.
// Implemented only to satisfy the Resource interface.
func (r *TaskDependenciesResource) Describe(id string) (Detail, error) {
	return Detail{}, fmt.Errorf("dependency entries are not viewable directly")
}

func (r *TaskDependenciesResource) RefreshInterval() time.Duration { return 0 }

// ListWebURL links to the scoped task's own page — there's no dedicated
// dependencies page in the web UI (dependencies are shown inline on a task's
// own page), and scope is that task's ID.
func (r *TaskDependenciesResource) ListWebURL(rootURL, scope string) string {
	return taskWebURL(rootURL, scope)
}

// DetailWebURL is never expected to be called — see Describe's doc comment.
func (r *TaskDependenciesResource) DetailWebURL(rootURL, id string) string { return "" }

// TaskDependentsResource lists tasks that declare the scoped task as one of
// their own dependencies — the reverse direction of
// TaskDependenciesResource. Unlike dependencies, the Queue API's
// listDependentTasks returns full task+status rows in one call, so this
// reuses the same row shape (and Describe) as TasksResource rather than
// overriding navigation via NavTarget.
type TaskDependentsResource struct {
	tc taskcluster.Taskcluster
}

func NewTaskDependentsResource(tc taskcluster.Taskcluster) *TaskDependentsResource {
	return &TaskDependentsResource{tc: tc}
}

func (r *TaskDependentsResource) Name() string      { return "dependents" }
func (r *TaskDependentsResource) Aliases() []string { return nil }
func (r *TaskDependentsResource) Description() string {
	return "Tasks that declare this task as one of their own dependencies (scoped list)"
}

func (r *TaskDependentsResource) Columns() []Column {
	return taskListColumns()
}

// List is never expected to be called via normal navigation — the shell
// always either has a scope, or redirects to EmptyScopeResource() first.
func (r *TaskDependentsResource) List() ([]Row, error) {
	return nil, fmt.Errorf("dependents requires a task scope")
}

func (r *TaskDependentsResource) ScopedList(taskID string) ([]Row, error) {
	tasks, err := r.tc.GetDependentTasks(taskID)
	if err != nil {
		return nil, err
	}

	return taskListRows(tasks), nil
}

func (r *TaskDependentsResource) EmptyScopeResource() string {
	return "task"
}

func (r *TaskDependentsResource) Describe(id string) (Detail, error) {
	return describeTask(r.tc, id)
}

func (r *TaskDependentsResource) RefreshInterval() time.Duration {
	return 15 * time.Second
}

// ListWebURL links to the scoped task's own page — there's no dedicated
// dependents page in the web UI, and scope is that task's ID.
func (r *TaskDependentsResource) ListWebURL(rootURL, scope string) string {
	return taskWebURL(rootURL, scope)
}

func (r *TaskDependentsResource) DetailWebURL(rootURL, id string) string {
	return taskWebURL(rootURL, id)
}
