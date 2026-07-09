package resource

import (
	"errors"
	"testing"

	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcqueue"

	"github.com/taskcluster/tc-tui/taskcluster"
)

func TestPendingTasksResourceScopedList(t *testing.T) {
	fake := &fakeTaskcluster{
		pendingTasks: taskcluster.PendingTaskList{
			{
				TaskID: "task-1",
				Task:   tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}, WorkerType: "linux-b-large"},
			},
		},
	}
	res := NewPendingTasksResource(fake)

	rows, err := res.ScopedList("gcp/pool-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].ID != "task-1" || rows[0].Cells[0] != "task-1" ||
		rows[0].Cells[1] != "build" || rows[0].Cells[2] != "linux-b-large" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
}

func TestPendingTasksResourceScopedListError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{pendingTasksErr: wantErr}
	res := NewPendingTasksResource(fake)

	_, err := res.ScopedList("gcp/pool-a")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestPendingTasksResourceListReturnsError(t *testing.T) {
	res := NewPendingTasksResource(&fakeTaskcluster{})

	if _, err := res.List(); err == nil {
		t.Fatalf("expected an error, got nil")
	}
}

func TestPendingTasksResourceEmptyScopeResource(t *testing.T) {
	res := NewPendingTasksResource(&fakeTaskcluster{})

	if got := res.EmptyScopeResource(); got != "workerpools" {
		t.Fatalf("expected %q, got %q", "workerpools", got)
	}
}

func TestPendingTasksResourceDescribeDelegatesToDescribeTask(t *testing.T) {
	fake := &fakeTaskcluster{
		task:       &tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}},
		taskStatus: &tcqueue.TaskStatusStructure{State: "pending"},
	}
	res := NewPendingTasksResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Title != "Task :: build (task-1)" {
		t.Fatalf("unexpected title: %s", detail.Title)
	}
}

func TestClaimedTasksResourceScopedList(t *testing.T) {
	fake := &fakeTaskcluster{
		claimedTasks: taskcluster.ClaimedTaskList{
			{
				TaskID:      "task-1",
				Task:        tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}},
				WorkerGroup: "us-west1",
				WorkerID:    "i-1234",
			},
		},
	}
	res := NewClaimedTasksResource(fake)

	rows, err := res.ScopedList("gcp/pool-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].ID != "task-1" || rows[0].Cells[0] != "task-1" ||
		rows[0].Cells[1] != "build" || rows[0].Cells[2] != "us-west1/i-1234" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
}

func TestClaimedTasksResourceScopedListError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{claimedTasksErr: wantErr}
	res := NewClaimedTasksResource(fake)

	_, err := res.ScopedList("gcp/pool-a")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestClaimedTasksResourceListReturnsError(t *testing.T) {
	res := NewClaimedTasksResource(&fakeTaskcluster{})

	if _, err := res.List(); err == nil {
		t.Fatalf("expected an error, got nil")
	}
}

func TestClaimedTasksResourceEmptyScopeResource(t *testing.T) {
	res := NewClaimedTasksResource(&fakeTaskcluster{})

	if got := res.EmptyScopeResource(); got != "workerpools" {
		t.Fatalf("expected %q, got %q", "workerpools", got)
	}
}

func TestClaimedTasksResourceDescribeDelegatesToDescribeTask(t *testing.T) {
	fake := &fakeTaskcluster{
		task:       &tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}},
		taskStatus: &tcqueue.TaskStatusStructure{State: "running"},
	}
	res := NewClaimedTasksResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Title != "Task :: build (task-1)" {
		t.Fatalf("unexpected title: %s", detail.Title)
	}
}
