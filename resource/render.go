package resource

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/rivo/tview"
	"sigs.k8s.io/yaml"
)

// renderMarkdown renders text (a Taskcluster field documented/treated as
// markdown, e.g. a task's description) through glamour and translates its
// ANSI output into tview's region-tag markup. Empty input renders as
// "(none)"; a render failure falls back to the raw text rather than hiding
// it.
func renderMarkdown(text string) string {
	if strings.TrimSpace(text) == "" {
		return "(none)"
	}

	out, err := renderGlamour(text)
	if err != nil {
		return text
	}
	return out
}

// renderYAML renders a json.RawMessage as a syntax-highlighted YAML code
// block. Empty/null input renders as "(none)"; a conversion or render
// failure falls back to the raw JSON text.
func renderYAML(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" || trimmed == "[]" {
		return "(none)"
	}

	yamlBytes, err := yaml.JSONToYAML(raw)
	if err != nil {
		return trimmed
	}

	out, err := renderGlamour(fmt.Sprintf("```yaml\n%s\n```", yamlBytes))
	if err != nil {
		return trimmed
	}
	return out
}

// renderGlamour renders markdown text to ANSI via glamour, then translates
// that ANSI into tview's own region-tag markup. A fresh TermRenderer is
// created per call: Describe runs at most a handful of times per
// navigation, not in a hot loop, so the setup cost isn't worth caching.
func renderGlamour(markdown string) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(0), // the shell's own TextView already wraps
	)
	if err != nil {
		return "", err
	}

	out, err := renderer.Render(markdown)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(tview.TranslateANSI(out)), nil
}

// formatAge renders how long ago t was, e.g. "2h15m3s" — used by list
// columns where a raw timestamp alone doesn't answer "how long has this
// been in this state". A zero t (not yet applicable) renders as "".
func formatAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return time.Since(t).Round(time.Second).String()
}

// taskStateColor maps a task/run state to a tview color name, so a task's
// overall health reads at a glance (e.g. in the Detail title, always visible
// above the fold) rather than requiring a scroll past a possibly-huge
// payload down to the runs section to see if it succeeded or failed.
func taskStateColor(state string) string {
	switch state {
	case "completed":
		return "green"
	case "failed", "exception":
		return "red"
	case "running":
		return "yellow"
	default: // "pending", "unscheduled"
		return "white"
	}
}

// renderTaskState colors state via taskStateColor, e.g. for a list's STATE
// column or a run's inline state — anywhere the bare value itself (not a
// label:value pair) needs the same color coding describeTask's title uses.
// Empty state (a fetch-failure placeholder, or a test that doesn't bother
// setting it) passes through unchanged rather than wrapping nothing in a
// pointless color span.
func renderTaskState(state string) string {
	if state == "" {
		return state
	}
	return fmt.Sprintf("[%s]%s[white]", taskStateColor(state), state)
}

// taskStateBadge renders state as a colored prefix (see renderTaskState) for
// the Detail title, with a trailing space to separate it from whatever
// follows (the task name). Empty state renders as "" rather than a bare
// trailing space.
func taskStateBadge(state string) string {
	if state == "" {
		return ""
	}
	return renderTaskState(state) + " "
}

// formatWorker renders a "group/id" worker pair — used by list columns for
// runs that may not have a worker assigned yet (e.g. a pending run), which
// would otherwise render as a bare "/".
func formatWorker(group, id string) string {
	if group == "" && id == "" {
		return "n/a"
	}
	return fmt.Sprintf("%s/%s", group, id)
}

// elapsedSince renders how long after `from` (labelled sinceLabel) the `to`
// event happened, e.g. " (5m12s after scheduled)" — used to annotate task
// run timestamps so a reader doesn't have to diff two absolute times by
// hand. Either time being zero (event not yet reached) renders as "".
func elapsedSince(from, to time.Time, sinceLabel string) string {
	if from.IsZero() || to.IsZero() {
		return ""
	}
	return fmt.Sprintf(" (%s after %s)", to.Sub(from).Round(time.Second), sinceLabel)
}

// formatBytes renders a byte count in human-readable units, e.g. "4.2 MiB".
func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
