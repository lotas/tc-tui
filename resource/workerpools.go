package resource

import (
	"fmt"
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
		{Title: "PROVIDER", Width: 20},
		{Title: "WORKER POOL ID"},
		{Title: "CAPACITY", Width: 12},
	}
}

func (r *WorkerPoolsResource) List() ([]Row, error) {
	pools, err := r.tc.GetWorkerPools()
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(pools))
	for _, pool := range pools {
		rows = append(rows, Row{
			ID: pool.WorkerPoolID,
			Cells: []string{
				pool.ProviderID,
				pool.WorkerPoolID,
				fmt.Sprintf("%d / %d", pool.CurrentCapacity, pool.RequestedCapacity),
			},
		})
	}

	return rows, nil
}

func (r *WorkerPoolsResource) Describe(id string) (Detail, error) {
	pool, err := r.tc.GetWorkerPool(id)
	if err != nil {
		return Detail{}, err
	}

	body := fmt.Sprintf(
		"[green]Description:[white] %s\n\n"+
			"[green]Created:[white] %s\n"+
			"[green]Owner:[white] %s\n\n"+
			"[green]Requested capacity:[blue] %d\n"+
			"[green]Running capacity:[blue] %d\n"+
			"[green]Stopped capacity:[blue] %d\n"+
			"[green]Running count:[blue] %d\n"+
			"[green]Stopped count:[blue] %d\n\n"+
			"[green]Config:[white] %s\n\n",
		pool.Description,
		pool.Created,
		pool.Owner,
		pool.RequestedCapacity,
		pool.RunningCapacity,
		pool.StoppedCapacity,
		pool.RunningCount,
		pool.StoppedCount,
		pool.Config,
	)

	return Detail{
		Title: fmt.Sprintf("Worker Pool :: %s", pool.WorkerPoolID),
		Body:  body,
		Actions: []DetailAction{
			{
				Key:   'w',
				Label: "workers",
				Target: NavTarget{
					ResourceName: "workers",
					ID:           pool.WorkerPoolID,
					Kind:         NavScopedList,
				},
			},
		},
	}, nil
}

func (r *WorkerPoolsResource) RefreshInterval() time.Duration {
	return 15 * time.Second
}
