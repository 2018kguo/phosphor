package phosphor

import (
	"bytes"
	"testing"
)

func TestCursorAppearance(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		shape       CursorShape
		blink       bool
		parameter   string
	}{
		{"default", "", CursorDefault, false, "2"},
		{"blink-default", "\x1b[?12h", CursorDefault, true, "1"},
		{"steady-bar", "\x1b[5 q\x1b[?12l", CursorBar, false, "6"},
		{"blink-underline", "\x1b[4 q\x1b[?12h", CursorUnderline, true, "3"},
		{"invalid-style", "\x1b[5 q\x1b[7 q\x1b[65536 q", CursorBar, true, "5"},
		{"saved-blink", "\x1b[?12h\x1b[?12s\x1b[?12l\x1b[?12r", CursorDefault, true, "1"},
		{"soft-reset", "\x1b[5 q\x1b[!p", CursorDefault, false, "2"},
		{"hard-reset", "\x1b[5 q\x1bc", CursorDefault, false, "2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var replies bytes.Buffer
			term := New(10, 3, WithReply(&replies))
			term.Write([]byte(tc.input))
			if cur := term.Cursor(); cur.Shape != tc.shape || cur.Blink != tc.blink {
				t.Fatalf("cursor %+v, want shape %v blink %v", cur, tc.shape, tc.blink)
			}
			term.Write([]byte("\x1bP$q q\x1b\\\x1b[?12$p"))
			mode := "2"
			if tc.blink {
				mode = "1"
			}
			if want := "\x1bP1$r" + tc.parameter + " q\x1b\\\x1b[?12;" + mode + "$y"; replies.String() != want {
				t.Fatalf("replies %q, want %q", replies.String(), want)
			}
			restored := New(10, 3)
			restored.Write(term.Repaint(false))
			if got := restored.Cursor(); got.Shape != tc.shape || got.Blink != tc.blink {
				t.Fatalf("repaint cursor %+v, want %+v", got, term.Cursor())
			}
		})
	}
}
