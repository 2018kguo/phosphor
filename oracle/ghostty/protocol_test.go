//go:build ghostty

package main

import (
	"bytes"
	"fmt"
	"github.com/2018kguo/phosphor"
	"strings"
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

func TestNativeKittyColorProtocol(t *testing.T) {
	for _, suffix := range []string{
		"1=rgb:12/34/56;1=?", "foreground=red;foreground=?", "background=rgbi:0/.5/1;background=?",
		"cursor=#aabbcc;cursor=?", "1=rgb:f/0/8;1=?", "1=FoReStGrEeN;1=?",
		"1=red;1=;1=?", "1=red;1;1=?", "001=red;001=?", "1_0=red;1_0=?;1=?",
		"foreground=bad-color;foreground=?", "selection_foreground=?;cursor_text=?;unknown=?;1=?",
		"1=;" + strings.Repeat("0=;", 528) + "1=?",
	} {
		for _, end := range []string{"\a", "\x1b\\"} {
			t.Run(suffix[:min(len(suffix), 60)]+end, func(t *testing.T) {
				p, n := newTracedEngine("phosphor", 8, 3), newTracedEngine("ghostty", 8, 3)
				defer p.Close()
				defer n.Close()
				// Give both dynamic colors concrete values before querying.
				// Palette resets are checked against each host's own base,
				// since their built-in themes differ.
				reset := strings.Contains(suffix, ";1=;") || strings.Contains(suffix, ";1;")
				base := make(map[*tracedEngine]string)
				if reset {
					for _, engine := range []*tracedEngine{p, n} {
						engine.Write([]byte("\x1b]21;1=?" + end))
						base[engine] = traceState(t, engine).Replies
					}
				}
				input := "\x1b]4;1;rgb:01/02/03\x1b\\\x1b]10;rgb:04/05/06;rgb:07/08/09;rgb:0a/0b/0c\x1b\\\x1b]21;" + suffix + end
				for _, engine := range []*tracedEngine{p, n} {
					engine.Write([]byte(input))
					if reset && traceState(t, engine).Replies != base[engine]+base[engine] {
						t.Fatalf("reset did not restore this host's base: %q", traceState(t, engine).Replies)
					}
				}
				if got, want := traceState(t, p).Replies, traceState(t, n).Replies; !reset && got != want {
					t.Fatalf("phosphor %q, ghostty %q", got, want)
				}
			})
		}
	}
}

func TestNativeExtendedColorEdges(t *testing.T) {
	for _, code := range []string{"38", "48", "58"} {
		for _, suffix := range []string{
			";5;0", ";5;255", ";5;256", ";5;65535", ";2;1;2;3", ";2;256;257;65535",
			":5:3", ":2:1:2:3", ":2::1:2:3", ":2:0:1:2:3", ":2:9:1:2:3",
			"", ";5", ";2", ";2;1", ";2;1;2", ";3;4", ":5", ":2:1:2", ":3:4", ":2:1:2:3:4:5", ":2::1:2:3:4",
			":5;1", ";5:1", ":5:1:2", ":5:1:2:3", ":2:1;2;3", ":2:1:2;3", ";2:1:2:3", ";2;1:2;3", ":2;1;2;3", ":2;1;2;3;4", ":2:1:2:3;1",
			":", ":5:", ":2:", ":2::1:2", ";5;65536", ";5;999999999", ";2;65536;999999999;257",
		} {
			t.Run(code+suffix, func(t *testing.T) {
				input := []byte("\x1b[38;5;9;48;5;10;58;5;11m\x1b[" + code + suffix + "mA")
				var reply bytes.Buffer
				p := phosphor.New(4, 2, phosphor.WithReply(&reply))
				p.Write(input)
				n := newTracedEngine("ghostty", 4, 2)
				defer n.Close()
				n.Write(input)
				got, want := p.Cell(0, 0).Style, traceState(t, n).Screen[0][0]
				if uint32(got.Fg) != want.Fg || uint32(got.Bg) != want.Bg || uint32(got.UnderlineColor) != want.UnderlineColor || uint16(got.Attrs) != want.Attrs || uint8(got.Underline) != want.Underline {
					t.Fatalf("phosphor=%+v ghostty=%+v", got, want)
				}
				p.Write([]byte("\x1bP$qm\x1b\\"))
				n.Write([]byte("\x1bP$qm\x1b\\"))
				for name, response := range map[string]string{"phosphor": reply.String(), "ghostty": traceState(t, n).Replies} {
					body, ok := strings.CutPrefix(response, "\x1bP1$r")
					body, end := strings.CutSuffix(body, "\x1b\\")
					if !ok || !end {
						t.Fatalf("%s invalid DECRQSS response: %q", name, response)
					}
					decoded := phosphor.New(4, 2)
					decoded.Write([]byte("\x1b[" + body + "A"))
					expected := got
					if name == "ghostty" {
						// Ghostty's reply omits underline color. Keep observing
						// that omission; phosphor returns the complete style.
						expected.UnderlineColor = 0
					}
					if decoded.Cell(0, 0).Style != expected {
						t.Fatalf("%s DECRQSS style: %q, %+v / %+v", name, response, decoded.Cell(0, 0).Style, expected)
					}
				}
			})
		}
	}
}
