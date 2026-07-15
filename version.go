package main

import (
	"fmt"
	"runtime/debug"
)

// printVersion prints tc-tui's own client version and local build hash —
// distinct from the Taskcluster server version shown in the app's header
// (taskcluster.Taskcluster.GetVersion). Sourced from the Go toolchain's VCS
// build stamping (populated automatically for `go build`/`go run` inside a
// git checkout), so there's no ldflags step to wire up.
func printVersion() {
	version := "(devel)"
	revision := "unknown"

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" {
			version = info.Main.Version
		}
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				revision = s.Value
			}
		}
	}

	if len(revision) > 12 {
		revision = revision[:12]
	}

	fmt.Printf("tc-tui %s (build %s)\n", version, revision)
}
