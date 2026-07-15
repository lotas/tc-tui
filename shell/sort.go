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
			af, _ := parseNumericCell(a)
			bf, _ := parseNumericCell(b)
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
// parses via parseNumericCell. An empty rows slice is treated as
// non-numeric.
func isNumericColumn(rows []resource.Row, column int) bool {
	if len(rows) == 0 {
		return false
	}
	for _, row := range rows {
		if _, ok := parseNumericCell(strings.TrimSpace(row.Cells[column])); !ok {
			return false
		}
	}
	return true
}

// byteSizeUnits maps a resource.formatBytes suffix to its power-of-1024
// byte count, so a SIZE column sorts by actual magnitude rather than
// lexically by rendered unit (e.g. "2.0 KiB" lexically preceding "500 B",
// or "4.2 MiB" preceding "800 KiB", both wrong by size).
var byteSizeUnits = map[string]float64{
	"B":   1,
	"KIB": 1024,
	"MIB": 1024 * 1024,
	"GIB": 1024 * 1024 * 1024,
	"TIB": 1024 * 1024 * 1024 * 1024,
	"PIB": 1024 * 1024 * 1024 * 1024 * 1024,
	"EIB": 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
}

// parseNumericCell parses a cell as a bare number (e.g. "42") or as a
// resource.formatBytes-rendered byte size (e.g. "2.0 KiB", "512 B"),
// returning the raw byte count in the latter case.
func parseNumericCell(s string) (float64, bool) {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, true
	}

	fields := strings.Fields(s)
	if len(fields) != 2 {
		return 0, false
	}

	n, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, false
	}

	mult, ok := byteSizeUnits[strings.ToUpper(fields[1])]
	if !ok {
		return 0, false
	}

	return n * mult, true
}
