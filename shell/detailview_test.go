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
