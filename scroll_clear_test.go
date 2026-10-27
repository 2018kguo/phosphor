package phosphor

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestScrollClear(t *testing.T) {
	cases := []struct {
		name       string
		cols, rows int
		input      string
		history    []string
		x, y       int
	}{
		{"partial", 8, 4, "A\r\nB", []string{"A", "B"}, 0, 0},
		{"below-content", 8, 4, "A\x1b[4;4H", []string{"A"}, 3, 2},
		{"empty", 8, 4, "\x1b[3;4H", nil, 3, 2},
		{"spaces", 8, 4, "  \r\n", []string{""}, 0, 0},
		{"soft-wrap", 3, 4, "abcdefg", []string{"abcdefg"}, 0, 0},
		{"wide-wrap", 3, 4, "ab漢", []string{"ab漢"}, 0, 0},
		{"colored-blanks", 8, 4, "\x1b[44m\x1b[2J", []string{"", "", "", ""}, 0, 0},
		{"protected", 8, 4, "\x1b[1\"qP\x1bVQ", []string{"PQ"}, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			term := New(c.cols, c.rows)
			term.Write([]byte(c.input))
			pen := term.cur.pen
			term.ClearDirty()
			term.Write([]byte("\x1b[22J"))
			if got := term.ScrollbackLines(); !slices.Equal(got, c.history) {
				t.Fatalf("history%q want%q", got, c.history)
			}
			if got := strings.TrimSpace(strings.Join(term.Lines(), "\n")); got != "" {
				t.Fatalf("screen not cleared:%q", got)
			}
			if cur := term.Cursor(); cur.X != c.x || cur.Y != c.y || cur.PendingWrap {
				t.Fatalf("cursor%+v want%d,%d", cur, c.x, c.y)
			}
			if term.cur.pen != pen {
				t.Fatalf("pen changed:%+v want%+v", term.cur.pen, pen)
			}
			for y := range c.rows {
				for x := range c.cols {
					if got := term.Cell(x, y); got.Style.Bg != 0 || got.Rune != 0 {
						t.Fatalf("cell%d,%d:%+v", x, y, got)
					}
				}
			}
			if len(c.history) > 0 && !term.Dirty() {
				t.Fatal("clear did not mark damage")
			}
			checkRepaint(t, term)
			repaint := term.Repaint(true)
			term.Write([]byte("\x1b[22J"))
			if got := term.Repaint(true); !bytes.Equal(got, repaint) {
				t.Fatal("clearing an empty screen changed state")
			}
		})
	}
}

func TestScrollClearMarginsAlternateAndBounds(t *testing.T) {
	t.Run("margins", func(t *testing.T) {
		term := feed(8, 4, "A\r\nB\r\nC\r\nD\x1b[2;3r\x1b[?69h\x1b[3;6s\x1b[22J")
		if got := term.ScrollbackLines(); !slices.Equal(got, []string{"A", "B", "C", "D"}) {
			t.Fatalf("history%q", got)
		}
		if term.top != 1 || term.bot != 2 || term.left != 2 || term.right != 5 || !term.lrmm {
			t.Fatal("margins changed")
		}
		checkRepaint(t, term)
	})
	t.Run("alternate", func(t *testing.T) {
		term := feed(8, 4, "MAIN\x1b[?1049h\x1b[HALT\x1b[22J")
		if term.ScrollbackLen() != 0 || strings.TrimSpace(strings.Join(term.Lines(), "\n")) != "" {
			t.Fatal("alternate screen not cleared")
		}
		term.Write([]byte("\x1b[?1049l"))
		if term.Line(0) != "MAIN" {
			t.Fatal("main screen changed")
		}
		checkRepaint(t, term)
	})
	t.Run("bounded", func(t *testing.T) {
		term := New(8, 4, WithScrollback(1))
		term.Write([]byte("A\r\nB\x1b[22J"))
		if got := term.ScrollbackLines(); !slices.Equal(got, []string{"B"}) {
			t.Fatalf("history budget not honored:%q", got)
		}
		checkRepaint(t, term)
	})
	t.Run("no-history", func(t *testing.T) {
		term := New(8, 4, WithScrollback(0))
		term.Write([]byte("A\r\nB\x1b[22J"))
		if term.ScrollbackLen() != 0 || term.Cursor().X != 0 || term.Cursor().Y != 0 {
			t.Fatal("history-disabled clear")
		}
		checkRepaint(t, term)
	})
	t.Run("selective", func(t *testing.T) {
		term := feed(8, 4, "\x1b[1\"qP\x1b[?22J")
		if term.Line(0) != "" || !slices.Equal(term.ScrollbackLines(), []string{"P"}) {
			t.Fatal("selective scroll-and-clear did not move protected content")
		}
		checkRepaint(t, term)
	})
	t.Run("saved-cursor", func(t *testing.T) {
		term := feed(8, 4, "\x1b[3;4H\x1b[31m\x1b7\x1b[HA\x1b[22J\x1b8X")
		if term.Line(2) != "   X" || term.Cell(3, 2).Style.Fg != IndexedColor(1) {
			t.Fatalf("saved cursor changed: %q, %+v", term.Lines(), term.Cell(3, 2))
		}
		checkRepaint(t, term)
	})
	t.Run("full-history-wrap-and-background", func(t *testing.T) {
		term := New(3, 2, WithScrollback(2))
		term.Write([]byte("old\r\nold\r\n\x1b[44mabcdefg\x1b[22J"))
		if got := term.ScrollbackLines(); !slices.Equal(got, []string{"old", "abcdefg"}) {
			t.Fatalf("history %q", got)
		}
		term.Write([]byte("X"))
		if got := term.Cell(0, 0); got.Rune != 'X' || got.Style.Bg != IndexedColor(4) {
			t.Fatalf("subsequent background changed: %+v", got)
		}
		for _, size := range [][2]int{{1, 4}, {6, 2}, {3, 2}} {
			term.Resize(size[0], size[1])
			if msg := term.checkInvariants(); msg != "" {
				t.Fatal(msg)
			}
			checkRepaint(t, term)
		}
	})
}

// DECRC can already leave the cursor outside an enabled origin region.
// Scroll-and-clear reaches the same state when it homes a scrolled-away
// cursor. VT repaint currently clamps that position; record the actual
// cursor and next printed cell because state() normalizes this known limit.
func TestScrollClearOutsideOriginReplayLimit(t *testing.T) {
	original := feed(8, 4, "A\x1b[2;3r\x1b[?69h\x1b[3;6s\x1b[?6hB\x1b[22J")
	if cur := original.Cursor(); cur.X != 0 || cur.Y != 0 || !original.cur.origin {
		t.Fatalf("scroll-and-clear cursor %+v, origin %v", cur, original.cur.origin)
	}
	restored := New(8, 4)
	restored.Write(original.Repaint(true))
	if cur := restored.Cursor(); cur.X != 2 || cur.Y != 1 || !restored.cur.origin {
		t.Fatalf("known repaint bound changed: cursor %+v, origin %v", cur, restored.cur.origin)
	}
	original.Write([]byte("X"))
	restored.Write([]byte("X"))
	if original.Line(0) != "X" || restored.Line(1) != "  X" {
		t.Fatalf("known continuation difference changed: original %q, restored %q", original.Lines(), restored.Lines())
	}
}

func TestScrollClearContinuation(t *testing.T) {
	for _, cols := range []int{1, 2, 3, 8} {
		for _, rows := range []int{1, 2, 4} {
			t.Run(fmt.Sprintf("%dx%d", cols, rows), func(t *testing.T) {
				input := []byte("ÁB\r\n漢\x1b[22JX\x1b[2;3H\x1b[22JY")
				for cut := 0; cut <= len(input); cut++ {
					a, b := New(cols, rows), New(cols, rows)
					a.Write(input[:cut])
					b.Write(input[:cut])
					b.Compact()
					restored := New(cols, rows)
					restored.Write(b.Repaint(true))
					if restored.Cursor() != a.Cursor() {
						t.Fatalf("cut %d: cursor changed across checkpoint", cut)
					}
					a.Write(input[cut:])
					b.Write(input[cut:])
					restored.Write(input[cut:])
					assertContinuationState(t, a, b, "compact+scroll-clear")
					assertContinuationState(t, a, restored, "scroll-clear checkpoint")
					if restored.Cursor() != a.Cursor() {
						t.Fatalf("cut %d: cursor changed after continuation", cut)
					}
					if msg := a.checkInvariants(); msg != "" {
						t.Fatal(msg)
					}
					checkRepaint(t, a)
					a.Resize(2, 3)
					restored.Resize(2, 3)
					assertContinuationState(t, a, restored, "scroll-clear checkpoint resize")
				}
			})
		}
	}
}

func TestScrollClearPreservesREP(t *testing.T) {
	input := []byte("AB\x1b[22J\x1b[3b")
	for cut := 0; cut <= len(input); cut++ {
		term := New(8, 4)
		term.Write(input[:cut])
		term.Compact()
		term.Write(input[cut:])
		if term.Line(0) != "BBB" || term.Cursor().X != 3 || term.Cursor().Y != 0 {
			t.Fatalf("cut %d: REP changed: %q, %+v", cut, term.Lines(), term.Cursor())
		}
	}
}
