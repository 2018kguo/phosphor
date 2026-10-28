//go:build ghostty

package main

import "testing"

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
