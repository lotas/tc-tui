package shell

import (
	"sort"
	"strconv"
	"strings"

	"github.com/taskcluster/tc-tui/resource"
)

// SortDirection is the direction a list view is currently sorted in.
type SortDirection int

const (
	// SortNone means "unsorted / API order" — the zero value, so a
	// zero-value SortState is a no-op for SortRows.
	SortNone SortDirection = iota
	SortAsc
	SortDesc
)

// SortState identifies the active sort column/direction for a list view.
type SortState struct {
	Column    int
	Direction SortDirection
}

// SortRows returns a stably-sorted copy of rows according to state. If
// state.Direction is SortNone, or state.Column doesn't index into rows'
// cells, rows is returned unchanged.
func SortRows(rows []resource.Row, state SortState) []resource.Row {
	if state.Direction == SortNone || state.Column < 0 {
		return rows
	}
	if len(rows) == 0 || state.Column >= len(rows[0].Cells) {
		return rows
	}

	sorted := make([]resource.Row, len(rows))
	copy(sorted, rows)

	numeric := isNumericColumn(sorted, state.Column)
	sort.SliceStable(sorted, func(i, j int) bool {
		a := strings.TrimSpace(sorted[i].Cells[state.Column])
		b := strings.TrimSpace(sorted[j].Cells[state.Column])

		if numeric {
			af, _ := strconv.ParseFloat(a, 64)
			bf, _ := strconv.ParseFloat(b, 64)
			if state.Direction == SortDesc {
				return af > bf
			}
			return af < bf
		}

		al, bl := strings.ToLower(a), strings.ToLower(b)
		if state.Direction == SortDesc {
			return al > bl
		}
		return al < bl
	})

	return sorted
}

// isNumericColumn reports whether every (trimmed) cell in the given column
// parses as a float. An empty rows slice is treated as non-numeric.
func isNumericColumn(rows []resource.Row, column int) bool {
	if len(rows) == 0 {
		return false
	}
	for _, row := range rows {
		if _, err := strconv.ParseFloat(strings.TrimSpace(row.Cells[column]), 64); err != nil {
			return false
		}
	}
	return true
}
