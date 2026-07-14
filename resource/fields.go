package resource

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// fieldTagRe matches tview's "[tag]" region/color markup, stripped when
// measuring a field's rendered width so grid padding lines up with what's
// actually drawn, not the tag bytes.
var fieldTagRe = regexp.MustCompile(`\[[^][]*\]`)

// visibleWidth measures s's rendered width, ignoring tview region tags.
func visibleWidth(s string) int {
	return utf8.RuneCountInString(fieldTagRe.ReplaceAllString(s, ""))
}

// fieldRow renders a handful of "[green]Label:[white] value" fields on one
// line, each but the last padded to width so consecutive fieldRow calls
// line up into a loose grid — used to cut vertical scrolling in Describe()
// bodies that otherwise put several short single-value fields one per
// line. pairs alternates label, value, label, value, ... An odd trailing
// element (a caller bug) is dropped. Multi-line fields (YAML/markdown
// blocks) are never passed through this — those stay full-width, as today.
func fieldRow(width int, pairs ...string) string {
	var b strings.Builder
	for i := 0; i+1 < len(pairs); i += 2 {
		field := fmt.Sprintf("[green]%s:[white] %s", pairs[i], pairs[i+1])
		if i+2 < len(pairs) {
			if pad := width - visibleWidth(field); pad > 0 {
				field += strings.Repeat(" ", pad)
			} else {
				field += " "
			}
		}
		b.WriteString(field)
	}
	b.WriteString("\n")
	return b.String()
}
