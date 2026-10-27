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
