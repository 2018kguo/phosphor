package phosphor

import (
	"bytes"
	"testing"
)

func TestExtendedColorMalformedAndContinuation(t *testing.T) {
	for _, code := range []string{"38", "48", "58"} {
		for _, tc := range []struct {
			suffix    string
			attrs     Attr
			underline Underline
		}{
			{"", 0, 0}, {";5", AttrBlink, 0}, {";2;1;2", AttrBold | AttrFaint, 0},
			{";3;4", AttrItalic, UnderlineSingle}, {":5", AttrBlink, 0},
			{":", 0, 0}, {":5:", 0, 0}, {":2:", 0, 0}, {":2:1:2", 0, 0},
			{":2:1:2:3:4:5", 0, 0}, {":3:4", 0, 0},
		} {
			t.Run(code+tc.suffix, func(t *testing.T) {
				in := []byte("\x1b[38;5;9;48;5;10;58;5;11m\x1b[" + code + tc.suffix + "mA\x1bP$qm\x1b\\")
				want := Style{Fg: IndexedColor(9), Bg: IndexedColor(10), UnderlineColor: IndexedColor(11), Attrs: tc.attrs, Underline: tc.underline}
				for cut := 0; cut <= len(in); cut++ {
					var reply bytes.Buffer
					term := New(8, 2, WithReply(&reply))
					term.Write(in[:cut])
					var restored Terminal
					if err := restored.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
						t.Fatalf("cut %d: %v", cut, err)
					}
					painted := New(8, 2)
					painted.Write(term.Repaint(true))
					for _, emu := range []*Terminal{term, &restored, painted} {
						emu.Write(in[cut:])
						if got := emu.Cell(0, 0).Style; got != want {
							t.Fatalf("cut %d: got %+v, want %+v", cut, got, want)
						}
					}
					// The response must describe underline color even though
					// Ghostty currently omits it from its DECRQSS response.
					if !bytes.Contains(reply.Bytes(), []byte(";58;5;11")) {
						t.Fatalf("cut %d: incomplete style reply %q", cut, reply.Bytes())
					}
				}
			})
		}
	}
}
