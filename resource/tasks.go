package resource

import (
	"fmt"
	"strings"
	"time"

	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcqueue"

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

// ListWebURL is never expected to be called — TaskResource is a
// DirectLookup and never renders a List view.
func (r *TaskResource) ListWebURL(rootURL, scope string) string { return "" }

func (r *TaskResource) DetailWebURL(rootURL, id string) string {
	return taskWebURL(rootURL, id)
}

type TasksResource struct {
	tc taskcluster.Taskcluster
}

func NewTasksResource(tc taskcluster.Taskcluster) *TasksResource {
	return &TasksResource{tc: tc}
}

func (r *TasksResource) Name() string      { return "tasks" }
func (r *TasksResource) Aliases() []string { return []string{"t"} }
func (r *TasksResource) Description() string {
	return "Tasks belonging to one task group (scoped list)"
}

func (r *TasksResource) Columns() []Column {
	return taskListColumns()
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

	return taskListRows(tasks), nil
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

// ListWebURL links to the task group's own page — scope is the task group
// ID this list is scoped to.
func (r *TasksResource) ListWebURL(rootURL, scope string) string {
	return taskGroupWebURL(rootURL, scope)
}

func (r *TasksResource) DetailWebURL(rootURL, id string) string {
	return taskWebURL(rootURL, id)
}

// taskListColumns is the column set shared by every list view whose rows
// are built by taskListRows (TasksResource, TaskDependentsResource).
func taskListColumns() []Column {
	return []Column{
		{Title: "TASK ID", Width: taskIDColumnWidth},
		{Title: "NAME"},
		{Title: "STATE", Width: 12},
		{Title: "WORKER POOL", Width: workerPoolColumnWidth},
		{Title: "AGE", Width: 12},
	}
}

// taskListRows builds list rows from a raw task+status list, shared by
// TasksResource.ScopedList and TaskDependentsResource.ScopedList.
func taskListRows(tasks taskcluster.TaskGroupTaskList) []Row {
	rows := make([]Row, 0, len(tasks))
	for _, t := range tasks {
		rows = append(rows, Row{
			ID: t.Status.TaskID,
			Cells: []string{
				t.Status.TaskID,
				t.Task.Metadata.Name,
				renderTaskState(t.Status.State),
				t.Task.ProvisionerID + "/" + t.Task.WorkerType,
				formatAge(time.Time(t.Task.Created)),
			},
		})
	}
	return rows
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
		runs.WriteString(renderRunBody(tc, taskID, run, "  "))
	}
	if runs.Len() == 0 {
		runs.WriteString("  (no runs yet)\n")
	}

	body := fmt.Sprintf(
		"[green]Name:[white] %s\n"+
			"[green]Description:[white]\n%s\n\n"+
			"%s\n"+
			"[green]State:[%s] %s[white]\n"+
			"%s\n"+
			"[green]Payload:[white]\n%s\n\n"+
			"%s\n"+
			"[green]Retries Left:[blue] %d[white]\n"+
			"[green]Dependencies (%d):[white]\n%s\n\n"+
			"[green]Scopes (%d):[white]\n%s\n\n"+
			"[green]Runs:[white]\n%s",
		task.Metadata.Name,
		renderMarkdown(task.Metadata.Description),
		fieldRow(32, "Owner", task.Metadata.Owner, "Source", task.Metadata.Source),
		taskStateColor(status.State), status.State,
		fieldRow(24, "Provisioner", task.ProvisionerID, "Worker Type", task.WorkerType, "Priority", task.Priority),
		renderYAML(task.Payload),
		fieldRow(30, "Created", fmt.Sprint(task.Created), "Deadline", fmt.Sprint(task.Deadline), "Expires", fmt.Sprint(task.Expires)),
		status.RetriesLeft,
		len(task.Dependencies),
		strings.Join(task.Dependencies, "\n"),
		len(task.Scopes),
		strings.Join(task.Scopes, "\n"),
		runs.String(),
	)

	actions := []DetailAction{
		{
			Key:   'W',
			Label: "worker pool",
			Target: NavTarget{
				ResourceName: "workerpools",
				ID:           task.ProvisionerID + "/" + task.WorkerType,
				Kind:         NavDetail,
			},
		},
		{
			Key:   'g',
			Label: "task group",
			Target: NavTarget{
				ResourceName: "taskgroup",
				ID:           task.TaskGroupID,
				Kind:         NavScopedList,
			},
		},
	}
	if len(task.Dependencies) > 0 {
		actions = append(actions, DetailAction{
			Key:   'd',
			Label: "dependencies",
			Target: NavTarget{
				ResourceName: "dependencies",
				ID:           taskID,
				Kind:         NavScopedList,
			},
		})
	}
	// Unlike Dependencies, there's no cheap way to know a task's dependent
	// count up front (listDependentTasks has no counts-only variant), so
	// this action is always shown — selecting it may just land on an empty
	// list, same as Taskcluster's own web UI.
	actions = append(actions, DetailAction{
		Key:   'D',
		Label: "dependents",
		Target: NavTarget{
			ResourceName: "dependents",
			ID:           taskID,
			Kind:         NavScopedList,
		},
	})
	if len(status.Runs) > 0 {
		actions = append(actions, DetailAction{
			Key:   'R',
			Label: "runs",
			Target: NavTarget{
				ResourceName: "runs",
				ID:           taskID,
				Kind:         NavScopedList,
			},
		})
	}
	// Always shown, like dependents — TaskArtifactsResource is scoped by
	// task (not by a single run), tabbing over whichever runs exist, so
	// there's no need to gate this on run count the way 'R' is; selecting it
	// for a task with no runs yet just lands on an empty list.
	actions = append(actions, DetailAction{
		Key:   'a',
		Label: "artifacts",
		Target: NavTarget{
			ResourceName: "artifacts",
			ID:           taskID,
			Kind:         NavScopedList,
		},
	})

	return Detail{
		Title:   fmt.Sprintf("Task :: %s%s (%s)", taskStateBadge(status.State), task.Metadata.Name, taskID),
		Body:    body,
		Actions: actions,
	}, nil
}

// renderRunBody renders one run's state, timestamps, and (if started)
// artifacts, shared by describeTask's inline runs section (indent "  ") and
// TaskRunsResource.Describe's single-run detail view (indent "").
func renderRunBody(tc taskcluster.Taskcluster, taskID string, run tcqueue.RunInformation, indent string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%srun %d: [%s]%s[white]", indent, run.RunID, taskStateColor(run.State), run.State))
	if run.ReasonResolved != "" {
		b.WriteString(fmt.Sprintf(" (reason: %s)", run.ReasonResolved))
	}
	if run.WorkerGroup != "" || run.WorkerID != "" {
		b.WriteString(fmt.Sprintf(" (worker: %s/%s)", run.WorkerGroup, run.WorkerID))
	}
	b.WriteString("\n")

	fieldIndent := indent + "  "
	b.WriteString(fmt.Sprintf("%sscheduled: %s\n", fieldIndent, run.Scheduled))
	if !time.Time(run.Started).IsZero() {
		b.WriteString(fmt.Sprintf("%sstarted:   %s%s\n", fieldIndent, run.Started,
			elapsedSince(time.Time(run.Scheduled), time.Time(run.Started), "scheduled")))
	}
	if !time.Time(run.Resolved).IsZero() {
		b.WriteString(fmt.Sprintf("%sresolved:  %s%s\n", fieldIndent, run.Resolved,
			elapsedSince(time.Time(run.Started), time.Time(run.Resolved), "started")))
	}
	if !time.Time(run.TakenUntil).IsZero() {
		b.WriteString(fmt.Sprintf("%stakenUntil:%s\n", fieldIndent, run.TakenUntil))
	}
	if !time.Time(run.Started).IsZero() {
		b.WriteString(renderRunArtifacts(tc, taskID, run.RunID))
	}
	return b.String()
}

// renderRunArtifacts lists the artifacts produced by one run, indented to
// nest under that run's timestamps. A fetch failure is shown inline rather
// than failing the whole task detail, since the rest of the task is still
// useful without it.
func renderRunArtifacts(tc taskcluster.Taskcluster, taskID string, runID int64) string {
	artifacts, err := tc.GetArtifacts(taskID, runID)
	if err != nil {
		return fmt.Sprintf("    artifacts: (failed to load: %s)\n", err)
	}
	if len(artifacts) == 0 {
		return "    artifacts: (none)\n"
	}

	var b strings.Builder
	b.WriteString("    artifacts:\n")
	for _, a := range artifacts {
		b.WriteString(fmt.Sprintf("      %s (%s, %s)\n", a.Name, a.ContentType, formatBytes(a.ContentLength)))
	}
	return b.String()
}
