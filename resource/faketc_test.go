package resource

import (
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcauth"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcqueue"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcworkermanager"

	"github.com/taskcluster/tc-tui/taskcluster"
)

type fakeTaskcluster struct {
	roles    taskcluster.RolesList
	rolesErr error
	role     *tcauth.GetRoleResponse
	roleErr  error

	workerPools    taskcluster.WorkerPoolList
	workerPoolsErr error
	workerPool     *tcworkermanager.WorkerPoolFullDefinition
	workerPoolErr  error

	workers      taskcluster.WorkerList
	workersErr   error
	workersState string // last `state` param GetWorkersForWorkerPool was called with

	stateCounts    map[string]int
	stateCountsErr error

	worker    *tcworkermanager.WorkerFullDefinition
	workerErr error

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

func (f *fakeTaskcluster) GetWorkersForWorkerPool(workerPoolID, state string) (taskcluster.WorkerList, error) {
	f.workersState = state
	return f.workers, f.workersErr
}

func (f *fakeTaskcluster) GetWorkerPoolStateCounts(workerPoolID string) (map[string]int, error) {
	return f.stateCounts, f.stateCountsErr
}

func (f *fakeTaskcluster) GetWorker(workerPoolID, workerGroup, workerID string) (*tcworkermanager.WorkerFullDefinition, error) {
	return f.worker, f.workerErr
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
