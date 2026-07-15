package resource

import (
	"errors"
	"strings"
	"testing"

	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcqueue"

	"github.com/taskcluster/tc-tui/taskcluster"
)

func TestTaskArtifactsResourceScopedListReturnsOneRowPerArtifactAcrossRuns(t *testing.T) {
	fake := &fakeTaskcluster{
		taskStatus: &tcqueue.TaskStatusStructure{
			Runs: []tcqueue.RunInformation{{RunID: 0}, {RunID: 1}},
		},
		// The fake returns the same artifact list regardless of which run
		// was requested, so both runs' rows carry the same artifact but a
		// different RUN cell — what matters here is that ScopedList fetches
		// per run and tags each row accordingly.
		artifacts: taskcluster.ArtifactList{
			{Name: "public/logs/live_backing.log", ContentType: "text/plain", ContentLength: 2048},
		},
	}
	res := NewTaskArtifactsResource(fake)

	rows, err := res.ScopedList("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	for i, runLabel := range []string{"0", "1"} {
		row := rows[i]
		if row.ID != "task-1/"+runLabel+"::public/logs/live_backing.log" ||
			row.Cells[0] != runLabel || row.Cells[1] != "public/logs/live_backing.log" || row.Cells[2] != "text/plain" {
			t.Fatalf("unexpected row %d: %+v", i, row)
		}
		if row.NavTarget != nil {
			t.Fatalf("expected no NavTarget override — default selection should call Describe directly, got %+v", row.NavTarget)
		}
	}
}

func TestTaskArtifactsResourceScopedListPropagatesTaskStatusFetchError(t *testing.T) {
	fake := &fakeTaskcluster{taskStatusErr: errors.New("boom")}
	res := NewTaskArtifactsResource(fake)

	if _, err := res.ScopedList("task-1"); err == nil {
		t.Fatalf("expected an error to propagate")
	}
}

func TestArtifactRowsForRunShowsFailureInlineOnFetchError(t *testing.T) {
	fake := &fakeTaskcluster{artifactsErr: errors.New("boom")}

	rows := artifactRowsForRun(fake, "task-1", 0)
	if len(rows) != 1 || rows[0].Cells[0] != "0" || rows[0].Cells[1] != "(failed to load)" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestTaskArtifactsResourceFacetOptionsDerivesDistinctRunsInOrder(t *testing.T) {
	res := NewTaskArtifactsResource(&fakeTaskcluster{})

	rows := []Row{
		{Cells: []string{"0", "a"}},
		{Cells: []string{"1", "b"}},
		{Cells: []string{"0", "c"}},
	}
	got := res.FacetOptions(rows)
	want := []string{"0", "1"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected facet options: %+v", got)
	}
	if res.FacetColumn() != 0 {
		t.Fatalf("expected FacetColumn 0, got %d", res.FacetColumn())
	}
}

func TestTaskArtifactsResourceListRequiresScope(t *testing.T) {
	res := NewTaskArtifactsResource(&fakeTaskcluster{})

	if _, err := res.List(); err == nil {
		t.Fatalf("expected an error for an unscoped List call")
	}
}

func TestTaskArtifactsResourceDescribeRendersContent(t *testing.T) {
	fake := &fakeTaskcluster{artifactContent: "line one\nline two\n"}
	res := NewTaskArtifactsResource(fake)

	detail, err := res.Describe("task-1/0::public/logs/live_backing.log")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Title != "Task :: task-1 :: Run 0 :: public/logs/live_backing.log" {
		t.Fatalf("unexpected title: %s", detail.Title)
	}
	if detail.Body != "line one\nline two\n" {
		t.Fatalf("unexpected body: %q", detail.Body)
	}
}

func TestTaskArtifactsResourceDescribeTruncatesLargeContent(t *testing.T) {
	lines := make([]string, maxArtifactContentLines+10)
	for i := range lines {
		lines[i] = "log line"
	}
	fake := &fakeTaskcluster{artifactContent: strings.Join(lines, "\n")}
	res := NewTaskArtifactsResource(fake)

	detail, err := res.Describe("task-1/0::public/logs/live_backing.log")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(detail.Body, "showing last 1000 of 1010 lines") {
		t.Fatalf("expected a truncation notice, got: %s", detail.Body[:200])
	}
	if strings.Count(detail.Body, "log line") != maxArtifactContentLines {
		t.Fatalf("expected exactly %d lines of content, got %d", maxArtifactContentLines, strings.Count(detail.Body, "log line"))
	}
}

func TestTaskArtifactsResourceDescribeRejectsMalformedID(t *testing.T) {
	res := NewTaskArtifactsResource(&fakeTaskcluster{})

	if _, err := res.Describe("not-an-artifact-id"); err == nil {
		t.Fatalf("expected an error for a malformed artifact id")
	}
}

func TestTaskArtifactsResourceDescribePropagatesFetchError(t *testing.T) {
	fake := &fakeTaskcluster{artifactContentErr: errors.New("boom")}
	res := NewTaskArtifactsResource(fake)

	if _, err := res.Describe("task-1/0::public/logs/live_backing.log"); err == nil {
		t.Fatalf("expected an error to propagate")
	}
}

func TestTaskArtifactsResourceDetailWebURLLinksToTaskPage(t *testing.T) {
	res := NewTaskArtifactsResource(&fakeTaskcluster{})

	got := res.DetailWebURL("https://tc.example.com", "task-1/0::public/logs/live_backing.log")
	if got == "" {
		t.Fatalf("expected a non-empty URL")
	}
}
