//go:build ghostty

package main

import (
	"testing"
)

func TestNativePhosphorRepaintSavedPendingWrap(t *testing.T) {
	for _, tc := range []struct {
		input string
		cols  int
	}{
		{"abcd\x1b7\rX", 4},
		{"abcd\x1b[?47h\x1b7X", 4},
		{"abcd\x1b[41m\x1b[?47h\x1b7X", 4},
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

func TestNativePhosphorRepaintOutsideOrigin(t *testing.T) {
	for _, prefix := range []string{
		"",
		"\x1b[?47h\x1b[3;4H\x1b[32m\x1b)A\x0e\x1b7\x1b[?47l",
		"\x1b[?6h\x1b[5;8H\x1b[34m\x1b(0\x1b7\x1b[?6l",
	} {
		for _, suffix := range []string{"", "q\x1b[B\x1b[C\x1b[41;1m\x1b[1\"q"} {
			p := newTracedEngine("phosphor", 10, 6)
			g := newTracedEngine("ghostty", 10, 6)
			r := newTracedEngine("ghostty", 10, 6)
			input := []byte(prefix + "\x1b[0m\x1b(B\x0f\x1b[H\x1b[2JA\x1b[3;5r\x1b[?69h\x1b[4;8s\x1b[?6hB\x1b[22J" + suffix)
			p.Write(input)
			g.Write(input)
			r.Write(p.Repaint(true))
			for _, next := range []string{"", "X", "\x1b8Z", "\x1b[?47h\x1b8A\x1b[?47l", "\x1b[?1047h\x1b8B\x1b[?1047l", "\x1b[?1049hQ\x1b[?1049lR"} {
				g.Write([]byte(next))
				r.Write([]byte(next))
				if diff := differences(traceState(t, g), traceState(t, r)); len(diff) > 0 {
					t.Fatalf("prefix %q suffix %q after %q: %+v", prefix, suffix, next, diff)
				}
			}
			p.Close()
			g.Close()
			r.Close()
		}
	}
}

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

func TestNativeRareStyledSpacesSurviveMode47(t *testing.T) {
	p, n := newTracedEngine("phosphor", 8, 3), newTracedEngine("ghostty", 8, 3)
	defer p.Close()
	defer n.Close()
	for _, input := range []string{
		"\x1b[?47h\x1b[58;5;3m\x1b]8;;https://space\a \x1b]8;;\a\x1b[0m",
		"\x1b[?47l", "\x1b[?47h",
	} {
		p.Write([]byte(input))
		n.Write([]byte(input))
		p.Engine.(*phos).t.Compact()
		if diff := differences(traceState(t, p), traceState(t, n)); len(diff) != 0 {
			t.Fatalf("after %q: %+v", input, diff)
		}
	}
}

func TestNativeColdPrefixErase(t *testing.T) {
	for _, input := range []string{
		"\x1b[32muser@host\x1b[0m:~$ \x1b[K",
		"\x1b[58;5;3m\x1b]8;;https://wide\a界界\x1b[0m\x1b]8;;\a\x1b[2G\x1b[X",
		"X\x1b[80G\x1b[K",
		"\x1b[44mshort\x1b[K",
		"\x1b[?47hshort\x1b[K\x1b[H\x1b[4hZ\x1b[4l\x1b[?47l",
		"X\r\nY\r\nZ\r\nW",
	} {
		t.Run(input[:min(32, len(input))], func(t *testing.T) {
			p, n := newTracedEngine("phosphor", 20, 3), newTracedEngine("ghostty", 20, 3)
			defer p.Close()
			defer n.Close()
			for _, c := range []byte(input) {
				p.Write([]byte{c})
				n.Write([]byte{c})
				if diff := differences(traceState(t, p), traceState(t, n)); len(diff) != 0 {
					t.Fatalf("after byte %02x: %+v", c, diff)
				}
			}
		})
	}
}
