package resource

import (
	"errors"
	"strings"
	"testing"
	"time"

	tcclient "github.com/taskcluster/taskcluster/v101/clients/client-go"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcworkermanager"

	"github.com/taskcluster/tc-tui/taskcluster"
)

func TestWorkerPoolsResourceList(t *testing.T) {
	fake := &fakeTaskcluster{
		workerPools: taskcluster.WorkerPoolList{
			{
				WorkerPoolID:      "proj/pool-a",
				ProviderID:        "gcp",
				CurrentCapacity:   3,
				RequestedCapacity: 5,
			},
		},
	}
	res := NewWorkerPoolsResource(fake)

	rows, err := res.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].ID != "proj/pool-a" {
		t.Fatalf("unexpected id: %s", rows[0].ID)
	}
	if rows[0].Cells[0] != "gcp" || rows[0].Cells[1] != "proj/pool-a" || rows[0].Cells[2] != "3 / 5" {
		t.Fatalf("unexpected cells: %+v", rows[0].Cells)
	}
}

func TestWorkerPoolsResourceListError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{workerPoolsErr: wantErr}
	res := NewWorkerPoolsResource(fake)

	_, err := res.List()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestWorkerPoolsResourceDescribe(t *testing.T) {
	fake := &fakeTaskcluster{
		workerPool: &tcworkermanager.WorkerPoolFullDefinition{
			WorkerPoolID:      "proj/pool-a",
			ProviderID:        "gcp",
			Description:       "a pool",
			Owner:             "owner@example.com",
			Created:           tcclient.Time(time.Now()),
			RequestedCapacity: 5,
			RunningCapacity:   3,
			StoppedCapacity:   2,
			RunningCount:      3,
			StoppedCount:      1,
		},
	}
	res := NewWorkerPoolsResource(fake)

	detail, err := res.Describe("proj/pool-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Title != "Worker Pool :: proj/pool-a" {
		t.Fatalf("unexpected title: %s", detail.Title)
	}
	if len(detail.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(detail.Actions))
	}
	action := detail.Actions[0]
	if action.Key != 'w' || action.Target.ResourceName != "workers" ||
		action.Target.ID != "proj/pool-a" || action.Target.Kind != NavScopedList {
		t.Fatalf("unexpected action: %+v", action)
	}
	if !strings.Contains(detail.Body, "a pool") || !strings.Contains(detail.Body, "owner@example.com") {
		t.Fatalf("unexpected body: %s", detail.Body)
	}
}

func TestWorkerPoolsResourceDescribeError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{workerPoolErr: wantErr}
	res := NewWorkerPoolsResource(fake)

	_, err := res.Describe("proj/pool-a")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
