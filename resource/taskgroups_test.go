package resource

import (
	"errors"
	"strings"
	"testing"
	"time"

	tcclient "github.com/taskcluster/taskcluster/v101/clients/client-go"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcqueue"
)

func TestTaskGroupResourceDescribe(t *testing.T) {
	fake := &fakeTaskcluster{
		taskGroup: &tcqueue.TaskGroupDefinitionResponse{
			TaskGroupID: "grp-1",
			SchedulerID: "taskcluster-github",
			Expires:     tcclient.Time(time.Now()),
		},
	}
	res := NewTaskGroupResource(fake)

	detail, err := res.Describe("grp-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Title != "Task Group :: grp-1" {
		t.Fatalf("unexpected title: %s", detail.Title)
	}
	if !strings.Contains(detail.Body, "taskcluster-github") || !strings.Contains(detail.Body, "not sealed") {
		t.Fatalf("unexpected body: %s", detail.Body)
	}
	if len(detail.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(detail.Actions))
	}
	action := detail.Actions[0]
	if action.Key != 't' || action.Target.ResourceName != "tasks" ||
		action.Target.ID != "grp-1" || action.Target.Kind != NavScopedList {
		t.Fatalf("unexpected action: %+v", action)
	}
}

func TestTaskGroupResourceDescribeSealed(t *testing.T) {
	fake := &fakeTaskcluster{
		taskGroup: &tcqueue.TaskGroupDefinitionResponse{
			TaskGroupID: "grp-1",
			Sealed:      tcclient.Time(time.Now()),
		},
	}
	res := NewTaskGroupResource(fake)

	detail, err := res.Describe("grp-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(detail.Body, "not sealed") {
		t.Fatalf("expected a sealed timestamp, got: %s", detail.Body)
	}
}

func TestTaskGroupResourceDescribeError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{taskGroupErr: wantErr}
	res := NewTaskGroupResource(fake)

	_, err := res.Describe("grp-1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestTaskGroupResourceListReturnsError(t *testing.T) {
	res := NewTaskGroupResource(&fakeTaskcluster{})

	if _, err := res.List(); err == nil {
		t.Fatalf("expected an error, got nil")
	}
}

func TestTaskGroupResourceIDPromptLabel(t *testing.T) {
	res := NewTaskGroupResource(&fakeTaskcluster{})

	if got := res.IDPromptLabel(); got != "task group id" {
		t.Fatalf("expected %q, got %q", "task group id", got)
	}
}
