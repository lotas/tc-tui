package main

import "testing"

func TestParseArgsNoArgs(t *testing.T) {
	name, scope, showHelp, showVersion := parseArgs(nil)
	if name != "" || scope != "" || showHelp || showVersion {
		t.Fatalf("parseArgs(nil) = (%q, %q, %v, %v), want all zero", name, scope, showHelp, showVersion)
	}
}

func TestParseArgsResourceAndScope(t *testing.T) {
	name, scope, showHelp, showVersion := parseArgs([]string{"wp", "proj-taskcluster/ci"})
	if name != "wp" || scope != "proj-taskcluster/ci" || showHelp || showVersion {
		t.Fatalf("parseArgs(wp, scope) = (%q, %q, %v, %v)", name, scope, showHelp, showVersion)
	}
}

func TestParseArgsResourceOnly(t *testing.T) {
	name, scope, _, _ := parseArgs([]string{"task"})
	if name != "task" || scope != "" {
		t.Fatalf("parseArgs(task) = (%q, %q)", name, scope)
	}
}

func TestParseArgsHelpFlag(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		_, _, showHelp, showVersion := parseArgs([]string{flag})
		if !showHelp || showVersion {
			t.Fatalf("parseArgs(%q) showHelp=%v showVersion=%v, want showHelp=true", flag, showHelp, showVersion)
		}
	}
}

func TestParseArgsVersionFlag(t *testing.T) {
	for _, flag := range []string{"-v", "--version"} {
		_, _, showHelp, showVersion := parseArgs([]string{flag})
		if !showVersion || showHelp {
			t.Fatalf("parseArgs(%q) showHelp=%v showVersion=%v, want showVersion=true", flag, showHelp, showVersion)
		}
	}
}

func TestParseArgsFlagAmongPositionals(t *testing.T) {
	name, scope, showHelp, _ := parseArgs([]string{"wp", "--help", "proj-taskcluster/ci"})
	if !showHelp {
		t.Fatalf("expected --help to be recognized regardless of position")
	}
	if name != "wp" || scope != "proj-taskcluster/ci" {
		t.Fatalf("flags shouldn't count as positional args: got name=%q scope=%q", name, scope)
	}
}
