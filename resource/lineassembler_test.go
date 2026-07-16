package resource

import "testing"

func TestLineAssemblerHoldsPartialLines(t *testing.T) {
	var a lineAssembler

	if got := a.Feed([]byte("hello\nwor")); got != "hello\n" {
		t.Fatalf("expected complete first line only, got %q", got)
	}
	if got := a.Feed([]byte("ld\n")); got != "world\n" {
		t.Fatalf("expected reassembled second line, got %q", got)
	}
	if got := a.Flush(); got != "" {
		t.Fatalf("expected nothing left to flush, got %q", got)
	}
}

func TestLineAssemblerNoNewlineYet(t *testing.T) {
	var a lineAssembler

	if got := a.Feed([]byte("partial")); got != "" {
		t.Fatalf("expected no output before a newline, got %q", got)
	}
	if got := a.Flush(); got != "partial" {
		t.Fatalf("expected the partial tail on flush, got %q", got)
	}
	// Flush drains: a second call must not repeat the tail.
	if got := a.Flush(); got != "" {
		t.Fatalf("expected flush to drain, got %q", got)
	}
}

func TestLineAssemblerMultipleLinesInOneChunk(t *testing.T) {
	var a lineAssembler

	if got := a.Feed([]byte("a\nb\nc\nd")); got != "a\nb\nc\n" {
		t.Fatalf("expected all complete lines at once, got %q", got)
	}
	if got := a.Flush(); got != "d" {
		t.Fatalf("expected the tail on flush, got %q", got)
	}
}

// A chunk boundary landing inside an ANSI escape sequence must not leak the
// half-sequence — the whole point of line buffering is that the escaping
// pipeline downstream only ever sees intact lines.
func TestLineAssemblerAnsiCodeSplitAcrossChunks(t *testing.T) {
	var a lineAssembler

	if got := a.Feed([]byte("x \x1b[3")); got != "" {
		t.Fatalf("expected no output mid-line, got %q", got)
	}
	if got := a.Feed([]byte("1mred\x1b[0m\n")); got != "x \x1b[31mred\x1b[0m\n" {
		t.Fatalf("expected the line reassembled intact, got %q", got)
	}
}
