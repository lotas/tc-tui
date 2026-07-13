package resource

import (
	"errors"
	"testing"

	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcqueue"
)

func TestTaskDependenciesResourceScopedListReturnsNavigableRows(t *testing.T) {
	fake := &fakeTaskcluster{
		task: &tcqueue.TaskDefinitionResponse{
			Dependencies: []string{"dep-1", "dep-2"},
		},
	}
	res := NewTaskDependenciesResource(fake)

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

func TestTaskDependenciesResourceScopedListPropagatesFetchError(t *testing.T) {
	fake := &fakeTaskcluster{taskErr: errors.New("boom")}
	res := NewTaskDependenciesResource(fake)

	if _, err := res.ScopedList("task-1"); err == nil {
		t.Fatalf("expected an error to propagate")
	}
}

func TestTaskDependenciesResourceListRequiresScope(t *testing.T) {
	res := NewTaskDependenciesResource(&fakeTaskcluster{})

	if _, err := res.List(); err == nil {
		t.Fatalf("expected an error for an unscoped List call")
	}
}

func TestTaskDependentsResourceScopedListReturnsTaskRows(t *testing.T) {
	fake := &fakeTaskcluster{
		dependentTasks: []tcqueue.TaskDefinitionAndStatus{
			{
				Task:   tcqueue.TaskDefinitionResponse{Metadata: tcqueue.TaskMetadata{Name: "dependent-1"}, WorkerType: "linux-b-large"},
				Status: tcqueue.TaskStatusStructure{TaskID: "dependent-task-1", State: "pending"},
			},
		},
	}
	res := NewTaskDependentsResource(fake)

	rows, err := res.ScopedList("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if row.ID != "dependent-task-1" || row.Cells[0] != "dependent-task-1" ||
		row.Cells[1] != "dependent-1" || row.Cells[2] != "pending" {
		t.Fatalf("unexpected row: %+v", row)
	}
	if row.NavTarget != nil {
		t.Fatalf("expected no NavTarget override — default selection should call Describe directly, got %+v", row.NavTarget)
	}
}

func TestTaskDependentsResourceScopedListPropagatesFetchError(t *testing.T) {
	fake := &fakeTaskcluster{dependentTasksErr: errors.New("boom")}
	res := NewTaskDependentsResource(fake)

	if _, err := res.ScopedList("task-1"); err == nil {
		t.Fatalf("expected an error to propagate")
	}
}

func TestTaskDependentsResourceListRequiresScope(t *testing.T) {
	res := NewTaskDependentsResource(&fakeTaskcluster{})

	if _, err := res.List(); err == nil {
		t.Fatalf("expected an error for an unscoped List call")
	}
}
