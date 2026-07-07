package shell

import (
	"strings"

	"github.com/taskcluster/tc-tui/resource"
)

// FilterRows returns rows whose cells substring-match query
// (case-insensitive), applied client-side over an already-fetched List()
// result — no extra API calls.
func FilterRows(rows []resource.Row, query string) []resource.Row {
	if query == "" {
		return rows
	}

	q := strings.ToLower(query)
	filtered := make([]resource.Row, 0, len(rows))
	for _, row := range rows {
		if rowMatches(row, q) {
			filtered = append(filtered, row)
		}
	}

	return filtered
}

func rowMatches(row resource.Row, lowerQuery string) bool {
	for _, cell := range row.Cells {
		if strings.Contains(strings.ToLower(cell), lowerQuery) {
			return true
		}
	}

	return false
}
