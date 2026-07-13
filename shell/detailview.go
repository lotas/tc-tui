package shell

import (
	"github.com/rivo/tview"

	"github.com/taskcluster/tc-tui/resource"
)

// DetailView renders a Resource.Describe(id) result. Any DetailActions the
// Detail carries are dispatched by Shell.globalInputCapture (shared with a
// List view's ScopeActions), not by DetailView itself.
type DetailView struct {
	*tview.TextView
}

func NewDetailView() *DetailView {
	d := &DetailView{
		TextView: tview.NewTextView(),
	}
	d.SetDynamicColors(true).SetWordWrap(true)

	return d
}

func (d *DetailView) SetData(detail resource.Detail) {
	d.Clear().SetText(detail.Body).ScrollToBeginning()
}
