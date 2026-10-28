//go:build ghostty

package main

import "testing"

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
