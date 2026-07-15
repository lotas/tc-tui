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
	onSelect func(row resource.Row)

	// lastSelectedID remembers the Row.ID of whichever row the cursor is
	// currently on, updated via SetSelectionChangedFunc (which fires on
	// any cursor movement, not just Enter). SetData uses it to restore the
	// selection after repopulating the table, e.g. after a background
	// auto-refresh or coming back from a Detail view via Esc.
	lastSelectedID string

	// expandColumns disables every column's Width cap (toggled by the 'x'
	// key) so columns instead size to their widest cell's natural content —
	// columns that no longer fit the terminal are scrolled into view with
	// the Left/Right arrow keys (tview.Table's built-in columnOffset
	// scrolling, already active since this table has column selection off).
	expandColumns bool
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

		r, ok := cell.GetReference().(resource.Row)
		if !ok {
			return
		}

		t.onSelect(r)
	})
	t.SetSelectionChangedFunc(func(row, column int) {
		if row == 0 {
			return
		}

		cell := t.GetCell(row, 0)
		if cell == nil {
			return
		}

		r, ok := cell.GetReference().(resource.Row)
		if !ok {
			return
		}

		t.lastSelectedID = r.ID
	})

	return t
}

func (t *TableView) SetOnSelect(fn func(row resource.Row)) {
	t.onSelect = fn
}

// SetExpandColumns toggles whether every column's Width cap is honored.
// Takes effect on the next SetData call — callers must re-render (e.g. via
// Shell.refreshTable) to see the change.
func (t *TableView) SetExpandColumns(expand bool) {
	t.expandColumns = expand
}

// ExpandColumns reports whether column truncation is currently disabled.
func (t *TableView) ExpandColumns() bool {
	return t.expandColumns
}

// ResetSelection forgets the remembered selection, so the next SetData call
// falls back to selecting the top row instead of trying to restore the
// cursor to wherever it was — used when a sort change reorders the list
// enough that "the same row" no longer means the same thing to the user.
func (t *TableView) ResetSelection() {
	t.lastSelectedID = ""
}

// columnGap is appended to every column's text except the last, on top of
// tview's own single-space column separator — otherwise adjacent columns of
// similar width (e.g. two 12-char columns side by side) read as one run of
// text with no visible break.
const columnGap = " "

// SetData replaces the table's header and rows. Each row's first cell holds
// a reference to the whole resource.Row (not just its ID), so selection can
// read row.NavTarget as well as row.ID without a separate index-to-row
// lookup. If sort.Direction is not SortNone, a ▲/▼ indicator is appended to
// that column's header text.
func (t *TableView) SetData(columns []resource.Column, rows []resource.Row, sort SortState) {
	t.Clear()

	lastCol := len(columns) - 1

	for col, column := range columns {
		title := column.Title
		if sort.Direction != SortNone && col == sort.Column {
			if sort.Direction == SortAsc {
				title += " ▲"
			} else {
				title += " ▼"
			}
		}
		if col != lastCol {
			title += columnGap
		}

		cell := tview.NewTableCell(title).
			SetTextColor(tview.Styles.SecondaryTextColor).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		if column.Width == 0 || column.Expand || t.expandColumns {
			cell.SetExpansion(1)
		}
		if column.Width != 0 && !t.expandColumns {
			cell.SetMaxWidth(columnMaxWidth(column.Width, col, lastCol))
		}
		t.SetCell(0, col, cell)
	}

	for r, row := range rows {
		for col, value := range row.Cells {
			if col != lastCol {
				value += columnGap
			}

			cell := tview.NewTableCell(value)
			if col == 0 {
				cell.SetReference(row)
			}
			if columns[col].Width == 0 || columns[col].Expand || t.expandColumns {
				cell.SetExpansion(1)
			}
			if columns[col].Width != 0 && !t.expandColumns {
				cell.SetMaxWidth(columnMaxWidth(columns[col].Width, col, lastCol))
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

// columnMaxWidth returns width's cap, plus one extra character for the
// columnGap suffix on every column but the last — otherwise a value that
// exactly fills width would have its trailing gap truncated away.
func columnMaxWidth(width, col, lastCol int) int {
	if col != lastCol {
		return width + 1
	}
	return width
}
