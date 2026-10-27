package phosphor

import (
	"bytes"
	"testing"
)

func TestPendingWrapSurvivesModeAndTabs(t *testing.T) {
	for _, mode := range []string{"", "\x1b[?7l"} {
		for _, tab := range []string{"\t", "\x1b[I", "\x1b[Z"} {
			term := New(6, 3)
			term.Write([]byte("abcdef" + mode + tab))
			wantX := 5
			if tab == "\x1b[Z" {
				wantX = 0
			}
			if cur := term.Cursor(); cur.X != wantX || cur.Y != 0 || !cur.PendingWrap {
				t.Fatalf("mode=%q tab=%q: %+v", mode, tab, cur)
			}
			term.Compact()
			restored := New(6, 3)
			restored.Write(term.Repaint(false))
			if restored.Cursor() != term.Cursor() || restored.Modes() != term.Modes() {
				t.Fatalf("repaint changed pending wrap: mode=%q tab=%q", mode, tab)
			}
			for _, e := range []*Terminal{term, restored} {
				e.Write([]byte("\x1b[?7hX"))
				if cur := e.Cursor(); cur.X != 1 || cur.Y != 1 || cur.PendingWrap || e.Cell(0, 1).Text() != "X" {
					t.Fatalf("mode=%q tab=%q did not wrap next print: %+v", mode, tab, cur)
				}
			}
		}
	}
}

func TestWideGlyphWithoutRoomMatchesNative(t *testing.T) {
	for _, text := range []string{"AAAA", "AAAAA"} {
		term := New(5, 3)
		term.Write([]byte("\x1b[?7l" + text + "🚨"))
		if term.Cursor().X != 4 || term.Cursor().PendingWrap || term.Lines()[0] != text {
			t.Fatalf("did not drop unprintable wide glyph: %+v %q", term.Cursor(), term.Lines())
		}
		term.Write([]byte("\x1b[?7hX"))
		if term.Cursor().Y != 0 || term.Cell(4, 0).Text() != "X" || !term.Cursor().PendingWrap {
			t.Fatal("dropped glyph changed future wrapping", term.Cursor())
		}
	}
	term := New(1, 3)
	term.Write([]byte("\x1b[1;31m😀"))
	if c := term.Cell(0, 0); c.Text() != "" || c.Style.Fg != IndexedColor(1) || c.Style.Attrs&AttrBold == 0 || !term.Cursor().PendingWrap {
		t.Fatal("one-column dropped glyph lost native blank style/pending wrap", c, term.Cursor())
	}
	copy := New(1, 3)
	copy.Write(term.Repaint(false))
	if copy.Cell(0, 0) != term.Cell(0, 0) || copy.Cursor() != term.Cursor() {
		t.Fatal("one-column repaint lost dropped-wide state")
	}
	term.Write([]byte("X"))
	if term.Cursor().Y != 1 || term.Cell(0, 1).Text() != "X" {
		t.Fatal("one-column dropped glyph failed to wrap the next print")
	}
}

func TestSmoothScrollingModeSavesAndRepaints(t *testing.T) {
	var replies bytes.Buffer
	term := New(12, 3, WithReply(&replies))
	term.Write([]byte("\x1b[?4h\x1b[?4s\x1b[?4l\x1b[?4r\x1b[?4$p"))
	if !term.Modes().SmoothScrolling || replies.String() != "\x1b[?4;1$y" {
		t.Fatal(term.Modes(), replies.String())
	}
	term.Compact()
	copy := New(12, 3)
	copy.Write(term.Repaint(false))
	if !copy.Modes().SmoothScrolling {
		t.Fatal("repaint discarded smooth scrolling")
	}
	copy.Write([]byte("\x1b[?4l\x1b[?4r"))
	if !copy.Modes().SmoothScrolling {
		t.Fatal("repaint discarded the saved smooth scrolling mode")
	}
}
