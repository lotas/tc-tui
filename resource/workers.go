package resource

import (
	"fmt"
	"strings"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

// workerIDSeparator is safe only because worker-manager's API schema
// disallows ':' in workerPoolId, workerGroup, and workerId
// (^[a-zA-Z0-9-_]*$ / ^[a-zA-Z0-9-_]{1,38}/[a-z]([-a-z0-9]{0,36}[a-z0-9])?$) —
// composeWorkerID/parseWorkerID's round trip depends on that external
// guarantee, not on any validation performed here.
const workerIDSeparator = "::"

type WorkersResource struct {
	tc taskcluster.Taskcluster
}

func NewWorkersResource(tc taskcluster.Taskcluster) *WorkersResource {
	return &WorkersResource{tc: tc}
}

func (r *WorkersResource) Name() string      { return "workers" }
func (r *WorkersResource) Aliases() []string { return []string{"w"} }
func (r *WorkersResource) Description() string {
	return "Individual workers within a worker pool (scoped list)"
}

func (r *WorkersResource) Columns() []Column {
	return []Column{
		{Title: "STATE", Width: 12},
		{Title: "WORKER GROUP", Width: 20},
		{Title: "WORKER ID"},
		{Title: "CAPACITY", Width: 12},
	}
}

// List is never expected to be called via normal navigation — the shell
// always either has a scope, or redirects to EmptyScopeResource() first.
func (r *WorkersResource) List() ([]Row, error) {
	return nil, fmt.Errorf("workers requires a worker pool scope")
}

func (r *WorkersResource) ScopedList(workerPoolID string) ([]Row, error) {
	workers, err := r.tc.GetWorkersForWorkerPool(workerPoolID)
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(workers))
	for _, worker := range workers {
		rows = append(rows, Row{
			ID: composeWorkerID(worker.WorkerPoolID, worker.WorkerGroup, worker.WorkerID),
			Cells: []string{
				worker.State,
				worker.WorkerGroup,
				worker.WorkerID,
				fmt.Sprintf("%d", worker.Capacity),
			},
		})
	}

	return rows, nil
}

func (r *WorkersResource) EmptyScopeResource() string {
	return "workerpools"
}

func (r *WorkersResource) Describe(id string) (Detail, error) {
	workerPoolID, workerGroup, workerID, err := parseWorkerID(id)
	if err != nil {
		return Detail{}, err
	}

	worker, err := r.tc.GetWorker(workerPoolID, workerGroup, workerID)
	if err != nil {
		return Detail{}, err
	}

	body := fmt.Sprintf(
		"[green]State:[white] %s\n\n"+
			"[green]Worker Pool:[white] %s\n"+
			"[green]Worker Group:[white] %s\n"+
			"[green]Worker ID:[white] %s\n\n"+
			"[green]Capacity:[blue] %d\n"+
			"[green]Launch Config ID:[white] %s\n\n"+
			"[green]Created:[white] %s\n"+
			"[green]Last Modified:[white] %s\n"+
			"[green]Last Checked:[white] %s\n"+
			"[green]Expires:[white] %s\n\n",
		worker.State,
		worker.WorkerPoolID,
		worker.WorkerGroup,
		worker.WorkerID,
		worker.Capacity,
		worker.LaunchConfigID,
		worker.Created,
		worker.LastModified,
		worker.LastChecked,
		worker.Expires,
	)

	return Detail{
		Title: fmt.Sprintf("Worker :: %s", worker.WorkerID),
		Body:  body,
	}, nil
}

func (r *WorkersResource) RefreshInterval() time.Duration {
	return 15 * time.Second
}

func composeWorkerID(workerPoolID, workerGroup, workerID string) string {
	return strings.Join([]string{workerPoolID, workerGroup, workerID}, workerIDSeparator)
}

func parseWorkerID(id string) (workerPoolID, workerGroup, workerID string, err error) {
	parts := strings.Split(id, workerIDSeparator)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid worker id %q", id)
	}

	return parts[0], parts[1], parts[2], nil
}
