package resource

import (
	"fmt"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

type PendingTasksResource struct {
	tc taskcluster.Taskcluster
}

func NewPendingTasksResource(tc taskcluster.Taskcluster) *PendingTasksResource {
	return &PendingTasksResource{tc: tc}
}

func (r *PendingTasksResource) Name() string      { return "pending" }
func (r *PendingTasksResource) Aliases() []string { return nil }
func (r *PendingTasksResource) Description() string {
	return "Tasks currently pending on a worker pool's task queue (scoped list)"
}

func (r *PendingTasksResource) Columns() []Column {
	return []Column{
		{Title: "TASK ID"},
		{Title: "NAME", Width: 40},
		{Title: "WORKER TYPE", Width: 24},
		{Title: "INSERTED", Width: 24},
		{Title: "AGE", Width: 12},
	}
}

func (r *PendingTasksResource) List() ([]Row, error) {
	return nil, fmt.Errorf("pending requires a task queue scope")
}

func (r *PendingTasksResource) ScopedList(taskQueueID string) ([]Row, error) {
	tasks, err := r.tc.GetPendingTasks(taskQueueID)
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(tasks))
	for _, t := range tasks {
		rows = append(rows, Row{
			ID: t.TaskID,
			Cells: []string{
				t.TaskID,
				t.Task.Metadata.Name,
				t.Task.WorkerType,
				t.Inserted.String(),
				formatAge(time.Time(t.Inserted)),
			},
		})
	}

	return rows, nil
}

func (r *PendingTasksResource) EmptyScopeResource() string {
	return "workerpools"
}

// ScopeActions returns the worker-pool sibling jump keys (minus "pending"
// itself) for scope — see resource.ScopeActions.
func (r *PendingTasksResource) ScopeActions(scope string) []DetailAction {
	workerPoolID, _ := parseScope(scope)
	return workerPoolActions(workerPoolID, r.Name())
}

func (r *PendingTasksResource) Describe(id string) (Detail, error) {
	return describeTask(r.tc, id)
}

func (r *PendingTasksResource) RefreshInterval() time.Duration {
	return 15 * time.Second
}

type ClaimedTasksResource struct {
	tc taskcluster.Taskcluster
}

func NewClaimedTasksResource(tc taskcluster.Taskcluster) *ClaimedTasksResource {
	return &ClaimedTasksResource{tc: tc}
}

func (r *ClaimedTasksResource) Name() string      { return "claimed" }
func (r *ClaimedTasksResource) Aliases() []string { return nil }
func (r *ClaimedTasksResource) Description() string {
	return "Tasks currently claimed (running) on a worker pool's task queue (scoped list)"
}

func (r *ClaimedTasksResource) Columns() []Column {
	return []Column{
		{Title: "TASK ID"},
		{Title: "NAME", Width: 40},
		{Title: "WORKER GROUP/ID", Width: 30},
		{Title: "CLAIMED", Width: 24},
		{Title: "AGE", Width: 12},
	}
}

func (r *ClaimedTasksResource) List() ([]Row, error) {
	return nil, fmt.Errorf("claimed requires a task queue scope")
}

func (r *ClaimedTasksResource) ScopedList(taskQueueID string) ([]Row, error) {
	tasks, err := r.tc.GetClaimedTasks(taskQueueID)
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(tasks))
	for _, t := range tasks {
		rows = append(rows, Row{
			ID: t.TaskID,
			Cells: []string{
				t.TaskID,
				t.Task.Metadata.Name,
				fmt.Sprintf("%s/%s", t.WorkerGroup, t.WorkerID),
				t.Claimed.String(),
				formatAge(time.Time(t.Claimed)),
			},
		})
	}

	return rows, nil
}

func (r *ClaimedTasksResource) EmptyScopeResource() string {
	return "workerpools"
}

// ScopeActions returns the worker-pool sibling jump keys (minus "claimed"
// itself) for scope — see resource.ScopeActions.
func (r *ClaimedTasksResource) ScopeActions(scope string) []DetailAction {
	workerPoolID, _ := parseScope(scope)
	return workerPoolActions(workerPoolID, r.Name())
}

func (r *ClaimedTasksResource) Describe(id string) (Detail, error) {
	return describeTask(r.tc, id)
}

func (r *ClaimedTasksResource) RefreshInterval() time.Duration {
	return 15 * time.Second
}
