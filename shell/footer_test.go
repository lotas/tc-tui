package shell

import (
	"strings"
	"testing"

	"github.com/taskcluster/tc-tui/resource"
)

func TestSplitCommandNameOnly(t *testing.T) {
	name, scope := splitCommand("workerpools")
	if name != "workerpools" || scope != "" {
		t.Fatalf("unexpected split: name=%q scope=%q", name, scope)
	}
}

func TestSplitCommandNameAndScope(t *testing.T) {
	name, scope := splitCommand("workers gcp/linux-b-gpu")
	if name != "workers" || scope != "gcp/linux-b-gpu" {
		t.Fatalf("unexpected split: name=%q scope=%q", name, scope)
	}
}

func TestSplitCommandCollapsesExtraWhitespace(t *testing.T) {
	name, scope := splitCommand("  workers   gcp/linux-b-gpu  ")
	if name != "workers" || scope != "gcp/linux-b-gpu" {
		t.Fatalf("unexpected split: name=%q scope=%q", name, scope)
	}
}

func TestSplitCommandEmptyInput(t *testing.T) {
	name, scope := splitCommand("")
	if name != "" || scope != "" {
		t.Fatalf("unexpected split: name=%q scope=%q", name, scope)
	}
}

func TestSplitCommandWhitespaceOnlyInput(t *testing.T) {
	name, scope := splitCommand("   ")
	if name != "" || scope != "" {
		t.Fatalf("unexpected split: name=%q scope=%q", name, scope)
	}
}

func TestRenderFooterHintsShowsTabHintWhenFaceted(t *testing.T) {
	s := New(resource.NewRegistry())
	s.currentFaceted = fakeFacetedColumn{column: 0, options: []string{"aws"}}

	s.renderFooterHints()

	text := s.footerHint.GetText(false)
	if !strings.Contains(text, "Tab") || !strings.Contains(text, "Shift+Tab") {
		t.Fatalf("expected Tab/Shift+Tab hint, got %q", text)
	}
}

func TestRenderFooterHintsOmitsTabHintWhenNotFaceted(t *testing.T) {
	s := New(resource.NewRegistry())

	s.renderFooterHints()

	text := s.footerHint.GetText(false)
	if strings.Contains(text, "Shift+Tab") {
		t.Fatalf("expected no Tab hint for a non-faceted resource, got %q", text)
	}
}
