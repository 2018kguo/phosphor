//go:build ghostty

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/2018kguo/phosphor"
)

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
