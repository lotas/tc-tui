package controller

import (
	"strings"
	"testing"
)

// `tc-tui --help` must never require a configured deployment just to print
// usage: HelpText has to work with no TASKCLUSTER_ROOT_URL in the
// environment (where constructing a real Taskcluster client panics).
func TestHelpTextNeedsNoTaskclusterClient(t *testing.T) {
	t.Setenv("TASKCLUSTER_ROOT_URL", "")

	text := HelpText()

	for _, want := range []string{"workerpools", "hooks", "secrets", "task"} {
		if !strings.Contains(text, want) {
			t.Fatalf("help text missing %q:\n%s", want, text)
		}
	}
}
