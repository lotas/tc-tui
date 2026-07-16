package resource

import (
	"regexp"

	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcauth"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcindex"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcqueue"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcsecrets"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcworkermanager"

	"github.com/taskcluster/tc-tui/taskcluster"
)

// regionTagRe matches tview's `[tag]` region/color markup (e.g. "[green]",
// "[#dddddd:]", "[-:-:-]").
var regionTagRe = regexp.MustCompile(`\[[^][]*\]`)

// stripRegionTags removes tview region/color tags from s. Markdown-rendered
// body text (see renderMarkdown/renderGlamour in render.go) can interleave
// style tags mid-phrase — e.g. glamour periodically re-asserts the same
// color with no visible effect — which breaks naive multi-word
// strings.Contains checks against the raw body. Tests that need to assert on
// rendered prose should strip tags first rather than relying on markup-free
// contiguous substrings.
func stripRegionTags(s string) string {
	return regionTagRe.ReplaceAllString(s, "")
}

type fakeTaskcluster struct {
	roles    taskcluster.RolesList
	rolesErr error
	role     *tcauth.GetRoleResponse
	roleErr  error

	workerPools    taskcluster.WorkerPoolList
	workerPoolsErr error
	workerPool     *tcworkermanager.WorkerPoolFullDefinition
	workerPoolErr  error

	taskQueueCounts map[string]taskcluster.TaskQueueCounts

	workerPoolErrorCounts    map[string]int
	workerPoolErrorCountsErr error

	workers               taskcluster.WorkerList
	workersErr            error
	workersState          string // last `state` param GetWorkersForWorkerPool was called with
	workersLaunchConfigID string // last `launchConfigId` param GetWorkersForWorkerPool was called with

	stateCounts               map[string]int
	stateCountsErr            error
	stateCountsLaunchConfigID string // last `launchConfigId` param GetWorkerPoolStateCounts was called with

	worker    *tcworkermanager.WorkerFullDefinition
	workerErr error

	workerRecentTasks    []tcqueue.TaskRun
	workerRecentTasksErr error

	launchConfigs                taskcluster.WorkerPoolLaunchConfigList
	launchConfigsErr             error
	launchConfigsIncludeArchived bool // last `includeArchived` param GetWorkerPoolLaunchConfigs was called with

	workerPoolErrors               taskcluster.WorkerPoolErrorList
	workerPoolErrorsErr            error
	workerPoolErrorsLaunchConfigID string // last `launchConfigId` param GetWorkerPoolErrors was called with

	workerPoolError    *tcworkermanager.WorkerPoolError
	workerPoolErrorErr error

	errorCount    int
	errorCountErr error

	task    *tcqueue.TaskDefinitionResponse
	taskErr error

	taskStatus    *tcqueue.TaskStatusStructure
	taskStatusErr error

	taskGroup    *tcqueue.TaskGroupDefinitionResponse
	taskGroupErr error

	taskGroupTasks    taskcluster.TaskGroupTaskList
	taskGroupTasksErr error

	dependentTasks    taskcluster.TaskGroupTaskList
	dependentTasksErr error

	pendingTasks    taskcluster.PendingTaskList
	pendingTasksErr error

	claimedTasks    taskcluster.ClaimedTaskList
	claimedTasksErr error

	artifacts    taskcluster.ArtifactList
	artifactsErr error

	artifactContent     string
	artifactContentType string
	artifactTruncated   bool
	artifactContentErr  error

	artifactURL    string
	artifactURLErr error

	clients    taskcluster.ClientList
	clientsErr error
	client     *tcauth.GetClientResponse
	clientErr  error

	secrets    []string
	secretsErr error
	secret     *tcsecrets.Secret
	secretErr  error

	purgeCacheRequestsForPool    taskcluster.PurgeCacheRequestList
	purgeCacheRequestsForPoolErr error

	indexNamespaces    taskcluster.IndexNamespaceList
	indexNamespacesErr error
	indexTasks         taskcluster.IndexTaskList
	indexTasksErr      error
	findIndexedTask    *tcindex.IndexedTaskResponse
	findIndexedTaskErr error
}

func (f *fakeTaskcluster) GetVersion() taskcluster.Version { return taskcluster.Version{} }
func (f *fakeTaskcluster) GetRoot() string                 { return "" }
func (f *fakeTaskcluster) GetClientID() string             { return "" }
func (f *fakeTaskcluster) IsAuthenticated() bool           { return false }

func (f *fakeTaskcluster) GetRoles() (taskcluster.RolesList, error) {
	return f.roles, f.rolesErr
}

func (f *fakeTaskcluster) GetRole(roleID string) (*tcauth.GetRoleResponse, error) {
	return f.role, f.roleErr
}

func (f *fakeTaskcluster) GetWorkerPools() (taskcluster.WorkerPoolList, error) {
	return f.workerPools, f.workerPoolsErr
}

func (f *fakeTaskcluster) GetWorkerPool(workerPoolID string) (*tcworkermanager.WorkerPoolFullDefinition, error) {
	return f.workerPool, f.workerPoolErr
}

// GetTaskQueueCounts calls onEach once per ID, in order, with whatever
// f.taskQueueCounts holds for that ID (the zero value if absent or if
// wanted rejects it) — no goroutines, so tests calling it don't need to
// synchronize.
func (f *fakeTaskcluster) GetTaskQueueCounts(workerPoolIDs []string, wanted func(workerPoolID string) bool, onEach func(workerPoolID string, counts taskcluster.TaskQueueCounts)) {
	for _, id := range workerPoolIDs {
		if !wanted(id) {
			onEach(id, taskcluster.TaskQueueCounts{})
			continue
		}
		onEach(id, f.taskQueueCounts[id])
	}
}

func (f *fakeTaskcluster) GetWorkerPoolErrorCounts() (map[string]int, error) {
	return f.workerPoolErrorCounts, f.workerPoolErrorCountsErr
}

func (f *fakeTaskcluster) GetWorkersForWorkerPool(workerPoolID, launchConfigID, state string) (taskcluster.WorkerList, error) {
	f.workersState = state
	f.workersLaunchConfigID = launchConfigID
	return f.workers, f.workersErr
}

func (f *fakeTaskcluster) GetWorkerPoolStateCounts(workerPoolID, launchConfigID string) (map[string]int, error) {
	f.stateCountsLaunchConfigID = launchConfigID
	return f.stateCounts, f.stateCountsErr
}

func (f *fakeTaskcluster) GetWorker(workerPoolID, workerGroup, workerID string) (*tcworkermanager.WorkerFullDefinition, error) {
	return f.worker, f.workerErr
}

func (f *fakeTaskcluster) GetWorkerRecentTasks(workerPoolID, workerGroup, workerID string) ([]tcqueue.TaskRun, error) {
	return f.workerRecentTasks, f.workerRecentTasksErr
}

func (f *fakeTaskcluster) GetWorkerPoolLaunchConfigs(workerPoolID string, includeArchived bool) (taskcluster.WorkerPoolLaunchConfigList, error) {
	f.launchConfigsIncludeArchived = includeArchived
	return f.launchConfigs, f.launchConfigsErr
}

func (f *fakeTaskcluster) GetWorkerPoolErrors(workerPoolID, launchConfigID string) (taskcluster.WorkerPoolErrorList, error) {
	f.workerPoolErrorsLaunchConfigID = launchConfigID
	return f.workerPoolErrors, f.workerPoolErrorsErr
}

func (f *fakeTaskcluster) GetWorkerPoolError(workerPoolID, errorID string) (*tcworkermanager.WorkerPoolError, error) {
	return f.workerPoolError, f.workerPoolErrorErr
}

func (f *fakeTaskcluster) GetWorkerPoolErrorCount(workerPoolID string) (int, error) {
	return f.errorCount, f.errorCountErr
}

func (f *fakeTaskcluster) GetTask(taskID string) (*tcqueue.TaskDefinitionResponse, error) {
	return f.task, f.taskErr
}

func (f *fakeTaskcluster) GetTaskStatus(taskID string) (*tcqueue.TaskStatusStructure, error) {
	return f.taskStatus, f.taskStatusErr
}

func (f *fakeTaskcluster) GetTaskGroup(taskGroupID string) (*tcqueue.TaskGroupDefinitionResponse, error) {
	return f.taskGroup, f.taskGroupErr
}

func (f *fakeTaskcluster) GetTaskGroupTasks(taskGroupID string) (taskcluster.TaskGroupTaskList, error) {
	return f.taskGroupTasks, f.taskGroupTasksErr
}

func (f *fakeTaskcluster) GetDependentTasks(taskID string) (taskcluster.TaskGroupTaskList, error) {
	return f.dependentTasks, f.dependentTasksErr
}

func (f *fakeTaskcluster) GetPendingTasks(taskQueueID string) (taskcluster.PendingTaskList, error) {
	return f.pendingTasks, f.pendingTasksErr
}

func (f *fakeTaskcluster) GetClaimedTasks(taskQueueID string) (taskcluster.ClaimedTaskList, error) {
	return f.claimedTasks, f.claimedTasksErr
}

func (f *fakeTaskcluster) GetArtifacts(taskID string, runID int64) (taskcluster.ArtifactList, error) {
	return f.artifacts, f.artifactsErr
}

func (f *fakeTaskcluster) GetArtifactContent(taskID string, runID int64, name string) (string, string, bool, error) {
	return f.artifactContent, f.artifactContentType, f.artifactTruncated, f.artifactContentErr
}

func (f *fakeTaskcluster) GetArtifactURL(taskID string, runID int64, name string) (string, error) {
	return f.artifactURL, f.artifactURLErr
}

func (f *fakeTaskcluster) GetClients() (taskcluster.ClientList, error) {
	return f.clients, f.clientsErr
}

func (f *fakeTaskcluster) GetClient(clientID string) (*tcauth.GetClientResponse, error) {
	return f.client, f.clientErr
}

func (f *fakeTaskcluster) GetSecrets() ([]string, error) {
	return f.secrets, f.secretsErr
}

func (f *fakeTaskcluster) GetSecret(name string) (*tcsecrets.Secret, error) {
	return f.secret, f.secretErr
}

func (f *fakeTaskcluster) GetPurgeCacheRequestsForPool(workerPoolID string) (taskcluster.PurgeCacheRequestList, error) {
	return f.purgeCacheRequestsForPool, f.purgeCacheRequestsForPoolErr
}

func (f *fakeTaskcluster) GetIndexNamespaces(namespace string) (taskcluster.IndexNamespaceList, error) {
	return f.indexNamespaces, f.indexNamespacesErr
}

func (f *fakeTaskcluster) GetIndexTasks(namespace string) (taskcluster.IndexTaskList, error) {
	return f.indexTasks, f.indexTasksErr
}

func (f *fakeTaskcluster) FindIndexedTask(indexPath string) (*tcindex.IndexedTaskResponse, error) {
	return f.findIndexedTask, f.findIndexedTaskErr
}
