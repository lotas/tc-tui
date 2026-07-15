package resource

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

// maxArtifactContentLines caps how much of an artifact's content is
// rendered — a worker log can run to hundreds of thousands of lines, and an
// unbounded render both floods the screen and gets slow to scroll. Only the
// tail is kept, since the interesting bit of a failing task's log is almost
// always the end.
const maxArtifactContentLines = 1000

// maxArtifactRunFetchConcurrency bounds how many runs' artifact lists are
// fetched at once — a task with many retries shouldn't fire off unbounded
// concurrent requests.
const maxArtifactRunFetchConcurrency = 16

// TaskArtifactsResource lists the artifacts produced by every run of a task,
// scoped by the task's own ID — reachable directly from a task's own detail
// (describeTask's 'a' action) or from a single run's detail
// (TaskRunsResource.Describe's 'a' action), so it's always one keypress away
// regardless of how many runs a task has. Rows carry a RUN column and the
// resource implements Faceted, giving a tab per run (plus "All") to narrow
// down to one — the same tab-bar pattern WorkersResource uses for worker
// states. Selecting a row fetches and displays that artifact's actual
// content (see Describe).
type TaskArtifactsResource struct {
	tc taskcluster.Taskcluster
}

func NewTaskArtifactsResource(tc taskcluster.Taskcluster) *TaskArtifactsResource {
	return &TaskArtifactsResource{tc: tc}
}

func (r *TaskArtifactsResource) Name() string      { return "artifacts" }
func (r *TaskArtifactsResource) Aliases() []string { return nil }
func (r *TaskArtifactsResource) Description() string {
	return "A task's artifacts across all its runs (scoped list) — select one to view its content"
}

func (r *TaskArtifactsResource) Columns() []Column {
	return []Column{
		{Title: "RUN", Width: 6},
		{Title: "NAME"},
		{Title: "CONTENT TYPE", Width: 30},
		{Title: "SIZE", Width: 12},
	}
}

// runColumn is the Columns() index carrying a run number — FacetColumn's
// value, and what artifactRowsForRun tags every row with.
const runColumn = 0

// List is never expected to be called via normal navigation — the shell
// always either has a scope, or redirects to EmptyScopeResource() first.
func (r *TaskArtifactsResource) List() ([]Row, error) {
	return nil, fmt.Errorf("artifacts requires a task scope")
}

func (r *TaskArtifactsResource) ScopedList(taskID string) ([]Row, error) {
	status, err := r.tc.GetTaskStatus(taskID)
	if err != nil {
		return nil, err
	}

	rowsByRun := make([][]Row, len(status.Runs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxArtifactRunFetchConcurrency)

	for i, run := range status.Runs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, runID int64) {
			defer wg.Done()
			defer func() { <-sem }()
			rowsByRun[i] = artifactRowsForRun(r.tc, taskID, runID)
		}(i, run.RunID)
	}
	wg.Wait()

	var rows []Row
	for _, rs := range rowsByRun {
		rows = append(rows, rs...)
	}

	return rows, nil
}

// artifactRowsForRun fetches and renders one run's artifacts, tagging every
// row with its run number in runColumn. A fetch failure shows inline as a
// single "(failed to load)" row for that run rather than failing the whole
// list — the rest of the task's runs are still useful to see.
func artifactRowsForRun(tc taskcluster.Taskcluster, taskID string, runID int64) []Row {
	runLabel := fmt.Sprintf("%d", runID)

	artifacts, err := tc.GetArtifacts(taskID, runID)
	if err != nil {
		return []Row{{
			ID:    composeArtifactID(taskID, runID, ""),
			Cells: []string{runLabel, "(failed to load)", "", ""},
		}}
	}

	rows := make([]Row, 0, len(artifacts))
	for _, a := range artifacts {
		rows = append(rows, Row{
			ID:    composeArtifactID(taskID, runID, a.Name),
			Cells: []string{runLabel, a.Name, a.ContentType, formatBytes(a.ContentLength)},
		})
	}

	return rows
}

func (r *TaskArtifactsResource) EmptyScopeResource() string {
	return "task"
}

// FacetColumn identifies runColumn as the tab bar's underlying column.
func (r *TaskArtifactsResource) FacetColumn() int { return runColumn }

// FacetOptions derives one tab per distinct run number found in rows,
// preserving first-seen order (ScopedList appends runs in the order
// GetTaskStatus returned them — oldest first) — dynamic since a task's run
// count isn't known ahead of the fetch, unlike WorkersResource's fixed
// state list.
func (r *TaskArtifactsResource) FacetOptions(rows []Row) []string {
	seen := make(map[string]bool)
	options := make([]string, 0, len(rows))
	for _, row := range rows {
		run := row.Cells[runColumn]
		if !seen[run] {
			seen[run] = true
			options = append(options, run)
		}
	}
	return options
}

// Describe fetches and renders one artifact's content, tail-truncated at
// maxArtifactContentLines — see that constant's doc comment.
func (r *TaskArtifactsResource) Describe(id string) (Detail, error) {
	taskID, runID, name, err := parseArtifactID(id)
	if err != nil {
		return Detail{}, err
	}

	content, err := r.tc.GetArtifactContent(taskID, runID, name)
	if err != nil {
		return Detail{}, err
	}

	return Detail{
		Title: fmt.Sprintf("Task :: %s :: Run %d :: %s", taskID, runID, name),
		Body:  formatArtifactContent(content),
	}, nil
}

func (r *TaskArtifactsResource) RefreshInterval() time.Duration { return 0 }

// ListWebURL links to the scoped task's own page — there's no dedicated
// artifacts page in the web UI, and scope is that task's ID.
func (r *TaskArtifactsResource) ListWebURL(rootURL, scope string) string {
	return taskWebURL(rootURL, scope)
}

// DetailWebURL links to the same task page as ListWebURL — the web UI has no
// per-artifact-content anchor of its own.
func (r *TaskArtifactsResource) DetailWebURL(rootURL, id string) string {
	taskID, _, _, err := parseArtifactID(id)
	if err != nil {
		return ""
	}
	return taskWebURL(rootURL, taskID)
}

// formatArtifactContent renders raw artifact content for display, keeping
// only the last maxArtifactContentLines lines.
func formatArtifactContent(content string) string {
	if strings.TrimSpace(content) == "" {
		return "(empty)"
	}

	lines := strings.Split(content, "\n")
	if len(lines) <= maxArtifactContentLines {
		return content
	}

	tail := lines[len(lines)-maxArtifactContentLines:]
	return fmt.Sprintf("[yellow](showing last %d of %d lines)[white]\n\n%s",
		maxArtifactContentLines, len(lines), strings.Join(tail, "\n"))
}

// composeArtifactID and parseArtifactID identify a single artifact as
// "<taskID>/<runID>::<name>" — "::" as the final separator rather than "/",
// since artifact names routinely contain their own "/" path segments (e.g.
// "public/logs/live_backing.log").
func composeArtifactID(taskID string, runID int64, name string) string {
	return fmt.Sprintf("%s::%s", composeRunID(taskID, runID), name)
}

func parseArtifactID(id string) (taskID string, runID int64, name string, err error) {
	idx := strings.Index(id, "::")
	if idx < 0 {
		return "", 0, "", fmt.Errorf("invalid artifact id %q", id)
	}

	taskID, runID, err = parseRunID(id[:idx])
	if err != nil {
		return "", 0, "", err
	}

	return taskID, runID, id[idx+2:], nil
}
