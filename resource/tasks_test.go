package resource

import (
	"errors"
	"strings"
	"testing"
	"time"

	tcclient "github.com/taskcluster/taskcluster/v101/clients/client-go"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcqueue"

	"github.com/taskcluster/tc-tui/taskcluster"
)

func TestTaskResourceDescribe(t *testing.T) {
	fake := &fakeTaskcluster{
		task: &tcqueue.TaskDefinitionResponse{
			Metadata: tcqueue.TaskMetadata{
				Name:        "build-linux",
				Description: "builds linux",
				Owner:       "owner@example.com",
				Source:      "https://example.com/source",
			},
			ProvisionerID: "gcp",
			WorkerType:    "linux-b-large",
			Priority:      "high",
			TaskGroupID:   "grp-1",
			Dependencies:  []string{"dep-1"},
			Scopes:        []string{"queue:get-task:*"},
			Created:       tcclient.Time(time.Now()),
			Deadline:      tcclient.Time(time.Now()),
			Expires:       tcclient.Time(time.Now()),
		},
		taskStatus: &tcqueue.TaskStatusStructure{
			State:       "completed",
			RetriesLeft: 3,
			Runs: []tcqueue.RunInformation{
				{RunID: 0, State: "completed", ReasonResolved: "completed", WorkerGroup: "us-west1", WorkerID: "i-1"},
			},
		},
	}
	res := NewTaskResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Title != "Task :: build-linux (task-1)" {
		t.Fatalf("unexpected title: %s", detail.Title)
	}
	if !strings.Contains(detail.Body, "build-linux") || !strings.Contains(detail.Body, "completed") {
		t.Fatalf("unexpected body: %s", detail.Body)
	}
	if len(detail.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(detail.Actions))
	}
	groupAction := detail.Actions[0]
	if groupAction.Key != 'g' || groupAction.Target.ResourceName != "taskgroup" ||
		groupAction.Target.ID != "grp-1" || groupAction.Target.Kind != NavDetail {
		t.Fatalf("unexpected action: %+v", groupAction)
	}
	depsAction := detail.Actions[1]
	if depsAction.Key != 'd' || depsAction.Target.ResourceName != "taskdeps" ||
		depsAction.Target.ID != "task-1" || depsAction.Target.Kind != NavScopedList {
		t.Fatalf("unexpected action: %+v", depsAction)
	}
}

func TestTaskResourceDescribeIncludesPayload(t *testing.T) {
	fake := &fakeTaskcluster{
		task: &tcqueue.TaskDefinitionResponse{
			Metadata: tcqueue.TaskMetadata{Name: "build", Description: "builds the thing"},
			Payload:  []byte(`{"command":["echo","hi"]}`),
		},
		taskStatus: &tcqueue.TaskStatusStructure{State: "completed"},
	}
	res := NewTaskResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(detail.Body, "Payload:") || !strings.Contains(detail.Body, "command") {
		t.Fatalf("expected the rendered payload in the body, got: %s", detail.Body)
	}
	if !strings.Contains(stripRegionTags(detail.Body), "builds the thing") {
		t.Fatalf("expected the rendered description in the body, got: %s", detail.Body)
	}
}

func TestTaskResourceDescribeTaskError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{taskErr: wantErr}
	res := NewTaskResource(fake)

	_, err := res.Describe("task-1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestTaskResourceDescribeStatusError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{
		task:          &tcqueue.TaskDefinitionResponse{},
		taskStatusErr: wantErr,
	}
	res := NewTaskResource(fake)

	_, err := res.Describe("task-1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestTaskResourceListReturnsError(t *testing.T) {
	res := NewTaskResource(&fakeTaskcluster{})

	if _, err := res.List(); err == nil {
		t.Fatalf("expected an error, got nil")
	}
}

func TestTaskResourceIDPromptLabel(t *testing.T) {
	res := NewTaskResource(&fakeTaskcluster{})

	if got := res.IDPromptLabel(); got != "task id" {
		t.Fatalf("expected %q, got %q", "task id", got)
	}
}

func TestTasksResourceScopedList(t *testing.T) {
	fake := &fakeTaskcluster{
		taskGroupTasks: taskcluster.TaskGroupTaskList{
			{
				Status: tcqueue.TaskStatusStructure{TaskID: "task-1", State: "pending"},
				Task: tcqueue.TaskDefinitionResponse{
					Metadata:   tcqueue.TaskMetadata{Name: "build"},
					WorkerType: "linux-b-large",
					Created:    tcclient.Time(time.Now().Add(-time.Hour)),
				},
			},
		},
	}
	res := NewTasksResource(fake)

	rows, err := res.ScopedList("grp-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].ID != "task-1" {
		t.Fatalf("unexpected id: %s", rows[0].ID)
	}
	if rows[0].Cells[0] != "task-1" || rows[0].Cells[1] != "build" ||
		rows[0].Cells[2] != "pending" || rows[0].Cells[3] != "linux-b-large" {
		t.Fatalf("unexpected cells: %+v", rows[0].Cells)
	}
	if rows[0].Cells[4] == "" {
		t.Fatalf("expected a non-empty AGE cell, got %+v", rows[0].Cells)
	}
}

func TestTasksResourceScopedListError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{taskGroupTasksErr: wantErr}
	res := NewTasksResource(fake)

	_, err := res.ScopedList("grp-1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestTasksResourceListReturnsError(t *testing.T) {
	res := NewTasksResource(&fakeTaskcluster{})

	if _, err := res.List(); err == nil {
		t.Fatalf("expected an error, got nil")
	}
}

func TestTasksResourceEmptyScopeResource(t *testing.T) {
	res := NewTasksResource(&fakeTaskcluster{})

	if got := res.EmptyScopeResource(); got != "workerpools" {
		t.Fatalf("expected %q, got %q", "workerpools", got)
	}
}

func TestDescribeTaskRunsIncludeTimestamps(t *testing.T) {
	scheduled := tcclient.Time(time.Now().Add(-time.Hour))
	started := tcclient.Time(time.Now().Add(-50 * time.Minute))
	resolved := tcclient.Time(time.Now().Add(-10 * time.Minute))
	fake := &fakeTaskcluster{
		task: &tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}},
		taskStatus: &tcqueue.TaskStatusStructure{
			State: "completed",
			Runs: []tcqueue.RunInformation{
				{RunID: 0, State: "completed", Scheduled: scheduled, Started: started, Resolved: resolved},
			},
		},
	}
	res := NewTaskResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(detail.Body, "scheduled:") || !strings.Contains(detail.Body, "started:") ||
		!strings.Contains(detail.Body, "resolved:") {
		t.Fatalf("expected scheduled/started/resolved timestamps in the run info, got: %s", detail.Body)
	}
}

func TestDescribeTaskRunsIncludeElapsedTimeBetweenEvents(t *testing.T) {
	scheduled := tcclient.Time(time.Now().Add(-time.Hour))
	started := tcclient.Time(time.Now().Add(-50 * time.Minute))
	resolved := tcclient.Time(time.Now().Add(-10 * time.Minute))
	fake := &fakeTaskcluster{
		task: &tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}},
		taskStatus: &tcqueue.TaskStatusStructure{
			State: "completed",
			Runs: []tcqueue.RunInformation{
				{RunID: 0, State: "completed", Scheduled: scheduled, Started: started, Resolved: resolved},
			},
		},
	}
	res := NewTaskResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(detail.Body, "10m0s after scheduled") {
		t.Fatalf("expected started timestamp annotated with elapsed time since scheduled, got: %s", detail.Body)
	}
	if !strings.Contains(detail.Body, "40m0s after started") {
		t.Fatalf("expected resolved timestamp annotated with elapsed time since started, got: %s", detail.Body)
	}
}

func TestDescribeTaskRunOmitsElapsedTimeWhenPriorEventIsUnset(t *testing.T) {
	resolved := tcclient.Time(time.Now().Add(-10 * time.Minute))
	fake := &fakeTaskcluster{
		task: &tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}},
		taskStatus: &tcqueue.TaskStatusStructure{
			State: "completed",
			Runs: []tcqueue.RunInformation{
				{RunID: 0, State: "completed", Resolved: resolved},
			},
		},
	}
	res := NewTaskResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(detail.Body, "after started") {
		t.Fatalf("expected no elapsed annotation when started is unset, got: %s", detail.Body)
	}
}

func TestDescribeTaskRunListsArtifactsForStartedRuns(t *testing.T) {
	started := tcclient.Time(time.Now().Add(-10 * time.Minute))
	fake := &fakeTaskcluster{
		task: &tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}},
		taskStatus: &tcqueue.TaskStatusStructure{
			State: "completed",
			Runs:  []tcqueue.RunInformation{{RunID: 0, State: "completed", Started: started}},
		},
		artifacts: taskcluster.ArtifactList{
			{Name: "public/logs/live_backing.log", ContentType: "text/plain", ContentLength: 2048},
			{Name: "public/build.tar.gz", ContentType: "application/gzip", ContentLength: 5 * 1024 * 1024},
		},
	}
	res := NewTaskResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(detail.Body, "public/logs/live_backing.log") ||
		!strings.Contains(detail.Body, "public/build.tar.gz") {
		t.Fatalf("expected artifact names in the body, got: %s", detail.Body)
	}
	if !strings.Contains(detail.Body, "2.0 KiB") || !strings.Contains(detail.Body, "5.0 MiB") {
		t.Fatalf("expected human-readable artifact sizes in the body, got: %s", detail.Body)
	}
}

func TestDescribeTaskRunSkipsArtifactFetchForUnstartedRuns(t *testing.T) {
	fake := &fakeTaskcluster{
		task: &tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}},
		taskStatus: &tcqueue.TaskStatusStructure{
			State: "pending",
			Runs:  []tcqueue.RunInformation{{RunID: 0, State: "pending"}},
		},
		artifactsErr: errors.New("should not be called"),
	}
	res := NewTaskResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(detail.Body, "artifacts:") {
		t.Fatalf("expected no artifacts section for an unstarted run, got: %s", detail.Body)
	}
}

func TestDescribeTaskRunShowsArtifactLoadFailureInline(t *testing.T) {
	started := tcclient.Time(time.Now().Add(-10 * time.Minute))
	fake := &fakeTaskcluster{
		task: &tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}},
		taskStatus: &tcqueue.TaskStatusStructure{
			State: "completed",
			Runs:  []tcqueue.RunInformation{{RunID: 0, State: "completed", Started: started}},
		},
		artifactsErr: errors.New("boom"),
	}
	res := NewTaskResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(detail.Body, "artifacts: (failed to load: boom)") {
		t.Fatalf("expected inline artifact load failure, got: %s", detail.Body)
	}
}

func TestDescribeTaskRunOmitsUnsetTimestamps(t *testing.T) {
	fake := &fakeTaskcluster{
		task: &tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}},
		taskStatus: &tcqueue.TaskStatusStructure{
			State: "pending",
			Runs:  []tcqueue.RunInformation{{RunID: 0, State: "pending"}},
		},
	}
	res := NewTaskResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(detail.Body, "started:") || strings.Contains(detail.Body, "resolved:") ||
		strings.Contains(detail.Body, "takenUntil:") {
		t.Fatalf("expected unset run timestamps to be omitted, got: %s", detail.Body)
	}
	if !strings.Contains(detail.Body, "scheduled:") {
		t.Fatalf("expected scheduled: to be present even when unset, got: %s", detail.Body)
	}
}

func TestTasksResourceDescribeDelegatesToDescribeTask(t *testing.T) {
	fake := &fakeTaskcluster{
		task:       &tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "build"}},
		taskStatus: &tcqueue.TaskStatusStructure{State: "completed"},
	}
	res := NewTasksResource(fake)

	detail, err := res.Describe("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Title != "Task :: build (task-1)" {
		t.Fatalf("unexpected title: %s", detail.Title)
	}
	if !strings.Contains(detail.Body, "build") || !strings.Contains(detail.Body, "completed") {
		t.Fatalf("unexpected body: %s", detail.Body)
	}
	if !strings.Contains(detail.Body, "(no runs yet)") {
		t.Fatalf("expected no-runs sentinel in body: %s", detail.Body)
	}
}
