package shell

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"

	"github.com/taskcluster/tc-tui/resource"
)

// streamViewMaxLines bounds the TextView's buffer while a live stream is
// appending to it (see StartStream) — a long-running task's log grows
// without limit, and unlike a one-shot render there's no
// maxArtifactRenderBytes gate in front of the view. Oldest lines are
// dropped; 'o'/'s' remain the full-content escape hatches. streamLines is
// bounded the same way, so a '/' filter applied mid-stream (or after it
// ends) only ever has to search/rebuild from the same bounded window
// that's actually on screen, never an unbounded log.
const streamViewMaxLines = 10000

// DetailView renders a Resource.Describe(id) result. Any DetailActions the
// Detail carries are dispatched by Shell.globalInputCapture (shared with a
// List view's ScopeActions), not by DetailView itself.
type DetailView struct {
	*tview.TextView

	wrapEnabled bool

	// rawBody is the last SetData/UpdateData body, before any '/' filter is
	// applied — filtering always re-derives from this, never from a
	// previously-filtered rendering, so widening/clearing the query can
	// always recover lines it had hidden.
	rawBody string

	// filterQuery is the active '/' line filter — shared meaning for both a
	// one-shot body (rawBody) and a live stream (streamLines), but tracked
	// independently of the Shell's own list-filter state so filtering a list
	// can never leak into a detail view navigated to afterward, or vice
	// versa (list filters are already scoped per-resource for the same
	// reason).
	filterQuery string

	// streaming is set once StartStream has been called and never reset —
	// even after the stream itself ends, filtering should keep working
	// against the accumulated streamLines rather than switching back to the
	// (empty) rawBody path.
	streaming   bool
	streamLines []string
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
	d.rawBody = detail.Body
	d.filterQuery = ""
	d.streaming = false
	d.streamLines = nil
	d.SetMaxLines(0) // clear any StartStream bound — one-shot bodies are pre-capped
	d.Clear().SetText(detail.Body).ScrollToBeginning()
}

// UpdateData replaces the body like SetData, but preserves the current
// scroll position — used when refreshing the view already on screen (as
// opposed to navigating to a new one), so a periodic auto-refresh or the
// manual `r` key doesn't yank the reader back to the top. Any active '/'
// filter carries over and is reapplied to the refreshed body, matching how a
// list view's filter survives its own auto-refresh.
func (d *DetailView) UpdateData(detail resource.Detail) {
	row, col := d.GetScrollOffset()
	d.rawBody = detail.Body
	d.SetMaxLines(0)
	d.Clear().SetText(d.filteredBody())
	d.ScrollTo(row, col)
}

// FilterQuery returns the active '/' line filter, if any — used both to
// prefill the footer input when '/' reopens it and to drive the detail
// title's "(query)" suffix and border tint, mirroring the list view's own
// filter affordances.
func (d *DetailView) FilterQuery() string {
	return d.filterQuery
}

// SetFilterQuery re-renders the body keeping only lines whose visible text
// contains query (case-insensitive substring) — the '/' key's behavior on a
// detail page, consistent with how it narrows rows in a list view. Matching
// is against each line's STRIPPED (tag-free) text, not its raw markup, so a
// query can't accidentally match inside a `[green]`-style color tag. An
// empty query restores every line.
func (d *DetailView) SetFilterQuery(query string) {
	d.filterQuery = query
	if d.streaming {
		d.rebuildStreamView()
		return
	}
	d.Clear().SetText(d.filteredBody()).ScrollToBeginning()
}

func (d *DetailView) filteredBody() string {
	return strings.Join(filterLines(strings.Split(d.rawBody, "\n"), d.filterQuery), "\n")
}

// StartStream prepares the view for a live stream: empty content, a line
// bound on the buffer, and follow mode — tview's ScrollToEnd keeps the view
// pinned to the newest line as content is appended, detaching when the user
// scrolls up and re-attaching when they return to the bottom, i.e. `tail -f`
// semantics for free. Any filter left over from a previous view is dropped —
// this is a fresh log.
func (d *DetailView) StartStream() {
	d.filterQuery = ""
	d.streaming = true
	d.streamLines = nil
	d.Clear()
	d.SetMaxLines(streamViewMaxLines)
	d.ScrollToEnd()
}

// AppendStream appends already-rendered text to the view — content is
// expected to consist of whole, newline-terminated lines (guaranteed by the
// LiveStreamer's own line assembly), except possibly the very last call at
// stream end, which may flush a final unterminated line. Every line is
// recorded into streamLines (capped at streamViewMaxLines, oldest dropped)
// so a '/' filter applied mid-stream or after it ends has the full retained
// window to search, even lines already scrolled out of the visible buffer.
// While a filter is active, only matching lines are actually written to the
// view; scroll/follow state is otherwise untouched.
func (d *DetailView) AppendStream(text string) {
	lines := splitLines(text)
	d.streamLines = append(d.streamLines, lines...)
	if excess := len(d.streamLines) - streamViewMaxLines; excess > 0 {
		d.streamLines = d.streamLines[excess:]
	}

	if d.filterQuery == "" {
		fmt.Fprint(d.TextView, text)
		return
	}
	if kept := filterLines(lines, d.filterQuery); len(kept) > 0 {
		fmt.Fprint(d.TextView, strings.Join(kept, "\n")+"\n")
	}
}

// AppendBanner appends a status line (stream ended/failed/truncated) that
// bypasses filtering entirely — it's UI chrome, not log content, so it
// should never be silently hidden by a '/' query that happens not to match
// it, and it's deliberately not recorded into streamLines, so a filter
// change afterward won't reproduce or duplicate it.
func (d *DetailView) AppendBanner(text string) {
	fmt.Fprint(d.TextView, text)
}

// rebuildStreamView redraws the view from streamLines under the current
// filter and re-attaches to the tail — used whenever the '/' query changes
// while streaming (or after a stream has ended), since unlike AppendStream's
// incremental path, a query CHANGE can both re-reveal previously-hidden
// lines and hide previously-visible ones, so the whole buffer must be
// redrawn rather than patched.
func (d *DetailView) rebuildStreamView() {
	d.Clear()
	if kept := filterLines(d.streamLines, d.filterQuery); len(kept) > 0 {
		fmt.Fprint(d.TextView, strings.Join(kept, "\n")+"\n")
	}
	d.ScrollToEnd()
}

// filterLines returns the subset of lines whose stripped (tag-free) text
// contains query (case-insensitive substring), preserving order — shared by
// the one-shot body filter and the live-stream filter. An empty query
// returns lines unchanged.
func filterLines(lines []string, query string) []string {
	if query == "" {
		return lines
	}

	stripped := strings.Split(stripDisplayTags(strings.Join(lines, "\n")), "\n")
	needle := strings.ToLower(query)

	var kept []string
	for i, line := range lines {
		if i < len(stripped) && strings.Contains(strings.ToLower(stripped[i]), needle) {
			kept = append(kept, line)
		}
	}
	return kept
}

// splitLines splits already-assembled stream text into individual lines,
// tolerating both a trailing newline (the common case) and none (the final
// flush at stream end may hand back an unterminated last line). Empty text
// yields no lines.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimRight(text, "\n"), "\n")
}

// stripDisplayTags strips tview's dynamic-color tags from text, using a
// scratch TextView (configured exactly like the real one) rather than
// reimplementing tview's own tag grammar — used so a '/' filter query is
// matched against what's actually on screen (e.g. "completed", not
// "[green]completed[white]"), not the raw markup.
func stripDisplayTags(text string) string {
	scratch := tview.NewTextView().SetDynamicColors(true)
	scratch.SetText(text)
	return scratch.GetText(true)
}
