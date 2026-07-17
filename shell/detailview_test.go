package shell

import (
	"strings"
	"testing"

	"github.com/taskcluster/tc-tui/resource"
)

func TestDetailViewSetDataResetsScrollToTop(t *testing.T) {
	d := NewDetailView()
	d.SetRect(0, 0, 20, 5)
	d.SetData(resource.Detail{Body: strings.Repeat("line\n", 50)})
	d.ScrollTo(10, 0)

	d.SetData(resource.Detail{Body: strings.Repeat("line\n", 50)})

	row, _ := d.GetScrollOffset()
	if row != 0 {
		t.Fatalf("expected SetData to reset scroll to top, got row %d", row)
	}
}

func TestDetailViewUpdateDataPreservesScroll(t *testing.T) {
	d := NewDetailView()
	d.SetRect(0, 0, 20, 5)
	d.SetData(resource.Detail{Body: strings.Repeat("line\n", 50)})
	d.ScrollTo(10, 0)

	d.UpdateData(resource.Detail{Body: strings.Repeat("line\n", 50)})

	row, _ := d.GetScrollOffset()
	if row != 10 {
		t.Fatalf("expected UpdateData to preserve scroll, got row %d, want 10", row)
	}
}

func TestDetailViewSetFilterQueryKeepsOnlyMatchingLines(t *testing.T) {
	d := NewDetailView()
	d.SetData(resource.Detail{Body: "alpha\nbeta\ngamma\n"})

	d.SetFilterQuery("eta")

	got := d.GetText(true)
	if got != "beta" {
		t.Fatalf("unexpected filtered text: %q", got)
	}
	if d.FilterQuery() != "eta" {
		t.Fatalf("expected FilterQuery to report %q, got %q", "eta", d.FilterQuery())
	}
}

func TestDetailViewSetFilterQueryIsCaseInsensitive(t *testing.T) {
	d := NewDetailView()
	d.SetData(resource.Detail{Body: "Completed\nFailed\n"})

	d.SetFilterQuery("completed")

	if got := d.GetText(true); got != "Completed" {
		t.Fatalf("unexpected filtered text: %q", got)
	}
}

func TestDetailViewSetFilterQueryMatchesVisibleTextNotColorTags(t *testing.T) {
	d := NewDetailView()
	d.SetData(resource.Detail{Body: "[green]Completed[white]\n[red]Failed[white]\n"})

	// "reen" only appears inside the "[green]" tag markup, never in the
	// visible text of either line — matching raw markup would wrongly keep
	// the first line.
	d.SetFilterQuery("reen")

	if got := d.GetText(true); got != "" {
		t.Fatalf("expected no lines to match a query that only appears in tag markup, got %q", got)
	}
}

func TestDetailViewSetFilterQueryEmptyRestoresAllLines(t *testing.T) {
	d := NewDetailView()
	d.SetData(resource.Detail{Body: "alpha\nbeta\ngamma\n"})
	d.SetFilterQuery("beta")

	d.SetFilterQuery("")

	if got := d.GetText(true); got != "alpha\nbeta\ngamma\n" {
		t.Fatalf("expected clearing the filter to restore every line, got %q", got)
	}
}

func TestDetailViewUpdateDataReappliesActiveFilter(t *testing.T) {
	d := NewDetailView()
	d.SetData(resource.Detail{Body: "alpha\nbeta\n"})
	d.SetFilterQuery("beta")

	d.UpdateData(resource.Detail{Body: "alpha\nbeta\ngamma\n"})

	if got := d.GetText(true); got != "beta" {
		t.Fatalf("expected the refreshed body to stay filtered, got %q", got)
	}
	if d.FilterQuery() != "beta" {
		t.Fatalf("expected FilterQuery to survive UpdateData, got %q", d.FilterQuery())
	}
}

func TestDetailViewSetDataResetsFilter(t *testing.T) {
	d := NewDetailView()
	d.SetData(resource.Detail{Body: "alpha\nbeta\n"})
	d.SetFilterQuery("beta")

	d.SetData(resource.Detail{Body: "alpha\nbeta\n"})

	if d.FilterQuery() != "" {
		t.Fatalf("expected navigating to a new detail to clear the filter, got %q", d.FilterQuery())
	}
	if got := d.GetText(true); got != "alpha\nbeta\n" {
		t.Fatalf("expected the new body unfiltered, got %q", got)
	}
}

func TestDetailViewStreamFiltersLinesLive(t *testing.T) {
	d := NewDetailView()
	d.SetRect(0, 0, 40, 5)
	d.StartStream()
	d.SetFilterQuery("err")

	d.AppendStream("info: starting\n")
	d.AppendStream("err: boom\n")
	d.AppendStream("info: done\n")

	if got := d.GetText(true); got != "err: boom\n" {
		t.Fatalf("expected only the matching streamed line to be visible, got %q", got)
	}
}

func TestDetailViewStreamFilterChangeRebuildsFromRetainedLines(t *testing.T) {
	d := NewDetailView()
	d.SetRect(0, 0, 40, 5)
	d.StartStream()

	d.AppendStream("info: starting\n")
	d.AppendStream("err: boom\n")
	d.AppendStream("info: done\n")

	d.SetFilterQuery("err")
	if got := d.GetText(true); got != "err: boom\n" {
		t.Fatalf("expected the filter to hide lines appended before it was set, got %q", got)
	}

	d.SetFilterQuery("")
	if got := d.GetText(true); got != "info: starting\nerr: boom\ninfo: done\n" {
		t.Fatalf("expected clearing the filter to restore every retained line, got %q", got)
	}
}

func TestDetailViewStreamBannerBypassesFilter(t *testing.T) {
	d := NewDetailView()
	d.SetRect(0, 0, 40, 5)
	d.StartStream()
	d.SetFilterQuery("err")
	d.AppendStream("info: nothing matches\n")

	d.AppendBanner("(live stream ended)")

	got := d.GetText(true)
	if !strings.Contains(got, "(live stream ended)") {
		t.Fatalf("expected the banner to bypass the active filter, got %q", got)
	}
}

func TestDetailViewStreamLinesCappedAtStreamViewMaxLines(t *testing.T) {
	d := NewDetailView()
	d.SetRect(0, 0, 40, 5)
	d.StartStream()

	for i := 0; i < streamViewMaxLines+10; i++ {
		d.AppendStream("line\n")
	}

	if len(d.streamLines) != streamViewMaxLines {
		t.Fatalf("expected retained lines capped at %d, got %d", streamViewMaxLines, len(d.streamLines))
	}
}
