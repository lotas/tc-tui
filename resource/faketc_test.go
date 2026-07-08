package resource

import (
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcauth"
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
