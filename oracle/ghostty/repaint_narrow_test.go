//go:build ghostty

package main

import "testing"

func TestNativePhosphorRepaintNarrowEdge(t *testing.T) {
	for _, tc := range []struct {
		prefix string
		cols   int
	}{{"A", 2}, {"AB", 3}, {"A界", 4}} {
		for _, history := range []bool{false, true} {
			p := newTracedEngine("phosphor", tc.cols+1, 4)
			g := newTracedEngine("ghostty", tc.cols+1, 4)
			r := newTracedEngine("ghostty", tc.cols, 4)
			input := "\x1b[?2027h" + tc.prefix + "\x1b[31;4m\x1b]8;;https://narrow\x1b\\☔︎\x1b]8;;\x1b\\\x1b[0m\x1b[K"
			if history {
				input += "\r\n1\r\n2\r\n3\r\n4"
			}
			p.Write([]byte(input))
			g.Write([]byte(input))
			p.Resize(tc.cols, 4)
			g.Resize(tc.cols, 4)
			p.Write([]byte("\x1b[3;1H"))
			g.Write([]byte("\x1b[3;1H"))
			r.Write(p.Repaint(true))
			if diff := differences(traceState(t, p), traceState(t, g)); len(diff) > 0 {
				t.Logf("direct resize input %q prefix %q history %v: %+v", input, tc.prefix, history, diff)
			}
			for _, suffix := range []string{"", "Z", "\x1b8Q", "\r\nnext\r\nlast"} {
				p.Write([]byte(suffix))
				r.Write([]byte(suffix))
				if diff := differences(traceState(t, p), traceState(t, r)); len(diff) > 0 {
					t.Fatalf("prefix %q history %v after %q: %+v", tc.prefix, history, suffix, diff)
				}
			}
			p.Close()
			g.Close()
			r.Close()
		}
	}
}
