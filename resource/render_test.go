package resource

import (
	"strings"
	"testing"
	"time"
)

func TestRenderYAMLEmptyInputRendersNone(t *testing.T) {
	for _, raw := range []string{"", "null", "{}", "[]"} {
		if got := renderYAML([]byte(raw)); got != "(none)" {
			t.Fatalf("renderYAML(%q) = %q, want %q", raw, got, "(none)")
		}
	}
}

func TestRenderYAMLValidInputProducesColoredOutput(t *testing.T) {
	got := renderYAML([]byte(`{"command":["echo","hi"],"maxRunTime":600}`))

	if got == "" {
		t.Fatalf("expected non-empty rendered output")
	}
	if strings.Contains(got, "command") == false {
		t.Fatalf("expected the rendered output to still contain the field name, got: %s", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("expected ANSI escapes to be translated into tview tags, got raw ANSI in: %q", got)
	}
}

func TestRenderYAMLMalformedInputFallsBackToRawText(t *testing.T) {
	raw := `{not valid json`
	if got := renderYAML([]byte(raw)); got != raw {
		t.Fatalf("renderYAML(%q) = %q, want the raw string back", raw, got)
	}
}

func TestTaskStateColor(t *testing.T) {
	cases := map[string]string{
		"completed":   "green",
		"failed":      "red",
		"exception":   "red",
		"running":     "yellow",
		"pending":     "white",
		"unscheduled": "white",
		"":            "white",
	}
	for state, want := range cases {
		if got := taskStateColor(state); got != want {
			t.Fatalf("taskStateColor(%q) = %q, want %q", state, got, want)
		}
	}
}

func TestRenderTaskState(t *testing.T) {
	if got := renderTaskState("completed"); got != "[green]completed[white]" {
		t.Fatalf("renderTaskState(completed) = %q", got)
	}
	if got := renderTaskState(""); got != "" {
		t.Fatalf("renderTaskState(\"\") = %q, want \"\"", got)
	}
}

func TestTaskStateBadge(t *testing.T) {
	if got := taskStateBadge("failed"); got != "[red]failed[white] " {
		t.Fatalf("taskStateBadge(failed) = %q", got)
	}
	if got := taskStateBadge(""); got != "" {
		t.Fatalf("taskStateBadge(\"\") = %q, want \"\"", got)
	}
}

func TestRenderMarkdownEmptyInputRendersNone(t *testing.T) {
	if got := renderMarkdown("   "); got != "(none)" {
		t.Fatalf("renderMarkdown(whitespace) = %q, want %q", got, "(none)")
	}
}

func TestRenderMarkdownValidInputProducesOutput(t *testing.T) {
	got := renderMarkdown("This task does **important** things.")

	if got == "" {
		t.Fatalf("expected non-empty rendered output")
	}
	if !strings.Contains(got, "important") {
		t.Fatalf("expected the rendered output to still contain the text, got: %s", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("expected ANSI escapes to be translated into tview tags, got raw ANSI in: %q", got)
	}
}

func TestFormatAgeZeroTimeIsEmpty(t *testing.T) {
	if got := formatAge(time.Time{}); got != "" {
		t.Fatalf("formatAge(zero) = %q, want empty string", got)
	}
}

func TestFormatAgeNonZeroTimeIsNonEmpty(t *testing.T) {
	got := formatAge(time.Now().Add(-2 * time.Hour))
	if got == "" {
		t.Fatalf("expected a non-empty age string")
	}
	if !strings.Contains(got, "h") {
		t.Fatalf("expected an hours component in a ~2h-old timestamp, got %q", got)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{2048, "2.0 KiB"},
		{5 * 1024 * 1024, "5.0 MiB"},
		{3 * 1024 * 1024 * 1024, "3.0 GiB"},
	}

	for _, tt := range tests {
		if got := formatBytes(tt.n); got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
