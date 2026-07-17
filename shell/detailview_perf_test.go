package shell

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/taskcluster/tc-tui/resource"
)

// drawBudget bounds how long a single DetailView.Draw may take for a large
// body. Deliberately generous relative to what a healthy render costs (~1ms,
// independent of size — the view only processes visible lines): this guards
// against a return to O(n^2) behavior, not against ordinary slowness, so it
// can't fail on a merely slow machine. The regression it exists for missed
// this budget by four orders of magnitude.
const drawBudget = 2 * time.Second

// largeLogBody builds a body shaped like the artifact that triggered the
// freeze: a plain-text worker log, which (unlike markdown/JSON) has no
// content-type-driven size cap in front of the view and so arrives whole.
func largeLogBody(lines int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&b, "[taskcluster 2026-07-16T12:%02d:%02d.123Z] step %d of the build is doing something useful here\n",
			i%60, i%60, i)
	}
	return b.String()
}

// A large artifact body must not stall the UI goroutine. Draw runs on that
// goroutine, so any per-line cost that scales with the whole buffer (rather
// than the visible window) freezes the entire app — no redraw, and no key
// handling, so not even 'q' or Ctrl-C get processed. tview's TextView used to
// re-scan its entire accumulated index once per buffer line while
// word-wrapping, making Draw quadratic: ~1s for 1k lines, ~90s for 10k, and
// effectively forever for a 20MiB log, all at 100% CPU with the screen frozen
// on the previously drawn frame.
func TestDetailViewDrawIsNotQuadraticForLargeBodies(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	screen.SetSize(200, 50)

	d := NewDetailView()
	d.SetRect(0, 0, 200, 50)
	d.SetData(resource.Detail{Body: largeLogBody(10000)})

	done := make(chan time.Duration, 1)
	go func() {
		start := time.Now()
		d.Draw(screen)
		done <- time.Since(start)
	}()

	select {
	case elapsed := <-done:
		if elapsed > drawBudget {
			t.Fatalf("Draw of a 10k-line body took %v, over the %v budget — the UI goroutine stalls this long", elapsed, drawBudget)
		}
	case <-time.After(drawBudget):
		// Reported without waiting it out: the quadratic version takes ~90s
		// here, and grows with the square of the body size.
		t.Fatalf("Draw of a 10k-line body did not finish within %v — the UI goroutine is stalled", drawBudget)
	}
}

// TestDetailViewSetFilterQueryIsNotQuadraticForLargeBodies guards the '/'
// filter's own per-keystroke cost (stripDisplayTags + line filtering), which
// runs entirely off tview's Draw path but over the same class of large body
// — a linear-per-keystroke cost is fine (comparable to a single Draw), a
// quadratic one would reintroduce the freeze this budget exists to catch,
// just triggered by typing instead of rendering.
func TestDetailViewSetFilterQueryIsNotQuadraticForLargeBodies(t *testing.T) {
	d := NewDetailView()
	d.SetRect(0, 0, 200, 50)
	d.SetData(resource.Detail{Body: largeLogBody(10000)})

	done := make(chan time.Duration, 1)
	go func() {
		start := time.Now()
		d.SetFilterQuery("step 9999")
		done <- time.Since(start)
	}()

	select {
	case elapsed := <-done:
		if elapsed > drawBudget {
			t.Fatalf("SetFilterQuery over a 10k-line body took %v, over the %v budget", elapsed, drawBudget)
		}
	case <-time.After(drawBudget):
		t.Fatalf("SetFilterQuery over a 10k-line body did not finish within %v", drawBudget)
	}
}
