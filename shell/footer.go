package shell

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/taskcluster/tc-tui/resource"
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
	hint := " [yellow]q[white] quit   [yellow]:[white] command   [yellow]/[white] filter   [yellow]Esc[white] back/quit   [yellow]?[white] help"
	if s.hasFacets() {
		hint += "   [yellow]Tab[white]/[yellow]Shift+Tab[white] switch state"
	}
	for _, action := range s.currentDetailActions {
		hint += fmt.Sprintf("   [yellow]<%c>[white] %s", action.Key, action.Label)
	}

	s.footerHint.SetText(hint)
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
// DirectLookup resource reached with no id (e.g. bare `:task`), rather than
// erroring or redirecting — there's no browsable list to redirect to.
func (s *Shell) openIDPrompt(res resource.DirectLookup) {
	s.footerMode = footerPrompt
	s.pendingLookup = res
	s.footerInput.SetLabel(fmt.Sprintf("[yellow]%s:[white] ", res.IDPromptLabel())).SetText("")
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
			s.closeFooterInput()
		case footerPrompt:
			id := strings.TrimSpace(s.footerInput.GetText())
			if id == "" {
				return // keep the prompt open; nothing to look up yet
			}
			res := s.pendingLookup
			s.pendingLookup = nil
			s.closeFooterInput()
			s.switchToDetail(res, id)
		}
	case tcell.KeyEscape:
		if s.footerMode == footerFilter {
			s.filterQuery = ""
			s.refreshTable()
		}
		s.pendingLookup = nil
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
