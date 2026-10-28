//go:build ghostty

package main

import (
	"fmt"
	"testing"

	"github.com/2018kguo/phosphor"
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

// Native Terminal.zig switchScreenMode clears the whole ALT on 1047 reset;
// deccolm only reaches 80/132, so it cannot stage and restore a 47-column host.
func TestNativeRepaintRectangleGrowthLimit(t *testing.T) {
	p, n, s, r := newTracedEngine("phosphor", 13, 12), newTracedEngine("ghostty", 13, 12), newTracedEngine("phosphor", 47, 8), newTracedEngine("ghostty", 47, 8)
	defer p.Close()
	defer n.Close()
	defer s.Close()
	defer r.Close()
	for _, e := range []*tracedEngine{p, n} {
		e.Write([]byte("\x1b[?47h\x1b[?1049h\x1b[48;2;1;2;3m\x1b[?1047l"))
		e.Resize(47, 8)
	}
	if err := s.Engine.(*phos).t.RestoreSnapshot(p.Snapshot(), phosphor.SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	r.Write(p.Repaint(false))
	for _, suffix := range []string{"", "\x1b[?47h"} {
		for _, e := range []*tracedEngine{p, n, s, r} {
			e.Write([]byte(suffix))
		}
		if d := differences(traceState(t, p), traceState(t, n)); len(d) > 0 {
			t.Fatalf("direct: %+v", d)
		}
		if d := differences(traceState(t, s), traceState(t, n)); len(d) > 0 {
			t.Fatalf("snapshot: %+v", d)
		}
		expected := traceState(t, n)
		if suffix != "" {
			if d := differences(expected, traceState(t, r)); len(d) == 0 {
				t.Fatal("raw ANSI loss disappeared; reassess boundary")
			}
			bg := expected.Screen[0][0].Bg
			for y := range expected.Screen {
				for x := 13; x < 47; x++ {
					expected.Screen[y][x].Bg = bg
				}
			}
		}
		if d := differences(expected, traceState(t, r)); len(d) > 0 {
			t.Fatalf("ANSI changed more than zero-padding background: %+v", d)
		}
	}
	for _, e := range []*tracedEngine{p, n, s} {
		e.Write([]byte("\x1b8X\x1b[?1049l\x1b8Y"))
		e.Resize(19, 9)
	}
	if d := differences(traceState(t, p), traceState(t, n)); len(d) > 0 {
		t.Fatalf("direct future: %+v", d)
	}
	if d := differences(traceState(t, s), traceState(t, n)); len(d) > 0 {
		t.Fatalf("snapshot future: %+v", d)
	}
}
