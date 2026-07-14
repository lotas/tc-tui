package resource

import (
	"strings"
	"testing"
)

func TestFieldRowRendersLabelValuePairs(t *testing.T) {
	got := fieldRow(20, "Created", "2026-01-01", "Archived", "true")
	if !strings.Contains(got, "[green]Created:[white] 2026-01-01") ||
		!strings.Contains(got, "[green]Archived:[white] true") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestFieldRowPadsToVisibleWidthIgnoringColorTags(t *testing.T) {
	got := fieldRow(20, "A", "1", "B", "2")

	first := "[green]A:[white] 1"
	want := first + strings.Repeat(" ", 20-visibleWidth(first)) + "[green]B:[white] 2\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFieldRowFallsBackToSingleSpaceWhenFieldExceedsWidth(t *testing.T) {
	got := fieldRow(5, "Provisioner", "gcp-provider-name", "B", "2")
	want := "[green]Provisioner:[white] gcp-provider-name [green]B:[white] 2\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFieldRowDoesNotPadTheLastField(t *testing.T) {
	got := fieldRow(30, "A", "1")
	if got != "[green]A:[white] 1\n" {
		t.Fatalf("expected the sole field unpadded, got %q", got)
	}
}
