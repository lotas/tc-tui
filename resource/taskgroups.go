package resource

import (
	"fmt"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

// TaskGroupResource is a DirectLookup resource: Taskcluster's Queue API has
// no "list all task groups" endpoint, so a task group is only ever looked up
// directly by its own ID.
type TaskGroupResource struct {
	tc taskcluster.Taskcluster
}

func NewTaskGroupResource(tc taskcluster.Taskcluster) *TaskGroupResource {
	return &TaskGroupResource{tc: tc}
}

func (r *TaskGroupResource) Name() string      { return "taskgroup" }
func (r *TaskGroupResource) Aliases() []string { return nil }
func (r *TaskGroupResource) Description() string {
	return "A task group's metadata, looked up directly by task group ID"
}
func (r *TaskGroupResource) IDPromptLabel() string { return "task group id" }
func (r *TaskGroupResource) Columns() []Column     { return nil }

func (r *TaskGroupResource) List() ([]Row, error) {
	return nil, fmt.Errorf("taskgroup requires a task group id")
}

func (r *TaskGroupResource) Describe(id string) (Detail, error) {
	group, err := r.tc.GetTaskGroup(id)
	if err != nil {
		return Detail{}, err
	}

	sealed := "not sealed"
	if !time.Time(group.Sealed).IsZero() {
		sealed = group.Sealed.String()
	}

	body := fmt.Sprintf(
		"[green]Task Group ID:[white] %s\n\n"+
			"[green]Scheduler ID:[white] %s\n"+
			"[green]Sealed:[white] %s\n"+
			"[green]Expires:[white] %s\n",
		group.TaskGroupID,
		group.SchedulerID,
		sealed,
		group.Expires,
	)

	return Detail{
		Title: fmt.Sprintf("Task Group :: %s", group.TaskGroupID),
		Body:  body,
		Actions: []DetailAction{
			{
				Key:   't',
				Label: "tasks",
				Target: NavTarget{
					ResourceName: "tasks",
					ID:           group.TaskGroupID,
					Kind:         NavScopedList,
				},
			},
		},
	}, nil
}

func (r *TaskGroupResource) RefreshInterval() time.Duration {
	return 15 * time.Second
}
