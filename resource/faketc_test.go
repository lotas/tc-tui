package resource

import (
	"regexp"

	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcauth"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcqueue"
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

	pendingTasks    taskcluster.PendingTaskList
	pendingTasksErr error

	claimedTasks    taskcluster.ClaimedTaskList
	claimedTasksErr error
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

func (f *fakeTaskcluster) GetPendingTasks(taskQueueID string) (taskcluster.PendingTaskList, error) {
	return f.pendingTasks, f.pendingTasksErr
}

func (f *fakeTaskcluster) GetClaimedTasks(taskQueueID string) (taskcluster.ClaimedTaskList, error) {
	return f.claimedTasks, f.claimedTasksErr
}
