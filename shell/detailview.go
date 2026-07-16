package shell

import (
	"fmt"

	"github.com/rivo/tview"

	"github.com/taskcluster/tc-tui/resource"
)

// streamViewMaxLines bounds the TextView's buffer while a live stream is
// appending to it (see StartStream) — a long-running task's log grows
// without limit, and unlike a one-shot render there's no
// maxArtifactRenderBytes gate in front of the view. Oldest lines are
// dropped; 'o'/'s' remain the full-content escape hatches.
const streamViewMaxLines = 10000

// DetailView renders a Resource.Describe(id) result. Any DetailActions the
// Detail carries are dispatched by Shell.globalInputCapture (shared with a
// List view's ScopeActions), not by DetailView itself.
type DetailView struct {
	*tview.TextView

	wrapEnabled bool
}

func NewDetailView() *DetailView {
	d := &DetailView{
		TextView:    tview.NewTextView(),
		wrapEnabled: true,
	}
	d.SetDynamicColors(true).SetWordWrap(true)

	return d
}

// WrapEnabled reports whether the body wraps at the view's width (the
// default) or, once toggled off via the 'x' key, runs lines out unbroken —
// reachable with Left/Right/h/l, tview.TextView's built-in horizontal scroll.
func (d *DetailView) WrapEnabled() bool {
	return d.wrapEnabled
}

// SetWrapEnabled flips between word-wrapping the body (default) and leaving
// long lines unbroken so they can be scrolled horizontally instead — useful
// for content with long unbroken lines (URLs, wide JSON) that word-wrap would
// otherwise force onto multiple lines.
func (d *DetailView) SetWrapEnabled(enabled bool) {
	d.wrapEnabled = enabled
	d.SetWrap(enabled)
}

func (d *DetailView) SetData(detail resource.Detail) {
	d.SetMaxLines(0) // clear any StartStream bound — one-shot bodies are pre-capped
	d.Clear().SetText(detail.Body).ScrollToBeginning()
}

// UpdateData replaces the body like SetData, but preserves the current
// scroll position — used when refreshing the view already on screen (as
// opposed to navigating to a new one), so a periodic auto-refresh or the
// manual `r` key doesn't yank the reader back to the top.
func (d *DetailView) UpdateData(detail resource.Detail) {
	row, col := d.GetScrollOffset()
	d.SetMaxLines(0)
	d.Clear().SetText(detail.Body)
	d.ScrollTo(row, col)
}

// StartStream prepares the view for a live stream: empty content, a line
// bound on the buffer, and follow mode — tview's ScrollToEnd keeps the view
// pinned to the newest line as content is appended, detaching when the user
// scrolls up and re-attaching when they return to the bottom, i.e. `tail -f`
// semantics for free.
func (d *DetailView) StartStream() {
	d.Clear()
	d.SetMaxLines(streamViewMaxLines)
	d.ScrollToEnd()
}

// AppendStream appends already-rendered text to the view without touching
// the scroll state — follow/detached behavior stays wherever StartStream and
// the user's own scrolling left it.
func (d *DetailView) AppendStream(text string) {
	fmt.Fprint(d.TextView, text)
}
