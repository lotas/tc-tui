package shell

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/taskcluster/tc-tui/resource"
)

// DetailView renders a Resource.Describe(id) result, including footer-hint
// keybindings for any DetailActions (cross-resource jumps). The shell
// executes the declared NavTarget on keypress without knowing what it means.
type DetailView struct {
	*tview.TextView
	actions  []resource.DetailAction
	onAction func(target resource.NavTarget)
}

func NewDetailView() *DetailView {
	d := &DetailView{
		TextView: tview.NewTextView(),
	}
	d.SetDynamicColors(true).SetWordWrap(true)
	d.SetInputCapture(d.handleKey)

	return d
}

func (d *DetailView) SetOnAction(fn func(target resource.NavTarget)) {
	d.onAction = fn
}

func (d *DetailView) SetData(detail resource.Detail) {
	d.actions = detail.Actions

	body := detail.Body
	if len(detail.Actions) > 0 {
		hints := make([]string, 0, len(detail.Actions))
		for _, action := range detail.Actions {
			hints = append(hints, fmt.Sprintf("[yellow]<%c>[white] %s", action.Key, action.Label))
		}
		body += "\n\n" + strings.Join(hints, "   ")
	}

	d.Clear().SetText(body).ScrollToBeginning()
}

func (d *DetailView) handleKey(event *tcell.EventKey) *tcell.EventKey {
	if d.onAction == nil || event.Key() != tcell.KeyRune {
		return event
	}

	for _, action := range d.actions {
		if action.Key == event.Rune() {
			d.onAction(action.Target)
			return nil
		}
	}

	return event
}
