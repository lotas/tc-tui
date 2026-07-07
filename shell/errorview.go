package shell

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ErrorView is the full-screen error view shown when a List()/Describe()
// call fails on initial load. It offers a retry action bound to the same
// refresh key.
type ErrorView struct {
	*tview.TextView
	onRetry func()
}

func NewErrorView() *ErrorView {
	e := &ErrorView{
		TextView: tview.NewTextView(),
	}
	e.SetDynamicColors(true).SetWordWrap(true)
	e.SetInputCapture(e.handleKey)

	return e
}

func (e *ErrorView) SetOnRetry(fn func()) {
	e.onRetry = fn
}

func (e *ErrorView) SetError(title string, err error) {
	e.Clear().SetText(fmt.Sprintf(
		"[red]Error loading %s:[white]\n\n%s\n\n[yellow]<r>[white] retry",
		title, err,
	))
}

func (e *ErrorView) handleKey(event *tcell.EventKey) *tcell.EventKey {
	if event.Rune() == 'r' && e.onRetry != nil {
		e.onRetry()
		return nil
	}

	return event
}
