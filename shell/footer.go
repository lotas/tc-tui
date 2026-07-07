package shell

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (s *Shell) initFooter() {
	s.footerHint = tview.NewTextView().SetDynamicColors(true)
	s.footerInput = tview.NewInputField().SetFieldWidth(0)
	s.footerInput.SetDoneFunc(s.handleFooterInputDone)
	s.footerInput.SetChangedFunc(s.handleFooterInputChanged)

	s.footer = tview.NewPages().
		AddPage(pageFooterHints, s.footerHint, true, true).
		AddPage(pageFooterInput, s.footerInput, true, false)

	s.renderFooterHints()
}

func (s *Shell) renderFooterHints() {
	s.footerHint.SetText(
		" [yellow]:[white] command   [yellow]/[white] filter   [yellow]Esc[white] back/quit",
	)
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

func (s *Shell) closeFooterInput() {
	s.footerMode = footerHints
	s.footer.SwitchToPage(pageFooterHints)
	s.renderFooterHints()
	s.app.SetFocus(s.activeContent)
}

func (s *Shell) handleFooterInputChanged(text string) {
	if s.footerMode != footerFilter {
		return
	}

	s.filterQuery = text
	s.table.SetData(s.currentColumns, FilterRows(s.lastRows, s.filterQuery))
}

func (s *Shell) handleFooterInputDone(key tcell.Key) {
	switch key {
	case tcell.KeyEnter:
		switch s.footerMode {
		case footerCommand:
			name, scope := splitCommand(s.footerInput.GetText())
			s.closeFooterInput()
			s.switchResource(name, scope)
		case footerFilter:
			s.filterQuery = s.footerInput.GetText()
			s.closeFooterInput()
		}
	case tcell.KeyEscape:
		if s.footerMode == footerFilter {
			s.filterQuery = ""
			s.table.SetData(s.currentColumns, s.lastRows)
		}
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
