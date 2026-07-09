package resource

import (
	"fmt"
	"strings"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

type TaskResource struct {
	tc taskcluster.Taskcluster
}

func NewTaskResource(tc taskcluster.Taskcluster) *TaskResource {
	return &TaskResource{tc: tc}
}

func (r *TaskResource) Name() string          { return "task" }
func (r *TaskResource) Aliases() []string     { return nil }
func (r *TaskResource) Description() string   { return "A single task, looked up directly by task ID" }
func (r *TaskResource) IDPromptLabel() string { return "task id" }
func (r *TaskResource) Columns() []Column     { return nil }

// List is never expected to be called via normal navigation — DirectLookup
// resources never render a List view.
func (r *TaskResource) List() ([]Row, error) {
	return nil, fmt.Errorf("task requires a task id")
}

func (r *TaskResource) Describe(id string) (Detail, error) {
	return describeTask(r.tc, id)
}

func (r *TaskResource) RefreshInterval() time.Duration {
	return 15 * time.Second
}

type TasksResource struct {
	tc taskcluster.Taskcluster
}

func NewTasksResource(tc taskcluster.Taskcluster) *TasksResource {
	return &TasksResource{tc: tc}
}

func (r *TasksResource) Name() string      { return "tasks" }
func (r *TasksResource) Aliases() []string { return nil }
func (r *TasksResource) Description() string {
	return "Tasks belonging to one task group (scoped list)"
}

func (r *TasksResource) Columns() []Column {
	return []Column{
		{Title: "TASK ID"},
		{Title: "NAME", Width: 40},
		{Title: "STATE", Width: 12},
		{Title: "WORKER TYPE", Width: 24},
	}
}

// List is never expected to be called via normal navigation — the shell
// always either has a scope, or redirects to EmptyScopeResource() first.
func (r *TasksResource) List() ([]Row, error) {
	return nil, fmt.Errorf("tasks requires a task group scope")
}

func (r *TasksResource) ScopedList(taskGroupID string) ([]Row, error) {
	tasks, err := r.tc.GetTaskGroupTasks(taskGroupID)
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(tasks))
	for _, t := range tasks {
		rows = append(rows, Row{
			ID: t.Status.TaskID,
			Cells: []string{
				t.Status.TaskID,
				t.Task.Metadata.Name,
				t.Status.State,
				t.Task.WorkerType,
			},
		})
	}

	return rows, nil
}

func (r *TasksResource) EmptyScopeResource() string {
	return "workerpools"
}

func (r *TasksResource) Describe(id string) (Detail, error) {
	return describeTask(r.tc, id)
}

func (r *TasksResource) RefreshInterval() time.Duration {
	return 15 * time.Second
}

// describeTask renders a single task's full detail (definition + status),
// shared by TaskResource, TasksResource, and the pending/claimed task-queue
// resources — all of them ultimately show the same screen for a task ID.
func describeTask(tc taskcluster.Taskcluster, taskID string) (Detail, error) {
	task, err := tc.GetTask(taskID)
	if err != nil {
		return Detail{}, err
	}

	status, err := tc.GetTaskStatus(taskID)
	if err != nil {
		return Detail{}, err
	}

	var runs strings.Builder
	for _, run := range status.Runs {
		runs.WriteString(fmt.Sprintf("  run %d: [blue]%s[white]", run.RunID, run.State))
		if run.ReasonResolved != "" {
			runs.WriteString(fmt.Sprintf(" (reason: %s)", run.ReasonResolved))
		}
		if run.WorkerGroup != "" || run.WorkerID != "" {
			runs.WriteString(fmt.Sprintf(" (worker: %s/%s)", run.WorkerGroup, run.WorkerID))
		}
		runs.WriteString("\n")
	}
	if runs.Len() == 0 {
		runs.WriteString("  (no runs yet)\n")
	}

	body := fmt.Sprintf(
		"[green]Name:[white] %s\n"+
			"[green]Description:[white] %s\n"+
			"[green]Owner:[white] %s\n"+
			"[green]Source:[white] %s\n\n"+
			"[green]State:[blue] %s[white]\n"+
			"[green]Provisioner:[white] %s\n"+
			"[green]Worker Type:[white] %s\n"+
			"[green]Priority:[white] %s\n\n"+
			"[green]Created:[white] %s\n"+
			"[green]Deadline:[white] %s\n"+
			"[green]Expires:[white] %s\n\n"+
			"[green]Retries Left:[blue] %d[white]\n"+
			"[green]Dependencies (%d):[white]\n%s\n\n"+
			"[green]Scopes (%d):[white]\n%s\n\n"+
			"[green]Runs:[white]\n%s",
		task.Metadata.Name,
		task.Metadata.Description,
		task.Metadata.Owner,
		task.Metadata.Source,
		status.State,
		task.ProvisionerID,
		task.WorkerType,
		task.Priority,
		task.Created,
		task.Deadline,
		task.Expires,
		status.RetriesLeft,
		len(task.Dependencies),
		strings.Join(task.Dependencies, "\n"),
		len(task.Scopes),
		strings.Join(task.Scopes, "\n"),
		runs.String(),
	)

	return Detail{
		Title: fmt.Sprintf("Task :: %s (%s)", task.Metadata.Name, taskID),
		Body:  body,
		Actions: []DetailAction{
			{
				Key:   'g',
				Label: "task group",
				Target: NavTarget{
					ResourceName: "taskgroup",
					ID:           task.TaskGroupID,
					Kind:         NavDetail,
				},
			},
		},
	}, nil
}
