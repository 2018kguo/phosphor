//go:build ghostty

package main

import "testing"

func TestNativePhosphorRepaintSavedPendingWrap(t *testing.T) {
	for _, tc := range []struct {
		input string
		cols  int
	}{
		{"abcd\x1b7\rX", 4},
		{"abcd\x1b7\x1b[2J", 4},
		{"ab漢\x1b7\rX", 4},
		{"abcd\x1b[31m\x1b]8;;https://saved\x1b\\\x1b7\x1b[0m\x1b]8;;\x1b\\\rX", 4},
		{"\x1b[?6habcd\x1b7\x1b[?6l\rX", 4},
		{"\x1b[?69h\x1b[1;2sab\x1b7\x1b[?69l\rX", 8},
		{"\x1b[3g\x1b[3G\x1bH\x1b[6G\x1bH\rabcdefgh\x1b[Z\x1b7\rX", 8},
		{"abcdefgh\x1b[Z\x1b7\rX", 8},
		{"a\x1b7\rX", 1},
		{"abcd\x1b7e\rX", 4},
		{"main\x1b[?1049h\x1b[Habcd\x1b7\rX", 4},
		{"\x1b[?47habcd\x1b7\rX\x1b[?47l", 4},
	} {
		p := newTracedEngine("phosphor", tc.cols, 3)
		g := newTracedEngine("ghostty", tc.cols, 3)
		r := newTracedEngine("ghostty", tc.cols, 3)
		p.Write([]byte(tc.input))
		g.Write([]byte(tc.input))
		r.Write(p.Repaint(true))
		for _, suffix := range []string{"", "\x1b8Z", "\x1b[?47h\x1b8Q\x1b[?47l", "\x1b[?1049lR", "\x1b8S"} {
			g.Write([]byte(suffix))
			r.Write([]byte(suffix))
			if diff := differences(traceState(t, g), traceState(t, r)); len(diff) > 0 {
				t.Fatalf("input %q after %q: %+v", tc.input, suffix, diff)
			}
		}
		p.Close()
		g.Close()
		r.Close()
	}
}
