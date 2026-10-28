//go:build ghostty

package main

import (
	"strings"
	"testing"
)

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
