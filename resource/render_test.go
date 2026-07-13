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
