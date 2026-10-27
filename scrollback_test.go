package phosphor

import (
	"fmt"
	"strings"
	"testing"
)

func TestScrollbackRecyclesEvictedLines(t *testing.T) {
	for _, opts := range [][]Option{{WithScrollback(8)}, {WithScrollbackBytes(256)}} {
		term := New(40, 2, opts...)
		line := []byte("\x1b[32mINFO\x1b[0m steady bounded output\r\n")
		for range 100 {
			term.Write(line)
		}
		if allocs := testing.AllocsPerRun(100, func() {
			for range 100 {
				term.Write(line)
			}
		}); allocs > 1 {
			t.Fatalf("steady scrolling allocated %.1f times per 100 lines", allocs)
		}
		for i := range 30 {
			term.Write([]byte(fmt.Sprintf("line %02d\r\n", i)))
		}
		got := term.ScrollbackLines()
		if len(got) == 0 || got[len(got)-1] != "line 28" {
			t.Fatalf("scrollback order: %q", got)
		}
		fresh := New(40, 2, opts...)
		fresh.Write(term.Repaint(true))
		if strings.Join(fresh.ScrollbackLines(), "\n") != strings.Join(got, "\n") {
			t.Fatalf("repaint changed recycled history")
		}
	}
}

func TestShortScrollbackLinesDoNotKeepLargeBuffers(t *testing.T) {
	term := New(100, 2, WithScrollback(8), WithScrollbackBytes(1<<20))
	term.Write([]byte(strings.Repeat("x", 8192) + "\r\n"))
	for range 40 {
		term.Write([]byte("tiny\r\n"))
	}
	for _, line := range term.sb[term.sbStart:] {
		if cap(line) > 128 {
			t.Fatalf("short line retained %d-byte buffer", cap(line))
		}
	}
}
