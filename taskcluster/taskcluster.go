package taskcluster

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcauth"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcworkermanager"
)

const PageSize = "150"

type RolesList []tcauth.GetRoleResponse
type WorkerPoolList []tcworkermanager.WorkerPoolFullDefinition

type Taskcluster interface {
	GetVersion() Version
	GetRoot() string
	GetClientID() string

	IsAuthenticated() bool

	GetRoles() (RolesList, error)
	GetRole(roleID string) (*tcauth.GetRoleResponse, error)
	GetWorkerPools() (WorkerPoolList, error)
	GetWorkerPool(workerPoolID string) (*tcworkermanager.WorkerPoolFullDefinition, error)
}

type TC struct {
	auth *tcauth.Auth
	wm   *tcworkermanager.WorkerManager

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
		auth: tcauth.NewFromEnv(),
		wm:   tcworkermanager.NewFromEnv(),
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

	return WorkerPoolList(pools), nil
}

func (tc *TC) GetWorkerPool(workerPoolID string) (*tcworkermanager.WorkerPoolFullDefinition, error) {
	return tc.wm.WorkerPool(workerPoolID)
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
