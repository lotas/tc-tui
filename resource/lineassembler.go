package resource

import "bytes"

// lineAssembler buffers streamed bytes and releases only complete lines —
// a chunk boundary can land anywhere (mid-line, mid-ANSI-escape-sequence,
// mid-rune), and the escaping pipeline downstream (escapeIgnoringANSI →
// tview.TranslateANSI) is only safe on intact lines. The zero value is
// ready to use.
type lineAssembler struct {
	buf []byte
}

// Feed appends chunk and returns every complete line accumulated so far
// (trailing newlines included), or "" when no newline has arrived yet.
func (a *lineAssembler) Feed(chunk []byte) string {
	a.buf = append(a.buf, chunk...)

	idx := bytes.LastIndexByte(a.buf, '\n')
	if idx < 0 {
		return ""
	}

	lines := string(a.buf[:idx+1])
	a.buf = append(a.buf[:0], a.buf[idx+1:]...)
	return lines
}

// Flush drains and returns whatever partial line is still buffered — called
// once at stream end, where an unterminated final line is real content.
func (a *lineAssembler) Flush() string {
	tail := string(a.buf)
	a.buf = a.buf[:0]
	return tail
}
