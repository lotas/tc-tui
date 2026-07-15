package taskcluster

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	tcurls "github.com/taskcluster/taskcluster-lib-urls"
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
type ArtifactList []tcqueue.Artifact                       // ListArtifacts' members

type Taskcluster interface {
	GetVersion() Version
	GetRoot() string
	GetClientID() string

	IsAuthenticated() bool

	GetRoles() (RolesList, error)
	GetRole(roleID string) (*tcauth.GetRoleResponse, error)
	GetWorkerPools() (WorkerPoolList, error)
	GetWorkerPool(workerPoolID string) (*tcworkermanager.WorkerPoolFullDefinition, error)
	GetTaskQueueCounts(workerPoolIDs []string, onEach func(workerPoolID string, counts TaskQueueCounts))
	GetWorkerPoolErrorCounts() (map[string]int, error)
	GetWorkersForWorkerPool(workerPoolID, launchConfigID, state string) (WorkerList, error)
	GetWorkerPoolStateCounts(workerPoolID, launchConfigID string) (map[string]int, error)
	GetWorker(workerPoolID, workerGroup, workerID string) (*tcworkermanager.WorkerFullDefinition, error)
	GetWorkerRecentTasks(workerPoolID, workerGroup, workerID string) ([]tcqueue.TaskRun, error)
	GetWorkerPoolLaunchConfigs(workerPoolID string, includeArchived bool) (WorkerPoolLaunchConfigList, error)
	GetWorkerPoolErrors(workerPoolID, launchConfigID string) (WorkerPoolErrorList, error)
	GetWorkerPoolError(workerPoolID, errorID string) (*tcworkermanager.WorkerPoolError, error)
	GetWorkerPoolErrorCount(workerPoolID string) (int, error)

	GetTask(taskID string) (*tcqueue.TaskDefinitionResponse, error)
	GetTaskStatus(taskID string) (*tcqueue.TaskStatusStructure, error)
	GetTaskGroup(taskGroupID string) (*tcqueue.TaskGroupDefinitionResponse, error)
	GetTaskGroupTasks(taskGroupID string) (TaskGroupTaskList, error)
	GetDependentTasks(taskID string) (TaskGroupTaskList, error)
	GetPendingTasks(taskQueueID string) (PendingTaskList, error)
	GetClaimedTasks(taskQueueID string) (ClaimedTaskList, error)
	GetArtifacts(taskID string, runID int64) (ArtifactList, error)
	GetArtifactContent(taskID string, runID int64, name string) (content string, contentType string, truncated bool, err error)
	GetArtifactURL(taskID string, runID int64, name string) (string, error)
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

// TaskQueueCounts holds a task queue's approximate pending/claimed task
// counts. PendingKnown/ClaimedKnown are false when even GetTaskQueueCounts's
// fallback paths (see its doc comment) couldn't obtain that particular
// number — the zero value must not be mistaken for a genuine zero count.
type TaskQueueCounts struct {
	Pending      int64
	PendingKnown bool
	Claimed      int64
	ClaimedKnown bool
}

// GetTaskQueueCounts fetches pending/claimed counts for each of
// workerPoolIDs concurrently, calling onEach exactly once per ID as each
// pool's fetch completes (success or failure) — a worker pool's ID doubles
// as its task queue's ID, and there is no bulk variant of this endpoint
// (Taskcluster's own web UI fetches it the same way: one call per pool,
// batched concurrently rather than sequentially). onEach is always called,
// even when neither fallback below succeeds (both Known flags false), so a
// caller counting ticks against len(workerPoolIDs) always reaches it.
//
// TaskQueueCounts (the combined, ideal call) requires both
// queue:pending-count and queue:claimed-count scopes together, so a
// credential granted only one of the two (as observed with community-tc's
// anonymous role, which grants queue:claimed-list but apparently not
// queue:claimed-count) fails the combined call entirely. Each number then
// falls back independently to an older, more narrowly-scoped call: Pending
// to the deprecated pending-count-only endpoint, Claimed to counting
// GetClaimedTasks's result (the same queue:claimed-list-scoped call the
// existing "claimed" list view already uses successfully) — an approximate
// but perfectly serviceable substitute for a single summary column, given
// currently-claimed tasks are bounded by worker capacity rather than an
// unbounded backlog.
func (tc *TC) GetTaskQueueCounts(workerPoolIDs []string, onEach func(workerPoolID string, counts TaskQueueCounts)) {
	const maxConcurrency = 24

	var (
		wg  sync.WaitGroup
		sem = make(chan struct{}, maxConcurrency)
	)

	for _, id := range workerPoolIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }()

			if counts, err := tc.queue.TaskQueueCounts(id); err == nil {
				onEach(id, TaskQueueCounts{
					Pending: counts.PendingTasks, PendingKnown: true,
					Claimed: counts.ClaimedTasks, ClaimedKnown: true,
				})
				return
			}

			var result TaskQueueCounts
			if pending, err := tc.queue.PendingTasks(id); err == nil {
				result.Pending, result.PendingKnown = pending.PendingTasks, true
			}
			if claimed, err := tc.GetClaimedTasks(id); err == nil {
				result.Claimed, result.ClaimedKnown = int64(len(claimed)), true
			}
			onEach(id, result)
		}(id)
	}

	wg.Wait()
}

// GetWorkerPoolErrorCounts returns each worker pool's error count over the
// last 7 days in one bulk call (workerPoolErrorStats with no workerPoolId
// filter), keyed by worker pool ID.
func (tc *TC) GetWorkerPoolErrorCounts() (map[string]int, error) {
	stats, err := tc.wm.WorkerPoolErrorStats("")
	if err != nil {
		return nil, err
	}

	var byPool map[string]float64
	if err := json.Unmarshal(stats.Totals.WorkerPool, &byPool); err != nil {
		return nil, err
	}

	counts := make(map[string]int, len(byPool))
	for id, n := range byPool {
		counts[id] = int(n)
	}
	return counts, nil
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

// GetWorkerRecentTasks returns the up-to-20 most recent tasks claimed by a
// worker, via the Queue service's own worker record (distinct from
// worker-manager's — this is the only place recentTasks is exposed).
// workerPoolID is split on "/" into the provisionerId/workerType pair the
// Queue API still expects.
func (tc *TC) GetWorkerRecentTasks(workerPoolID, workerGroup, workerID string) ([]tcqueue.TaskRun, error) {
	parts := strings.SplitN(workerPoolID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid worker pool id %q", workerPoolID)
	}

	resp, err := tc.queue.GetWorker(parts[0], parts[1], workerGroup, workerID)
	if err != nil {
		return nil, err
	}

	return resp.RecentTasks, nil
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

// GetDependentTasks lists tasks that declare taskID as one of their
// dependencies — the reverse of a task's own Dependencies field.
func (tc *TC) GetDependentTasks(taskID string) (TaskGroupTaskList, error) {
	tasks, err := paginate(func(cont string) ([]tcqueue.TaskDefinitionAndStatus, string, error) {
		resp, err := tc.queue.ListDependentTasks(taskID, cont, PageSize)
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

// GetArtifacts lists the artifacts produced by one run of a task.
func (tc *TC) GetArtifacts(taskID string, runID int64) (ArtifactList, error) {
	runIDStr := strconv.FormatInt(runID, 10)

	artifacts, err := paginate(func(cont string) ([]tcqueue.Artifact, string, error) {
		resp, err := tc.queue.ListArtifacts(taskID, runIDStr, cont, PageSize)
		if err != nil {
			return nil, "", err
		}
		return resp.Artifacts, resp.ContinuationToken, nil
	})
	if err != nil {
		return nil, err
	}

	return ArtifactList(artifacts), nil
}

// MaxArtifactContentBytes caps how much of an artifact's content
// GetArtifactContent reads into memory — a build log can run to hundreds of
// MB, and reading it in full just to render a tail-truncated preview would
// both waste bandwidth and risk exhausting memory. GetArtifactContent
// reports via its truncated return value when this cap was hit.
const MaxArtifactContentBytes = 20 * 1024 * 1024 // 20 MiB

// artifactURL builds the URL to fetch or link to one artifact's content —
// signed when actual credentials are configured, matching the Taskcluster
// web UI's own getArtifactUrl (buildSignedUrlSync only when user.credentials
// is set, buildUrl otherwise). IsAuthenticated can't be used for this check:
// it tests whether currentScopes succeeds, which it does even anonymously
// (the anonymous role's own scopes), whereas signing with an empty client
// ID/access token produces a bewit missing its id/mac fields, which the
// queue rejects with "Missing bewit attributes" rather than falling back to
// anonymous access.
func (tc *TC) artifactURL(taskID string, runID int64, name string, duration time.Duration) (string, error) {
	runIDStr := strconv.FormatInt(runID, 10)

	if tc.auth.Credentials.ClientID != "" {
		signedURL, err := tc.queue.GetArtifact_SignedURL(taskID, runIDStr, name, duration)
		if err != nil {
			return "", err
		}
		return signedURL.String(), nil
	}

	route := fmt.Sprintf("task/%s/runs/%s/artifacts/%s", url.PathEscape(taskID), url.PathEscape(runIDStr), url.PathEscape(name))
	return tcurls.API(tc.tcRoot, "queue", "v1", route), nil
}

// GetArtifactContent fetches one artifact's content, capped at
// MaxArtifactContentBytes (see its doc comment) — truncated reports whether
// the cap was hit. The queue's "artifact" endpoint responds with either the
// content itself or a redirect to it depending on storage type; a plain
// http.Get follows that redirect automatically, so no response-body parsing
// is needed either way. contentType is the fetch response's own Content-Type
// header, letting callers render markdown/JSON/YAML artifacts specially
// without a second API call.
func (tc *TC) GetArtifactContent(taskID string, runID int64, name string) (content string, contentType string, truncated bool, err error) {
	fetchURL, err := tc.artifactURL(taskID, runID, name, 60*time.Second)
	if err != nil {
		return "", "", false, err
	}

	data, contentType, truncated, err := getHttpResponseCapped(fetchURL, MaxArtifactContentBytes)
	if err != nil {
		return "", "", false, err
	}

	return string(data), contentType, truncated, nil
}

// GetArtifactURL returns a URL suitable for opening or downloading one
// artifact directly (e.g. via a browser) — signed when authenticated, with a
// longer lifetime than GetArtifactContent's own fetch since there's a delay
// between generating it and the user actually loading it.
func (tc *TC) GetArtifactURL(taskID string, runID int64, name string) (string, error) {
	return tc.artifactURL(taskID, runID, name, 5*time.Minute)
}

func (tc *TC) GetRoot() string {
	return tc.tcRoot
}

// getHttpResponseCapped fetches url, reading at most maxBytes of the
// response body — truncated reports whether the body was longer than that.
// contentType is the response's own Content-Type header.
func getHttpResponseCapped(url string, maxBytes int64) (content []byte, contentType string, truncated bool, err error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, "", false, err
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, "", false, err
	}

	contentType = response.Header.Get("Content-Type")
	if int64(len(data)) > maxBytes {
		return data[:maxBytes], contentType, true, nil
	}
	return data, contentType, false, nil
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
