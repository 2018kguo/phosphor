//go:build ghostty

package main

import (
	"fmt"
	"testing"
)

func TestNativeCursorLifetimeAndRepaint(t *testing.T) {
	inputs := []string{
		"\x1b]8;;https://old\aA\x1b7\x1b]8;;\a\x1b8B",
		"\x1b[2;2H\x1b7\x1b[3;3H\x1b[?1049h\x1b[?1049l\x1b8X",
		"\x1b]8;;https://main\aA\x1b[?47h\x1b]8;;https://alt\aB\x1b[?47lC",
		"\x1b*A\x1bN\x1b7#\x1b8#",
		"\x1b*A\x1bN\x1b7",
		"\x1b*A\x1bN\x1b[?1049h\x1b7",
		"\x1b[?47h\x1b[2;2H\x1b[?1049h\x1b[3;3H\x1b[?1049h",
		"\x1b[?1049hALT\x1b[?1049l",
		"\x1b[?47h\x1b[?1049h\x1b[?1049l",
		"\x1b[?47h\x1b[?47;1047;1049s\x1b[?1049h\x1b[?47l",
	}
	for _, a := range []int{47, 1047, 1049} {
		for _, b := range []int{47, 1047, 1049} {
			for _, c := range []int{47, 1047, 1049} {
				for flags := 0; flags < 8; flags++ {
					input := "\x1b[31mA"
					for i, mode := range []int{a, b, c} {
						final := 'l'
						if flags&(1<<i) != 0 {
							final = 'h'
						}
						input += fmt.Sprintf("\x1b[?%d%c", mode, final) + "B"
					}
					inputs = append(inputs, input)
				}
			}
		}
	}
	for _, input := range inputs {
		p, n, r := newTracedEngine("phosphor", 8, 4), newTracedEngine("ghostty", 8, 4), newTracedEngine("ghostty", 8, 4)
		p.Write([]byte(input))
		n.Write([]byte(input))
		r.Write(p.Repaint(true))
		for _, suffix := range []string{"", "\x1b8#", "\x1b[?47l\x1b8M", "\x1b[?47h\x1b8A", "\x1b[?1049l\x1b8P", "\x1b[?47;1047;1049r", "\x1b[?47$p\x1b[?1047$p\x1b[?1049$p"} {
			for _, e := range []*tracedEngine{p, n, r} {
				e.Write([]byte(suffix))
			}
			if d := differences(traceState(t, p), traceState(t, n)); len(d) > 0 {
				t.Fatalf("direct input %q suffix %q: %+v", input, suffix, d)
			}
			if d := differences(traceState(t, n), traceState(t, r)); len(d) > 0 {
				t.Fatalf("repaint input %q suffix %q: %+v", input, suffix, d)
			}
		}
		p.Close()
		n.Close()
		r.Close()
	}
}
