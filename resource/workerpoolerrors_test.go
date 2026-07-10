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

func TestErrorsResourceScopedListPoolWide(t *testing.T) {
	fake := &fakeTaskcluster{
		workerPoolErrors: taskcluster.WorkerPoolErrorList{
			{
				WorkerPoolID:   "gcp/pool-a",
				ErrorID:        "err-1",
				Title:          "launch failed",
				Kind:           "provider-error",
				LaunchConfigID: "lc-1",
				Reported:       tcclient.Time(time.Now()),
			},
		},
	}
	res := NewErrorsResource(fake)

	rows, err := res.ScopedList("gcp/pool-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.workerPoolErrorsLaunchConfigID != "" {
		t.Fatalf("expected no launch config filter, got %q", fake.workerPoolErrorsLaunchConfigID)
	}
	if len(rows) != 1 || rows[0].ID != "gcp/pool-a::err-1" || rows[0].Cells[1] != "launch failed" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestErrorsResourceScopedListLaunchConfigScoped(t *testing.T) {
	fake := &fakeTaskcluster{}
	res := NewErrorsResource(fake)

	if _, err := res.ScopedList("gcp/pool-a::lc-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.workerPoolErrorsLaunchConfigID != "lc-1" {
		t.Fatalf("expected launch config filter %q, got %q", "lc-1", fake.workerPoolErrorsLaunchConfigID)
	}
}

func TestErrorsResourceScopedListError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{workerPoolErrorsErr: wantErr}
	res := NewErrorsResource(fake)

	_, err := res.ScopedList("gcp/pool-a")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestErrorsResourceListReturnsScopeRequiredError(t *testing.T) {
	res := NewErrorsResource(&fakeTaskcluster{})

	if _, err := res.List(); err == nil {
		t.Fatalf("expected an error, got nil")
	}
}

func TestErrorsResourceEmptyScopeResource(t *testing.T) {
	res := NewErrorsResource(&fakeTaskcluster{})

	if got := res.EmptyScopeResource(); got != "workerpools" {
		t.Fatalf("expected %q, got %q", "workerpools", got)
	}
}

func TestErrorsResourceDescribe(t *testing.T) {
	fake := &fakeTaskcluster{
		workerPoolError: &tcworkermanager.WorkerPoolError{
			WorkerPoolID:   "gcp/pool-a",
			ErrorID:        "err-1",
			Title:          "launch failed",
			Kind:           "provider-error",
			Description:    "something went wrong",
			LaunchConfigID: "lc-1",
			Reported:       tcclient.Time(time.Now()),
		},
	}
	res := NewErrorsResource(fake)

	detail, err := res.Describe("gcp/pool-a::err-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Title != "Worker Pool Error :: err-1" {
		t.Fatalf("unexpected title: %s", detail.Title)
	}
	if !strings.Contains(detail.Body, "launch failed") || !strings.Contains(detail.Body, "something went wrong") {
		t.Fatalf("unexpected body: %s", detail.Body)
	}
}

func TestErrorsResourceDescribeError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{workerPoolErrorErr: wantErr}
	res := NewErrorsResource(fake)

	_, err := res.Describe("gcp/pool-a::err-1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
