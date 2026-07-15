package shell

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (s *Shell) initFooter() {
	s.footerBreadcrumb = tview.NewTextView().SetDynamicColors(true)
	s.footerInput = tview.NewInputField().SetFieldWidth(0)
	s.footerInput.SetDoneFunc(s.handleFooterInputDone)
	s.footerInput.SetChangedFunc(s.handleFooterInputChanged)

	s.footer = tview.NewPages().
		AddPage(pageFooterBreadcrumb, s.footerBreadcrumb, true, true).
		AddPage(pageFooterInput, s.footerInput, true, false)

	s.renderHeaderHints()
	s.renderBreadcrumbs()
}

// hintColumns caps how many hints share a line in the header's center
// column, laid out as a left-aligned grid (k9s-style) rather than relying
// on the terminal's own wrapping, which would break a hint mid-string (e.g.
// splitting "Esc" from "back/quit") on a narrow terminal.
const hintColumns = 3

// headerHintGap is the minimum number of spaces separating one hint column
// from the next.
const headerHintGap = 3

// hint pairs a hint's plain-text form (used to measure column width, since
// tview's [color] region tags aren't rendered) with its colored form
// (what's actually shown).
type hint struct {
	plain   string
	colored string
}

// renderHeaderHints rebuilds the header's center hint column: global keys,
// the filter hint (list views only — `/` is a no-op on a detail view), the
// facet-switch hint when the current list has facets, and any
// per-detail-action keys the current detail screen exposes. Hints are laid
// out as a left-aligned grid of hintColumns columns, each padded to the
// width of the longest hint, rather than centered free text.
func (s *Shell) renderHeaderHints() {
	hints := []hint{
		{"q quit", "[yellow]q[white] quit"},
		{": command", "[yellow]:[white] command"},
		{"r refresh", "[yellow]r[white] refresh"},
		{"Esc back", "[yellow]Esc[white] back"},
		{"? help", "[yellow]?[white] help"},
	}
	if top, ok := s.stack.Top(); !ok || top.Kind == ListKind {
		hints = append(hints, hint{"/ filter", "[yellow]/[white] filter"})
		hints = append(hints, hint{"x truncate", "[yellow]x[white] truncate"})
	}
	if s.hasFacets() {
		hints = append(hints, hint{"Tab/Shift+Tab switch state", "[yellow]Tab[white]/[yellow]Shift+Tab[white] switch state"})
	}
	if url, ok := s.currentWebURL(); ok && url != "" {
		hints = append(hints, hint{"o open in browser", "[yellow]o[white] open in browser"})
	}
	for _, action := range s.currentDetailActions {
		hints = append(hints, hint{
			plain:   fmt.Sprintf("%c %s", action.Key, action.Label),
			colored: fmt.Sprintf("[yellow]%c[white] %s", action.Key, action.Label),
		})
	}

	// Each column is only as wide as the widest hint that actually falls in
	// it, not the widest hint overall — otherwise one long hint (e.g. the
	// facet-switch hint) in one column would pad every column to its width.
	colWidth := make([]int, hintColumns)
	for i, h := range hints {
		col := i % hintColumns
		if len(h.plain) > colWidth[col] {
			colWidth[col] = len(h.plain)
		}
	}

	var b strings.Builder
	for i, h := range hints {
		col := i % hintColumns
		if col == 0 {
			b.WriteString(" ")
		}
		b.WriteString(h.colored)

		lastInRow := col == hintColumns-1 || i == len(hints)-1
		if !lastInRow {
			b.WriteString(strings.Repeat(" ", colWidth[col]-len(h.plain)+headerHintGap))
		} else if i != len(hints)-1 {
			b.WriteString("\n")
		}
	}

	s.headerHint.SetText(b.String())

	// The header row's height is fixed at grid-construction time (see
	// s.root.SetRows in init) to fit headerLeft's constant 3 lines. A detail
	// view can expose enough DetailActions to need more hint rows than that
	// — grow the header row to fit rather than silently clipping hints below
	// the visible area.
	//
	// s.root doesn't exist yet on the very first call, made from initFooter
	// before the grid is constructed — nothing to resize yet in that case.
	if s.root != nil {
		s.root.SetRows(headerRowsNeeded(len(hints)), 0, 1)
	}
}

// headerRowsNeeded returns how tall the header grid row must be to fit
// hintCount hints laid out hintColumns-wide, never shrinking below 3 —
// headerLeft always renders exactly 3 lines (root URL, version, client ID)
// regardless of how few hints there are.
func headerRowsNeeded(hintCount int) int {
	rows := (hintCount + hintColumns - 1) / hintColumns
	if rows < 3 {
		return 3
	}
	return rows
}

// renderBreadcrumbs rebuilds the footer's navigation trail from the current
// view stack, e.g. "workerpools › gecko-3/b-linux (workers)".
func (s *Shell) renderBreadcrumbs() {
	views := s.stack.views
	parts := make([]string, len(views))
	for i, v := range views {
		switch v.Kind {
		case DetailKind:
			parts[i] = fmt.Sprintf("%s:%s", v.ResourceName, v.SelectedID)
		default:
			if v.Scope != "" {
				parts[i] = fmt.Sprintf("%s (%s)", v.ResourceName, v.Scope)
			} else {
				parts[i] = v.ResourceName
			}
		}
	}

	s.footerBreadcrumb.SetText(" " + strings.Join(parts, " › "))
}

func (s *Shell) openCommandBar() {
	s.footerMode = footerCommand
	s.footerInput.SetLabel("[yellow]:[white] ").SetText("")
	s.footer.SwitchToPage(pageFooterInput)
	s.app.SetFocus(s.footerInput)
}

func (s *Shell) openFilter() {
	s.footerMode = footerFilter
	s.footerInput.SetLabel("[yellow]/[white] ").SetText(s.filterQuery)
	s.footer.SwitchToPage(pageFooterInput)
	s.app.SetFocus(s.footerInput)
}

// openIDPrompt switches the footer to an inline id-entry field for a
// DirectLookup or DirectScopedResource reached with no id (e.g. bare
// `:task`), rather than erroring or redirecting — there's no browsable list
// to redirect to. commit is called with the entered id once Enter is
// pressed; label is what's shown before the input field, e.g. "task id".
func (s *Shell) openIDPrompt(label string, commit func(id string)) {
	s.footerMode = footerPrompt
	s.pendingLookupCommit = commit
	s.footerInput.SetLabel(fmt.Sprintf("[yellow]%s:[white] ", label)).SetText("")
	s.footer.SwitchToPage(pageFooterInput)
	s.app.SetFocus(s.footerInput)
}

func (s *Shell) closeFooterInput() {
	s.footerMode = footerIdle
	s.footer.SwitchToPage(pageFooterBreadcrumb)
	s.app.SetFocus(s.activeContent)
}

func (s *Shell) handleFooterInputChanged(text string) {
	if s.footerMode != footerFilter {
		return
	}

	s.filterQuery = text
	s.refreshTable()
}

func (s *Shell) handleFooterInputDone(key tcell.Key) {
	switch key {
	case tcell.KeyEnter:
		switch s.footerMode {
		case footerCommand:
			name, scope := splitCommand(s.footerInput.GetText())
			s.closeFooterInput()
			if strings.EqualFold(name, "help") {
				s.openHelp()
			} else {
				s.switchResource(name, scope)
			}
		case footerFilter:
			s.filterQuery = s.footerInput.GetText()
			s.filterByResource[s.currentListResource] = s.filterQuery
			s.closeFooterInput()
		case footerPrompt:
			id := strings.TrimSpace(s.footerInput.GetText())
			if id == "" {
				return // keep the prompt open; nothing to look up yet
			}
			commit := s.pendingLookupCommit
			s.pendingLookupCommit = nil
			s.closeFooterInput()
			commit(id)
		}
	case tcell.KeyEscape:
		if s.footerMode == footerFilter {
			s.filterQuery = ""
			s.filterByResource[s.currentListResource] = ""
			s.refreshTable()
		}
		s.pendingLookupCommit = nil
		s.closeFooterInput()
	}
}

// splitCommand splits a command-bar input into a resource name and an
// optional scope argument: the remaining whitespace-separated fields,
// re-joined with a single space each (surrounding/repeated whitespace is
// normalized, not preserved literally).
func splitCommand(input string) (name, scope string) {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return "", ""
	}

	return fields[0], strings.Join(fields[1:], " ")
}
