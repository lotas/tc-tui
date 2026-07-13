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
func (r *TaskGroupResource) Aliases() []string { return []string{"g"} }
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

// ListWebURL is never expected to be called — TaskGroupResource is a
// DirectLookup and never renders a List view.
func (r *TaskGroupResource) ListWebURL(rootURL, scope string) string { return "" }

func (r *TaskGroupResource) DetailWebURL(rootURL, id string) string {
	return taskGroupWebURL(rootURL, id)
}

// taskGroupWebURL links to a task group's own page — the web UI's task
// group page also lists that group's tasks, so this doubles as the link for
// TasksResource's List view. Shared by TaskGroupResource and TasksResource.
func taskGroupWebURL(rootURL, taskGroupID string) string {
	return webUIPath(rootURL, "tasks/groups/"+pathSegment(taskGroupID))
}
