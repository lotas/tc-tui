package shell

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"

	"github.com/taskcluster/tc-tui/resource"
)

// HelpView renders the help overlay's body (global keys + registry-derived
// resource list). It has no state beyond what SetData is given each time
// help opens, since buildHelpText regenerates the full body from the live
// registry on every open.
type HelpView struct {
	*tview.TextView
}

func NewHelpView() *HelpView {
	h := &HelpView{TextView: tview.NewTextView()}
	h.SetDynamicColors(true).SetWordWrap(true)

	return h
}

func (h *HelpView) SetData(text string) {
	h.Clear().SetText(text).ScrollToBeginning()
}

// buildHelpText renders the full help body from the live registry, so it
// can never drift out of sync with which resources are actually registered.
func buildHelpText(registry *resource.Registry) string {
	var b strings.Builder

	b.WriteString("[green]Global keys[white]\n\n")
	b.WriteString("  [yellow]q[white]     quit from any view\n")
	b.WriteString("  [yellow]:[white]     open the command bar (switch resource, e.g. `:workerpools`, `:wp`, `:workers <poolId>`, `:help`)\n")
	b.WriteString("  [yellow]/[white]     filter the current list (list views only)\n")
	b.WriteString("  [yellow]Esc[white]   go back / quit at the root\n")
	b.WriteString("  [yellow]?[white]     toggle this help screen\n\n")

	b.WriteString("[green]Resources[white]\n\n")
	for _, name := range registry.Names() {
		res, ok := registry.Resolve(name)
		if !ok {
			continue
		}

		aliases := "none"
		if len(res.Aliases()) > 0 {
			aliases = strings.Join(res.Aliases(), ", ")
		}

		columns := make([]string, len(res.Columns()))
		for i, col := range res.Columns() {
			columns[i] = col.Title
		}

		b.WriteString(fmt.Sprintf(
			"  [yellow]%s[white] (aliases: %s)\n      %s\n      columns: %s\n",
			res.Name(), aliases, res.Description(), strings.Join(columns, ", "),
		))

		if scoped, isScoped := res.(resource.ScopedResource); isScoped {
			b.WriteString(fmt.Sprintf(
				"      requires a scope, e.g. `:%s <id>` — no scope redirects to `%s`\n",
				res.Name(), scoped.EmptyScopeResource(),
			))
		}

		b.WriteString("\n")
	}

	b.WriteString("[green]Context actions[white]\n\n")
	b.WriteString("  Some detail screens expose extra keys (e.g. <w> workers) shown in that screen's own footer hint bar when available.\n")

	return b.String()
}
