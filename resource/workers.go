package resource

import (
	"fmt"
	"strings"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

// idSeparator is safe only because worker-manager's API schema disallows ':'
// in workerPoolId, workerGroup, workerId, launchConfigId, and errorId
// (^[a-zA-Z0-9-_]*$ / ^[a-zA-Z0-9-_]{1,38}/[a-z]([-a-z0-9]{0,36}[a-z0-9])?$) —
// every compose/parse round trip in this package depends on that external
// guarantee, not on any validation performed here.
const idSeparator = "::"

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

// FacetOptions returns the states worker-manager defines, "running" first
// since it's the most commonly useful default tab.
func (r *WorkersResource) FacetOptions() []string {
	return []string{"running", "requested", "stopping", "stopped"}
}

// FacetList fetches only the workers in the given state — never an
// unfiltered/combined list, since a single pool can have tens of thousands
// of stopped workers. scope is either a bare workerPoolId (pool-wide) or a
// workerPoolId::launchConfigId compound (narrowed to one launch config, e.g.
// when reached from a Launch Config's Detail view).
func (r *WorkersResource) FacetList(scope, state string) ([]Row, error) {
	workerPoolID, launchConfigID := parseScope(scope)
	workers, err := r.tc.GetWorkersForWorkerPool(workerPoolID, launchConfigID, state)
	if err != nil {
		return nil, err
	}

	return workerRows(workers), nil
}

// FacetCounts returns worker counts by state without fetching any worker
// rows, via worker-manager's per-pool stats endpoint. See FacetList for the
// scope format.
func (r *WorkersResource) FacetCounts(scope string) (map[string]int, error) {
	workerPoolID, launchConfigID := parseScope(scope)
	return r.tc.GetWorkerPoolStateCounts(workerPoolID, launchConfigID)
}

// ScopedList exists only so WorkersResource still satisfies ScopedResource
// (needed for the EmptyScopeResource redirect when no scope is given); the
// shell always prefers FacetList via the ServerFaceted branch.
func (r *WorkersResource) ScopedList(workerPoolID string) ([]Row, error) {
	return r.FacetList(workerPoolID, r.FacetOptions()[0])
}

// workerRows converts a WorkerList into the shell's generic Row shape,
// shared by ScopedList and FacetList.
func workerRows(workers taskcluster.WorkerList) []Row {
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

	return rows
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
	return strings.Join([]string{workerPoolID, workerGroup, workerID}, idSeparator)
}

func parseWorkerID(id string) (workerPoolID, workerGroup, workerID string, err error) {
	parts := strings.Split(id, idSeparator)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid worker id %q", id)
	}

	return parts[0], parts[1], parts[2], nil
}

// composeScope joins a worker pool ID with a secondary component (a launch
// config ID, an error ID, ...) into the compound string used for both scoped
// navigation targets and list row IDs across this package.
func composeScope(workerPoolID, secondary string) string {
	return strings.Join([]string{workerPoolID, secondary}, idSeparator)
}

// parseScope splits a compound scope/ID back into its worker pool ID and
// secondary component. If no separator is present, secondary is "".
func parseScope(scope string) (workerPoolID, secondary string) {
	parts := strings.SplitN(scope, idSeparator, 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return scope, ""
}
