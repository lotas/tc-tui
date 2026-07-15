package resource

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/glamour"
	"github.com/rivo/tview"

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

// Describe fetches and renders one artifact's content — see
// renderArtifactBody for how binary/huge/markdown-ish content is handled
// differently.
func (r *TaskArtifactsResource) Describe(id string) (Detail, error) {
	taskID, runID, name, err := parseArtifactID(id)
	if err != nil {
		return Detail{}, err
	}

	content, contentType, truncated, err := r.tc.GetArtifactContent(taskID, runID, name)
	if err != nil {
		return Detail{}, err
	}

	return Detail{
		Title: fmt.Sprintf("Task :: %s :: Run %d :: %s", taskID, runID, name),
		Body:  renderArtifactBody(contentType, content, truncated),
	}, nil
}

func (r *TaskArtifactsResource) RefreshInterval() time.Duration { return 0 }

// ListWebURL links to the scoped task's own page — there's no dedicated
// artifacts page in the web UI, and scope is that task's ID.
func (r *TaskArtifactsResource) ListWebURL(rootURL, scope string) string {
	return taskWebURL(rootURL, scope)
}

// DetailWebURL links directly to the artifact's own content (signed when
// authenticated) rather than the task's web page, so 'o' doubles as the
// "open/download it" escape hatch for content this view won't render raw
// (binary, or larger than taskcluster.MaxArtifactContentBytes).
func (r *TaskArtifactsResource) DetailWebURL(rootURL, id string) string {
	taskID, runID, name, err := parseArtifactID(id)
	if err != nil {
		return ""
	}

	artifactURL, err := r.tc.GetArtifactURL(taskID, runID, name)
	if err != nil {
		return ""
	}
	return artifactURL
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

// maxArtifactRenderBytes caps how much rendered text is ever handed to
// tview's TextView.SetText. tview runs on a single goroutine shared with
// tcell's key-event loop (see shell.Shell's loadGeneration doc comment), so
// a slow SetText call — its word-wrap/color-tag parsing cost scales with
// input size — freezes the entire UI, including quitting, until it returns.
// Sized to comfortably fit chroma's syntax-highlighted expansion of content
// right at maxArtifactHighlightBytes (~4.4x for JSON, per that constant's
// calibration) without truncating it; it's also a backstop independent of
// maxArtifactContentLines, since even line-truncated text could be huge if
// a handful of lines are each very long (e.g. a minified JSON blob on one
// line).
const maxArtifactRenderBytes = 512 * 1024 // 512 KiB

// renderArtifactBody renders one artifact's content for display: binary
// content shows metadata instead of raw bytes (garbled and possibly slow to
// render); everything else renders via renderHighlightedOrPlain, preserved
// in its original form rather than reformatted — debugging an artifact means
// seeing exactly what it contains, which is a different concern from a
// task's own (typically tiny) description/payload getting a prettified
// render. A banner is shown whenever what's displayed may be incomplete —
// either because the fetch itself hit taskcluster.MaxArtifactContentBytes,
// or because the rendered text hit maxArtifactRenderBytes.
func renderArtifactBody(contentType, content string, truncated bool) string {
	if isBinaryArtifact(contentType, content) {
		body := fmt.Sprintf(
			"[green]Content-Type:[white] %s\n[green]Size:[white] %s\n\n"+
				"[yellow](binary content isn't rendered here — press 'o' to open/download it)[white]",
			contentTypeOrUnknown(contentType), formatBytes(int64(len(content))),
		)
		if truncated {
			body += "\n[yellow](this size reflects only what was fetched — the artifact may be larger)[white]"
		}
		return body
	}

	body := renderHighlightedOrPlain(contentType, content)

	capped := false
	if len(body) > maxArtifactRenderBytes {
		body = body[:maxArtifactRenderBytes]
		capped = true
	}

	if !truncated && !capped {
		return body
	}

	return fmt.Sprintf(
		"[yellow](showing only part of this artifact — press 'o' to open/download the full content)[white]\n\n%s",
		body,
	)
}

// maxArtifactHighlightBytes caps how much content gets syntax-highlighted
// (see renderHighlightedOrPlain) rather than shown as plain text. Chroma's
// highlighting cost scales roughly linearly with input size — measured at
// ~0.4ms/KiB for JSON — so this keeps the highlighting pass itself
// comfortably under 30ms even before tview's own SetText cost on the
// resulting (larger, tag-heavy) output; a size that felt "instant" for a
// one-off action rather than "the UI just stalled."
const maxArtifactHighlightBytes = 64 * 1024 // 64 KiB

// renderHighlightedOrPlain syntax-highlights content in its original,
// unmodified form (via renderHighlightedArtifact) when both a chroma
// language is known for contentType and content is small enough for that to
// stay fast; otherwise it falls back to renderArtifactText.
func renderHighlightedOrPlain(contentType, content string) string {
	lang, ok := chromaLanguageFor(contentType)
	if !ok || len(content) > maxArtifactHighlightBytes {
		return renderArtifactText(content)
	}

	highlighted, err := renderHighlightedArtifact(lang, content)
	if err != nil {
		return renderArtifactText(content)
	}
	return highlighted
}

// chromaLanguageFor maps a Content-Type to the language chroma should
// highlight it as, for content types worth the highlighting cost.
func chromaLanguageFor(contentType string) (string, bool) {
	switch baseContentType(contentType) {
	case "application/json":
		return "json", true
	case "application/x-yaml", "application/yaml", "text/yaml", "text/x-yaml":
		return "yaml", true
	case "application/xml", "text/xml":
		return "xml", true
	case "application/javascript", "text/javascript":
		return "javascript", true
	case "text/markdown":
		return "markdown", true
	default:
		return "", false
	}
}

// renderHighlightedArtifact syntax-highlights content in its original,
// unmodified form — wrapped verbatim in a fenced code block of the given
// language, never reformatted — via glamour/chroma. Unlike the shared
// renderGlamour helper (render.go, used for a task's own first-party
// description/payload), this escapes literal "[...]" sequences before
// translating glamour's ANSI codes into tview tags — see
// escapeIgnoringANSI's doc comment for why naively escaping the whole
// ANSI-coded string (tview.Escape then tview.TranslateANSI, as renderGlamour
// does) actively corrupts content like JSON that's full of literal "[" "]"
// sitting right next to a color code.
func renderHighlightedArtifact(lang, content string) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		return "", err
	}

	out, err := renderer.Render(fmt.Sprintf("```%s\n%s\n```", lang, content))
	if err != nil {
		return "", err
	}

	return tview.TranslateANSI(escapeIgnoringANSI(strings.TrimSpace(out))), nil
}

// ansiCSIPattern matches a single ANSI CSI escape sequence — ESC "[" then
// parameter bytes (digits/semicolons) then one final letter, e.g.
// "\x1b[38;5;208m" or "\x1b[0m" — the form tview.TranslateANSI/chroma's
// color codes use.
var ansiCSIPattern = regexp.MustCompile("\x1b\\[[0-9;]*[A-Za-z]")

// escapeIgnoringANSI applies tview.Escape only to the plain-text runs of s,
// leaving any ANSI CSI escape sequence untouched. tview.Escape's own
// tag-detection regex has no concept of an ANSI escape sequence — given
// "...\x1b[38;5;187m],..." (an ANSI color code immediately followed by a
// literal "]," from real content, e.g. a JSON array's own closing bracket),
// it can match starting at the *ANSI code's own* "[", treating the code's
// parameter digits as tag content and consuming the adjacent literal "]" as
// that "tag"'s close — inserting a spurious "[" right before it and
// corrupting the text. Escaping only ANSI-free substrings means Escape
// never sees an ANSI code's bracket at all, so this can't happen — while
// still leaving the real ANSI codes untouched for TranslateANSI to convert
// afterward.
func escapeIgnoringANSI(s string) string {
	matches := ansiCSIPattern.FindAllStringIndex(s, -1)
	if len(matches) == 0 {
		return tview.Escape(s)
	}

	var b strings.Builder
	last := 0
	for _, loc := range matches {
		b.WriteString(tview.Escape(s[last:loc[0]]))
		b.WriteString(s[loc[0]:loc[1]])
		last = loc[1]
	}
	b.WriteString(tview.Escape(s[last:]))
	return b.String()
}

// renderArtifactText renders arbitrary plain-text artifact content safely:
// escapeIgnoringANSI neutralizes any literal "[...]" in the raw text (e.g. a
// log line like "[INFO] ...") that would otherwise be misread as a tview
// color/region tag — including one sitting immediately after a real ANSI
// color code (see that function's doc comment for why a plain tview.Escape
// isn't safe there) — and tview.TranslateANSI then converts genuine ANSI
// color codes (common in a live worker log) into real tview tags.
// Content is tail-truncated to the last maxArtifactContentLines lines, since
// the interesting bit of a failing task's log is almost always the end.
func renderArtifactText(content string) string {
	body := tview.TranslateANSI(escapeIgnoringANSI(content))
	if strings.TrimSpace(body) == "" {
		return "(empty)"
	}

	lines := strings.Split(body, "\n")
	if len(lines) <= maxArtifactContentLines {
		return body
	}

	tail := lines[len(lines)-maxArtifactContentLines:]
	return fmt.Sprintf("[yellow](showing last %d of %d lines)[white]\n\n%s",
		maxArtifactContentLines, len(lines), strings.Join(tail, "\n"))
}

// isBinaryArtifact reports whether content shouldn't be rendered as text. A
// declared textual content-type is trusted outright; otherwise content is
// sniffed for a NUL byte or invalid UTF-8 — cheap enough to run on the whole
// string since GetArtifactContent already caps it at MaxArtifactContentBytes
// (sampling a fixed prefix instead would risk slicing mid multi-byte rune
// right at the sample boundary, misclassifying valid text as binary).
func isBinaryArtifact(contentType, content string) bool {
	if isTextualContentType(contentType) {
		return false
	}

	if strings.ContainsRune(content, '\x00') {
		return true
	}

	return !validUTF8AllowingTruncatedTail(content)
}

// validUTF8AllowingTruncatedTail reports whether content is valid UTF-8,
// tolerating an incomplete multi-byte sequence dangling at the very end —
// GetArtifactContent's size cap can produce exactly that for an otherwise-
// valid huge text artifact cut off mid-rune, which would otherwise make it
// look binary.
func validUTF8AllowingTruncatedTail(content string) bool {
	if utf8.ValidString(content) {
		return true
	}
	for cut := 1; cut <= 3 && cut < len(content); cut++ {
		if utf8.ValidString(content[:len(content)-cut]) {
			return true
		}
	}
	return false
}

// isTextualContentType reports whether contentType is a MIME type this view
// is willing to trust as text outright without sniffing its bytes.
func isTextualContentType(contentType string) bool {
	switch ct := baseContentType(contentType); {
	case strings.HasPrefix(ct, "text/"):
		return true
	case ct == "application/json", ct == "application/xml", ct == "application/javascript",
		ct == "application/x-yaml", ct == "application/yaml", ct == "application/x-ndjson":
		return true
	default:
		return false
	}
}

// baseContentType strips any parameters (e.g. "; charset=utf-8") and
// normalizes case from a Content-Type header value.
func baseContentType(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	return ct
}

// contentTypeOrUnknown renders a Content-Type header value for display,
// covering artifacts the queue served with no header at all.
func contentTypeOrUnknown(contentType string) string {
	if contentType == "" {
		return "(unknown)"
	}
	return contentType
}
