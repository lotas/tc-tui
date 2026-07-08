package shell

import "github.com/taskcluster/tc-tui/resource"

// FacetTab is one rendered tab in the facet bar: its underlying value and
// the number of currently-visible rows matching it. Value "" represents the
// "All" tab (client-side facets only).
type FacetTab struct {
	Value string
	Count int
}

// FilterByFacet returns rows narrowed to those whose FacetColumn() cell
// equals value. If faceted is nil or value is "" (the "All" tab), rows is
// returned unchanged.
func FilterByFacet(rows []resource.Row, faceted resource.Faceted, value string) []resource.Row {
	if faceted == nil || value == "" {
		return rows
	}

	col := faceted.FacetColumn()
	filtered := make([]resource.Row, 0, len(rows))
	for _, row := range rows {
		if col < len(row.Cells) && row.Cells[col] == value {
			filtered = append(filtered, row)
		}
	}

	return filtered
}

// ClientFacetTabs builds the "All" + per-option tab list for a client-side
// Faceted resource, counting matches within rows (expected to already be
// text-filtered, per refreshTable's pipeline).
func ClientFacetTabs(faceted resource.Faceted, rows []resource.Row) []FacetTab {
	options := faceted.FacetOptions(rows)
	col := faceted.FacetColumn()

	tabs := make([]FacetTab, 0, len(options)+1)
	tabs = append(tabs, FacetTab{Value: "", Count: len(rows)})

	for _, opt := range options {
		count := 0
		for _, row := range rows {
			if col < len(row.Cells) && row.Cells[col] == opt {
				count++
			}
		}
		tabs = append(tabs, FacetTab{Value: opt, Count: count})
	}

	return tabs
}

// ServerFacetTabs builds the tab list for a ServerFaceted resource directly
// from its FacetOptions() and a separately-fetched counts map — no row
// scanning, since the loaded rows only ever cover the currently-selected
// tab.
func ServerFacetTabs(serverFaceted resource.ServerFaceted, counts map[string]int) []FacetTab {
	options := serverFaceted.FacetOptions()

	tabs := make([]FacetTab, 0, len(options))
	for _, opt := range options {
		tabs = append(tabs, FacetTab{Value: opt, Count: counts[opt]})
	}

	return tabs
}

// cycleFacetValue returns the value of the tab that follows current in tabs
// (direction 1) or precedes it (direction -1), wrapping around at the ends.
// If current isn't found among tabs, it's treated as index 0.
func cycleFacetValue(tabs []FacetTab, current string, direction int) string {
	if len(tabs) == 0 {
		return current
	}

	idx := 0
	for i, tab := range tabs {
		if tab.Value == current {
			idx = i
			break
		}
	}

	idx = (idx + direction) % len(tabs)
	if idx < 0 {
		idx += len(tabs)
	}

	return tabs[idx].Value
}
