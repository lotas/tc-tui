package resource

import (
	"errors"
	"testing"

	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcqueue"
)

func TestTaskDepsResourceScopedListReturnsNavigableRows(t *testing.T) {
	fake := &fakeTaskcluster{
		task: &tcqueue.TaskDefinitionResponse{
			Dependencies: []string{"dep-1", "dep-2"},
		},
	}
	res := NewTaskDepsResource(fake)

	rows, err := res.ScopedList("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	for i, depID := range []string{"dep-1", "dep-2"} {
		row := rows[i]
		if row.ID != depID || row.Cells[0] != depID {
			t.Fatalf("unexpected row %d: %+v", i, row)
		}
		if row.NavTarget == nil || row.NavTarget.ResourceName != "task" ||
			row.NavTarget.ID != depID || row.NavTarget.Kind != NavDetail {
			t.Fatalf("unexpected NavTarget for row %d: %+v", i, row.NavTarget)
		}
	}
}

func TestTaskDepsResourceScopedListPropagatesFetchError(t *testing.T) {
	fake := &fakeTaskcluster{taskErr: errors.New("boom")}
	res := NewTaskDepsResource(fake)

	if _, err := res.ScopedList("task-1"); err == nil {
		t.Fatalf("expected an error to propagate")
	}
}

func TestTaskDepsResourceListRequiresScope(t *testing.T) {
	res := NewTaskDepsResource(&fakeTaskcluster{})

	if _, err := res.List(); err == nil {
		t.Fatalf("expected an error for an unscoped List call")
	}
}
