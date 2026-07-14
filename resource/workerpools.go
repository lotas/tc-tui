package resource

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

type WorkerPoolsResource struct {
	tc taskcluster.Taskcluster
}

func NewWorkerPoolsResource(tc taskcluster.Taskcluster) *WorkerPoolsResource {
	return &WorkerPoolsResource{tc: tc}
}

func (r *WorkerPoolsResource) Name() string      { return "workerpools" }
func (r *WorkerPoolsResource) Aliases() []string { return []string{"wp", "pools"} }
func (r *WorkerPoolsResource) Description() string {
	return "Worker pool provisioning configuration — provider, capacity, launch config"
}

func (r *WorkerPoolsResource) Columns() []Column {
	return []Column{
		{Title: "WORKER POOL ID"},
		{Title: "PROVIDER", Width: 32},
		{Title: "CAPACITY", Width: 16},
		{Title: "REQUESTED", Width: 16},
		{Title: "PENDING", Width: 12},
		{Title: "CLAIMED", Width: 12},
		{Title: "ERRORS (7D)", Width: 14},
	}
}

// FacetColumn identifies the PROVIDER column (see Columns()).
func (r *WorkerPoolsResource) FacetColumn() int { return 1 }

// FacetOptions derives the distinct providers actually present in the
// already-loaded pool list — no listProviders call, since the full pool
// list (including provider) is already fetched for the table.
func (r *WorkerPoolsResource) FacetOptions(rows []Row) []string {
	seen := map[string]bool{}
	var options []string
	for _, row := range rows {
		p := row.Cells[r.FacetColumn()]
		if !seen[p] {
			seen[p] = true
			options = append(options, p)
		}
	}

	sort.Strings(options)
	return options
}

func (r *WorkerPoolsResource) List() ([]Row, error) {
	pools, err := r.tc.GetWorkerPools()
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(pools))
	for i, pool := range pools {
		ids[i] = pool.WorkerPoolID
	}
	taskQueueCounts := r.tc.GetTaskQueueCounts(ids)
	errorCounts, errorCountsErr := r.tc.GetWorkerPoolErrorCounts() // best-effort: a failed call leaves the whole column blank

	rows := make([]Row, 0, len(pools))
	for _, pool := range pools {
		var pending, claimed string
		if c, ok := taskQueueCounts[pool.WorkerPoolID]; ok {
			if c.PendingKnown {
				pending = fmt.Sprintf("%10d", c.Pending)
			}
			if c.ClaimedKnown {
				claimed = fmt.Sprintf("%10d", c.Claimed)
			}
		}
		// A pool absent from a *successful* bulk result genuinely has zero
		// errors (the endpoint only breaks down pools that have at least
		// one) — only a failed call as a whole leaves this column blank.
		var errs string
		if errorCountsErr == nil {
			errs = fmt.Sprintf("%10d", errorCounts[pool.WorkerPoolID])
		}

		rows = append(rows, Row{
			ID: pool.WorkerPoolID,
			Cells: []string{
				pool.WorkerPoolID,
				pool.ProviderID,
				fmt.Sprintf("%10d", pool.CurrentCapacity),
				fmt.Sprintf("%10d", pool.RequestedCapacity),
				pending,
				claimed,
				errs,
			},
		})
	}

	return rows, nil
}

// workerPoolActions returns the standard set of quick-jump keys to a worker
// pool's sub-resources (workers/pending/claimed/launchconfigs/errors),
// scoped pool-wide to workerPoolID. exclude omits the action whose
// ResourceName matches — typically the resource currently showing the
// hints itself, since pressing its own key to "jump" to the view already on
// screen isn't useful. If exclude doesn't match any of the 5 ResourceNames
// (e.g. a typo), it has no effect and all 5 actions are returned.
func workerPoolActions(workerPoolID, exclude string) []DetailAction {
	all := []DetailAction{
		{Key: 'w', Label: "workers", Target: NavTarget{ResourceName: "workers", ID: workerPoolID, Kind: NavScopedList}},
		{Key: 'p', Label: "pending", Target: NavTarget{ResourceName: "pending", ID: workerPoolID, Kind: NavScopedList}},
		{Key: 'c', Label: "claimed", Target: NavTarget{ResourceName: "claimed", ID: workerPoolID, Kind: NavScopedList}},
		{Key: 'l', Label: "launchconfigs", Target: NavTarget{ResourceName: "launchconfigs", ID: workerPoolID, Kind: NavScopedList}},
		{Key: 'e', Label: "errors", Target: NavTarget{ResourceName: "errors", ID: workerPoolID, Kind: NavScopedList}},
	}

	actions := make([]DetailAction, 0, len(all))
	for _, a := range all {
		if a.Target.ResourceName == exclude {
			continue
		}
		actions = append(actions, a)
	}
	return actions
}

func (r *WorkerPoolsResource) Describe(id string) (Detail, error) {
	pool, err := r.tc.GetWorkerPool(id)
	if err != nil {
		return Detail{}, err
	}

	// Best-effort summary lines: a failure in either just omits that line
	// rather than failing the whole Detail fetch, since these are
	// supplementary to the pool data rendered below.
	var summary strings.Builder
	if total, active, err := launchConfigCounts(r.tc, id); err == nil {
		summary.WriteString(fmt.Sprintf("[green]Launch configs:[blue] %d[white] (%d archived)\n", total, total-active))
	}
	if count, err := r.tc.GetWorkerPoolErrorCount(id); err == nil {
		summary.WriteString(fmt.Sprintf("[green]Errors (last 7d):[blue] %d[white]\n", count))
	}

	body := fmt.Sprintf(
		"[green]Description:[white]\n%s\n\n"+
			"[green]Created:[white] %s\n"+
			"[green]Owner:[white] %s\n\n"+
			"%s\n"+
			"%s"+
			"%s\n"+
			"[green]Config:[white]\n%s\n\n",
		renderMarkdown(pool.Description),
		pool.Created,
		pool.Owner,
		summary.String(),
		fieldRow(24, "Requested capacity", fmt.Sprint(pool.RequestedCapacity), "Running capacity", fmt.Sprint(pool.RunningCapacity), "Stopped capacity", fmt.Sprint(pool.StoppedCapacity)),
		fieldRow(24, "Running count", fmt.Sprint(pool.RunningCount), "Stopped count", fmt.Sprint(pool.StoppedCount)),
		renderYAML(pool.Config),
	)

	return Detail{
		Title:   fmt.Sprintf("Worker Pool :: %s", pool.WorkerPoolID),
		Body:    body,
		Actions: workerPoolActions(pool.WorkerPoolID, ""),
	}, nil
}

func (r *WorkerPoolsResource) RefreshInterval() time.Duration {
	return 15 * time.Second
}

func (r *WorkerPoolsResource) ListWebURL(rootURL, scope string) string {
	return webUIPath(rootURL, "worker-manager")
}

func (r *WorkerPoolsResource) DetailWebURL(rootURL, id string) string {
	return webUIPath(rootURL, "worker-manager/"+pathSegment(id))
}
