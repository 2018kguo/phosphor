package phosphor

import (
	"bytes"
	"testing"
)

func TestOSCEndsAtEscape(t *testing.T) {
	for _, suffix := range []string{"\\", "[31mX", "X"} {
		term := New(8, 3)
		term.Write([]byte("\x1b]2;old\a\x1b]2;new\x1b"))
		if term.Title() != "new" {
			t.Fatalf("title before ST %q", term.Title())
		}
		if got := term.appendContinuation(nil); !bytes.Equal(got, []byte{0x1b}) {
			t.Fatalf("continuation %q, want ESC", got)
		}
		term.Write([]byte(suffix))
		if term.Title() != "new" {
			t.Fatalf("title after %q: %q", suffix, term.Title())
		}
		if suffix == "[31mX" && (term.Cell(0, 0).Rune != 'X' || term.Cell(0, 0).Style.Fg != IndexedColor(1)) {
			t.Fatal("escape after OSC did not execute")
		}
	}
}

func TestOSCCancel(t *testing.T) {
	for _, cancel := range []byte{0x18, 0x1a} {
		term := New(8, 3)
		term.Write(append([]byte("\x1b]2;old\a\x1b]2;cancel"), cancel, 'X'))
		if term.Title() != "old" || term.Cell(0, 0).Rune != 'X' {
			t.Fatalf("cancel %#x: title %q, cell %+v", cancel, term.Title(), term.Cell(0, 0))
		}
	}
}
