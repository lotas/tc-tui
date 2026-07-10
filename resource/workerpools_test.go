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
	if rows[0].Cells[0] != "gcp" || rows[0].Cells[1] != "proj/pool-a" ||
		rows[0].Cells[2] != "3" || rows[0].Cells[3] != "5" {
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
		launchConfigs: taskcluster.WorkerPoolLaunchConfigList{
			{LaunchConfigID: "lc-1", IsArchived: false},
			{LaunchConfigID: "lc-2", IsArchived: true},
		},
		errorCount: 4,
	}
	res := NewWorkerPoolsResource(fake)

	detail, err := res.Describe("proj/pool-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Title != "Worker Pool :: proj/pool-a" {
		t.Fatalf("unexpected title: %s", detail.Title)
	}
	if len(detail.Actions) != 5 {
		t.Fatalf("expected 5 actions, got %d", len(detail.Actions))
	}
	if a := detail.Actions[0]; a.Key != 'w' || a.Target.ResourceName != "workers" ||
		a.Target.ID != "proj/pool-a" || a.Target.Kind != NavScopedList {
		t.Fatalf("unexpected action[0]: %+v", a)
	}
	if a := detail.Actions[1]; a.Key != 'p' || a.Target.ResourceName != "pending" ||
		a.Target.ID != "proj/pool-a" || a.Target.Kind != NavScopedList {
		t.Fatalf("unexpected action[1]: %+v", a)
	}
	if a := detail.Actions[2]; a.Key != 'c' || a.Target.ResourceName != "claimed" ||
		a.Target.ID != "proj/pool-a" || a.Target.Kind != NavScopedList {
		t.Fatalf("unexpected action[2]: %+v", a)
	}
	if a := detail.Actions[3]; a.Key != 'l' || a.Target.ResourceName != "launchconfigs" ||
		a.Target.ID != "proj/pool-a" || a.Target.Kind != NavScopedList {
		t.Fatalf("unexpected action[3]: %+v", a)
	}
	if a := detail.Actions[4]; a.Key != 'e' || a.Target.ResourceName != "errors" ||
		a.Target.ID != "proj/pool-a" || a.Target.Kind != NavScopedList {
		t.Fatalf("unexpected action[4]: %+v", a)
	}
	if !strings.Contains(detail.Body, "a pool") || !strings.Contains(detail.Body, "owner@example.com") {
		t.Fatalf("unexpected body: %s", detail.Body)
	}
	if !strings.Contains(detail.Body, "Launch configs:[blue] 2[white] (1 archived)") {
		t.Fatalf("unexpected body: %s", detail.Body)
	}
	if !strings.Contains(detail.Body, "Errors (last 7d):[blue] 4[white]") {
		t.Fatalf("unexpected body: %s", detail.Body)
	}
}

func TestWorkerPoolsResourceDescribeOmitsSummaryLinesOnError(t *testing.T) {
	fake := &fakeTaskcluster{
		workerPool: &tcworkermanager.WorkerPoolFullDefinition{
			WorkerPoolID: "proj/pool-a",
			Description:  "a pool",
		},
		launchConfigsErr: errors.New("boom"),
		errorCountErr:    errors.New("boom"),
	}
	res := NewWorkerPoolsResource(fake)

	detail, err := res.Describe("proj/pool-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(detail.Body, "Launch configs:") || strings.Contains(detail.Body, "Errors (last 7d):") {
		t.Fatalf("expected summary lines to be omitted on error, got body: %s", detail.Body)
	}
}

func TestWorkerPoolsResourceDescribePartialSummaryFailure(t *testing.T) {
	fake := &fakeTaskcluster{
		workerPool: &tcworkermanager.WorkerPoolFullDefinition{
			WorkerPoolID: "proj/pool-a",
			Description:  "a pool",
		},
		launchConfigsErr: errors.New("boom"),
		errorCount:       7,
	}
	res := NewWorkerPoolsResource(fake)

	detail, err := res.Describe("proj/pool-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(detail.Body, "Launch configs:") {
		t.Fatalf("expected launch configs summary line to be omitted, got body: %s", detail.Body)
	}
	if !strings.Contains(detail.Body, "Errors (last 7d):[blue] 7[white]") {
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

func TestWorkerPoolsResourceFacetColumn(t *testing.T) {
	res := NewWorkerPoolsResource(&fakeTaskcluster{})

	if got := res.FacetColumn(); got != 1 {
		t.Fatalf("expected column 1 (PROVIDER), got %d", got)
	}
}

func TestWorkerPoolsResourceFacetOptionsDedupsAndSorts(t *testing.T) {
	res := NewWorkerPoolsResource(&fakeTaskcluster{})
	rows := []Row{
		{ID: "a", Cells: []string{"proj/pool-a", "gcp", "1", "1"}},
		{ID: "b", Cells: []string{"proj/pool-b", "aws", "1", "1"}},
		{ID: "c", Cells: []string{"proj/pool-c", "gcp", "1", "1"}},
	}

	got := res.FacetOptions(rows)
	want := []string{"aws", "gcp"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestWorkerPoolsResourceFacetOptionsEmptyRows(t *testing.T) {
	res := NewWorkerPoolsResource(&fakeTaskcluster{})

	got := res.FacetOptions(nil)
	if len(got) != 0 {
		t.Fatalf("expected no options, got %v", got)
	}
}
