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
