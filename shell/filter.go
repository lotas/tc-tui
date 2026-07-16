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

// mergeRowsByID returns a copy of base with each row whose ID also appears
// in updates replaced by that updated row, preserving base's order and
// length. Used to fold an Augment result computed over only a filtered
// subset of rows back into the full row set, leaving filtered-out rows
// (absent from updates) untouched rather than discarding them.
func mergeRowsByID(base, updates []resource.Row) []resource.Row {
	byID := make(map[string]resource.Row, len(updates))
	for _, u := range updates {
		byID[u.ID] = u
	}

	merged := make([]resource.Row, len(base))
	for i, row := range base {
		if u, ok := byID[row.ID]; ok {
			merged[i] = u
		} else {
			merged[i] = row
		}
	}

	return merged
}
