package shell

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/taskcluster/tc-tui/resource"
)

// TableView is a generic multi-column table (like k9s's NAME/STATUS/AGE
// tables). It renders whatever Resource.Columns()/List() gives it.
type TableView struct {
	*tview.Table
	onSelect func(id string)

	// lastSelectedID remembers the Row.ID of whichever row the cursor is
	// currently on, updated via SetSelectionChangedFunc (which fires on
	// any cursor movement, not just Enter). SetData uses it to restore the
	// selection after repopulating the table, e.g. after a background
	// auto-refresh or coming back from a Detail view via Esc.
	lastSelectedID string
}

func NewTableView() *TableView {
	t := &TableView{
		Table: tview.NewTable(),
	}
	t.SetSelectable(true, false)
	t.SetFixed(1, 0)
	t.SetSelectedFunc(func(row, column int) {
		if row == 0 || t.onSelect == nil {
			return
		}

		cell := t.GetCell(row, 0)
		if cell == nil {
			return
		}

		id, ok := cell.GetReference().(string)
		if !ok {
			return
		}

		t.onSelect(id)
	})
	t.SetSelectionChangedFunc(func(row, column int) {
		if row == 0 {
			return
		}

		cell := t.GetCell(row, 0)
		if cell == nil {
			return
		}

		id, ok := cell.GetReference().(string)
		if !ok {
			return
		}

		t.lastSelectedID = id
	})

	return t
}

func (t *TableView) SetOnSelect(fn func(id string)) {
	t.onSelect = fn
}

// ResetSelection forgets the remembered selection, so the next SetData call
// falls back to selecting the top row instead of trying to restore the
// cursor to wherever it was — used when a sort change reorders the list
// enough that "the same row" no longer means the same thing to the user.
func (t *TableView) ResetSelection() {
	t.lastSelectedID = ""
}

// SetData replaces the table's header and rows. Row IDs are stashed on
// each row's first cell via SetReference so selection can look them up
// without a separate index-to-ID slice. If sort.Direction is not SortNone,
// a ▲/▼ indicator is appended to that column's header text.
func (t *TableView) SetData(columns []resource.Column, rows []resource.Row, sort SortState) {
	t.Clear()

	for col, column := range columns {
		title := column.Title
		if sort.Direction != SortNone && col == sort.Column {
			if sort.Direction == SortAsc {
				title += " ▲"
			} else {
				title += " ▼"
			}
		}

		cell := tview.NewTableCell(title).
			SetTextColor(tview.Styles.SecondaryTextColor).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		if column.Width == 0 {
			cell.SetExpansion(1)
		} else {
			cell.SetMaxWidth(column.Width)
		}
		t.SetCell(0, col, cell)
	}

	for r, row := range rows {
		for col, value := range row.Cells {
			cell := tview.NewTableCell(value)
			if col == 0 {
				cell.SetReference(row.ID)
			}
			if columns[col].Width == 0 {
				cell.SetExpansion(1)
			} else {
				cell.SetMaxWidth(columns[col].Width)
			}
			t.SetCell(r+1, col, cell)
		}
	}

	// Try to restore the previously selected row (by ID) so that a
	// background auto-refresh or navigating back from a Detail view
	// doesn't silently reset the cursor to the top of the list. If the
	// remembered ID isn't present in the new rows (e.g. the user switched
	// to a different resource entirely), fall back to the old behavior of
	// selecting row 1 (or row 0 if the list is empty) and scrolling to the
	// top, since there's no meaningful previous selection to restore.
	restored := false
	if t.lastSelectedID != "" {
		for r, row := range rows {
			if row.ID == t.lastSelectedID {
				t.Select(r+1, 0)
				restored = true
				break
			}
		}
	}

	if !restored {
		if len(rows) > 0 {
			t.Select(1, 0)
		} else {
			t.Select(0, 0)
		}
		t.ScrollToBeginning()
	}
}
