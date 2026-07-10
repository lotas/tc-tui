package taskcluster

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcauth"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcqueue"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcworkermanager"
)

const PageSize = "150"

type RolesList []tcauth.GetRoleResponse
type WorkerPoolList []tcworkermanager.WorkerPoolFullDefinition
type WorkerList []tcworkermanager.WorkerFullDefinition
type TaskGroupTaskList []tcqueue.TaskDefinitionAndStatus   // ListTaskGroup's members
type PendingTaskList []tcqueue.Var3                        // ListPendingTasks' members
type ClaimedTaskList []tcqueue.Var4                        // ListClaimedTasks' members
type WorkerPoolLaunchConfigList []tcworkermanager.Var1     // ListWorkerPoolLaunchConfigs' members
type WorkerPoolErrorList []tcworkermanager.WorkerPoolError // ListWorkerPoolErrors' members

type Taskcluster interface {
	GetVersion() Version
	GetRoot() string
	GetClientID() string

	IsAuthenticated() bool

	GetRoles() (RolesList, error)
	GetRole(roleID string) (*tcauth.GetRoleResponse, error)
	GetWorkerPools() (WorkerPoolList, error)
	GetWorkerPool(workerPoolID string) (*tcworkermanager.WorkerPoolFullDefinition, error)
	GetWorkersForWorkerPool(workerPoolID, launchConfigID, state string) (WorkerList, error)
	GetWorkerPoolStateCounts(workerPoolID, launchConfigID string) (map[string]int, error)
	GetWorker(workerPoolID, workerGroup, workerID string) (*tcworkermanager.WorkerFullDefinition, error)
	GetWorkerPoolLaunchConfigs(workerPoolID string, includeArchived bool) (WorkerPoolLaunchConfigList, error)
	GetWorkerPoolErrors(workerPoolID, launchConfigID string) (WorkerPoolErrorList, error)
	GetWorkerPoolError(workerPoolID, errorID string) (*tcworkermanager.WorkerPoolError, error)
	GetWorkerPoolErrorCount(workerPoolID string) (int, error)

	GetTask(taskID string) (*tcqueue.TaskDefinitionResponse, error)
	GetTaskStatus(taskID string) (*tcqueue.TaskStatusStructure, error)
	GetTaskGroup(taskGroupID string) (*tcqueue.TaskGroupDefinitionResponse, error)
	GetTaskGroupTasks(taskGroupID string) (TaskGroupTaskList, error)
	GetPendingTasks(taskQueueID string) (PendingTaskList, error)
	GetClaimedTasks(taskQueueID string) (ClaimedTaskList, error)
}

type TC struct {
	auth  *tcauth.Auth
	wm    *tcworkermanager.WorkerManager
	queue *tcqueue.Queue

	tcRoot string
}

type Version struct {
	Source  string `json:"source"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Build   string `json:"build"`
}

func NewTaskcluster() Taskcluster {
	tc := &TC{
		auth:  tcauth.NewFromEnv(),
		wm:    tcworkermanager.NewFromEnv(),
		queue: tcqueue.NewFromEnv(),
	}

	tc.tcRoot = tc.auth.RootURL
	if tc.tcRoot == "" {
		panic("Root URL not defined. export TASKCLUSTER_ROOT_URL=x")
	}

	return tc
}

func (tc *TC) GetClientID() string {
	if tc.auth.Credentials.ClientID != "" {
		return tc.auth.Credentials.ClientID
	}

	return "(anonymous)"
}

func (tc *TC) IsAuthenticated() bool {
	_, err := tc.auth.CurrentScopes()
	return err == nil
}

func (tc *TC) GetVersion() Version {
	versionJson, err := getHttpResponse(tc.tcRoot + "/__version__")
	if err != nil {
		panic(err)
	}
	ver := Version{}
	if err := json.Unmarshal([]byte(versionJson), &ver); err != nil {
		panic(err)
	}

	return ver
}

func (tc *TC) GetRoles() (RolesList, error) {
	roles, err := paginate(func(cont string) ([]tcauth.GetRoleResponse, string, error) {
		resp, err := tc.auth.ListRoles2(cont, PageSize)
		if err != nil {
			return nil, "", err
		}
		return resp.Roles, resp.ContinuationToken, nil
	})
	if err != nil {
		return nil, err
	}

	return RolesList(roles), nil
}

func (tc *TC) GetRole(roleID string) (*tcauth.GetRoleResponse, error) {
	return tc.auth.Role(roleID)
}

func (tc *TC) GetWorkerPools() (WorkerPoolList, error) {
	pools, err := paginate(func(cont string) ([]tcworkermanager.WorkerPoolFullDefinition, string, error) {
		resp, err := tc.wm.ListWorkerPools(cont, PageSize)
		if err != nil {
			return nil, "", err
		}
		return resp.WorkerPools, resp.ContinuationToken, nil
	})
	if err != nil {
		return nil, err
	}

	stats, err := paginate(func(cont string) ([]tcworkermanager.Var3, string, error) {
		resp, err := tc.wm.ListWorkerPoolsStats(cont, PageSize)
		if err != nil {
			return nil, "", err
		}
		return resp.WorkerPoolsStats, resp.ContinuationToken, nil
	})
	if err != nil {
		return nil, err
	}

	statsByID := make(map[string]tcworkermanager.Var3, len(stats))
	for _, s := range stats {
		statsByID[s.WorkerPoolID] = s
	}

	for i, pool := range pools {
		s, ok := statsByID[pool.WorkerPoolID]
		if !ok {
			continue
		}
		pools[i].CurrentCapacity = s.CurrentCapacity
		pools[i].RequestedCapacity = s.RequestedCapacity
		pools[i].RequestedCount = s.RequestedCount
		pools[i].RunningCapacity = s.RunningCapacity
		pools[i].RunningCount = s.RunningCount
		pools[i].StoppedCapacity = s.StoppedCapacity
		pools[i].StoppedCount = s.StoppedCount
		pools[i].StoppingCapacity = s.StoppingCapacity
		pools[i].StoppingCount = s.StoppingCount
	}

	return WorkerPoolList(pools), nil
}

func (tc *TC) GetWorkerPool(workerPoolID string) (*tcworkermanager.WorkerPoolFullDefinition, error) {
	return tc.wm.WorkerPool(workerPoolID)
}

func (tc *TC) GetWorkersForWorkerPool(workerPoolID, launchConfigID, state string) (WorkerList, error) {
	workers, err := paginate(func(cont string) ([]tcworkermanager.WorkerFullDefinition, string, error) {
		resp, err := tc.wm.ListWorkersForWorkerPool(workerPoolID, cont, launchConfigID, PageSize, state)
		if err != nil {
			return nil, "", err
		}
		return resp.Workers, resp.ContinuationToken, nil
	})
	if err != nil {
		return nil, err
	}

	return WorkerList(workers), nil
}

// GetWorkerPoolStateCounts returns worker counts by state for one pool. With
// launchConfigID empty, counts are summed across every launch configuration;
// otherwise only the matching launch configuration's counts are returned. It
// calls the lightweight worker-pool stats endpoint — no individual worker
// rows are fetched either way.
func (tc *TC) GetWorkerPoolStateCounts(workerPoolID, launchConfigID string) (map[string]int, error) {
	stats, err := tc.wm.WorkerPoolStats(workerPoolID)
	if err != nil {
		return nil, err
	}

	counts := map[string]int{"requested": 0, "running": 0, "stopping": 0, "stopped": 0}
	for _, lc := range stats.LaunchConfigStats {
		if launchConfigID != "" && lc.LaunchConfigID != launchConfigID {
			continue
		}
		counts["requested"] += int(lc.RequestedCount)
		counts["running"] += int(lc.RunningCount)
		counts["stopping"] += int(lc.StoppingCount)
		counts["stopped"] += int(lc.StoppedCount)
	}

	return counts, nil
}

func (tc *TC) GetWorker(workerPoolID, workerGroup, workerID string) (*tcworkermanager.WorkerFullDefinition, error) {
	return tc.wm.Worker(workerPoolID, workerGroup, workerID)
}

// GetWorkerPoolLaunchConfigs lists a worker pool's launch configurations.
// With includeArchived false, only active (non-archived) configs are
// returned — matches the API's own default.
func (tc *TC) GetWorkerPoolLaunchConfigs(workerPoolID string, includeArchived bool) (WorkerPoolLaunchConfigList, error) {
	archived := "false"
	if includeArchived {
		archived = "true"
	}

	configs, err := paginate(func(cont string) ([]tcworkermanager.Var1, string, error) {
		resp, err := tc.wm.ListWorkerPoolLaunchConfigs(workerPoolID, cont, archived, PageSize)
		if err != nil {
			return nil, "", err
		}
		return resp.WorkerPoolLaunchConfigs, resp.ContinuationToken, nil
	})
	if err != nil {
		return nil, err
	}

	return WorkerPoolLaunchConfigList(configs), nil
}

// GetWorkerPoolErrors lists provisioning errors reported for a worker pool.
// With launchConfigID empty, every launch configuration's errors are
// returned; otherwise only errors reported against that launch configuration.
func (tc *TC) GetWorkerPoolErrors(workerPoolID, launchConfigID string) (WorkerPoolErrorList, error) {
	errs, err := paginate(func(cont string) ([]tcworkermanager.WorkerPoolError, string, error) {
		resp, err := tc.wm.ListWorkerPoolErrors(workerPoolID, cont, "" /* errorId */, launchConfigID, PageSize)
		if err != nil {
			return nil, "", err
		}
		return resp.WorkerPoolErrors, resp.ContinuationToken, nil
	})
	if err != nil {
		return nil, err
	}

	return WorkerPoolErrorList(errs), nil
}

// GetWorkerPoolError fetches a single worker pool error by ID, using the
// API's own errorId filter rather than listing and searching client-side.
func (tc *TC) GetWorkerPoolError(workerPoolID, errorID string) (*tcworkermanager.WorkerPoolError, error) {
	resp, err := tc.wm.ListWorkerPoolErrors(workerPoolID, "", errorID, "", "1")
	if err != nil {
		return nil, err
	}
	if len(resp.WorkerPoolErrors) == 0 {
		return nil, fmt.Errorf("worker pool error %q not found in worker pool %q", errorID, workerPoolID)
	}

	return &resp.WorkerPoolErrors[0], nil
}

// GetWorkerPoolErrorCount returns the total number of provisioning errors
// reported for a worker pool over worker-manager's fixed lookback window
// (roughly the last 7 days), via the dedicated stats endpoint — no error
// rows are fetched.
func (tc *TC) GetWorkerPoolErrorCount(workerPoolID string) (int, error) {
	stats, err := tc.wm.WorkerPoolErrorStats(workerPoolID)
	if err != nil {
		return 0, err
	}

	return int(stats.Totals.Total), nil
}

func (tc *TC) GetTask(taskID string) (*tcqueue.TaskDefinitionResponse, error) {
	return tc.queue.Task(taskID)
}

func (tc *TC) GetTaskStatus(taskID string) (*tcqueue.TaskStatusStructure, error) {
	resp, err := tc.queue.Status(taskID)
	if err != nil {
		return nil, err
	}
	return &resp.Status, nil
}

func (tc *TC) GetTaskGroup(taskGroupID string) (*tcqueue.TaskGroupDefinitionResponse, error) {
	return tc.queue.GetTaskGroup(taskGroupID)
}

func (tc *TC) GetTaskGroupTasks(taskGroupID string) (TaskGroupTaskList, error) {
	tasks, err := paginate(func(cont string) ([]tcqueue.TaskDefinitionAndStatus, string, error) {
		resp, err := tc.queue.ListTaskGroup(taskGroupID, cont, PageSize)
		if err != nil {
			return nil, "", err
		}
		return resp.Tasks, resp.ContinuationToken, nil
	})
	if err != nil {
		return nil, err
	}

	return TaskGroupTaskList(tasks), nil
}

func (tc *TC) GetPendingTasks(taskQueueID string) (PendingTaskList, error) {
	tasks, err := paginate(func(cont string) ([]tcqueue.Var3, string, error) {
		resp, err := tc.queue.ListPendingTasks(taskQueueID, cont, PageSize)
		if err != nil {
			return nil, "", err
		}
		return resp.Tasks, resp.ContinuationToken, nil
	})
	if err != nil {
		return nil, err
	}

	return PendingTaskList(tasks), nil
}

func (tc *TC) GetClaimedTasks(taskQueueID string) (ClaimedTaskList, error) {
	tasks, err := paginate(func(cont string) ([]tcqueue.Var4, string, error) {
		resp, err := tc.queue.ListClaimedTasks(taskQueueID, cont, PageSize)
		if err != nil {
			return nil, "", err
		}
		return resp.Tasks, resp.ContinuationToken, nil
	})
	if err != nil {
		return nil, err
	}

	return ClaimedTaskList(tasks), nil
}

func (tc *TC) GetRoot() string {
	return tc.tcRoot
}

func getHttpResponse(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(contents), nil
}
