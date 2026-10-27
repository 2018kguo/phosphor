package phosphor

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestCSIParameterSlotsAndSaturation(t *testing.T) {
	for count := 1; count <= maxParams+2; count++ {
		parts := make([]string, count)
		want := make([]uint32, count)
		for i := range parts {
			switch i % 5 {
			case 0:
				parts[i], want[i] = "0", 0
			case 1:
				parts[i], want[i] = "65535", 65535
			case 2:
				parts[i], want[i] = "65536", 65536
			case 3:
				parts[i], want[i] = "65537", 65536
			case 4:
				parts[i], want[i] = "999999999999999999999999", 65536
			}
		}
		for _, separator := range []string{";", ":"} {
			in := []byte("\x1b[" + strings.Join(parts, separator))
			for chunk := 1; chunk <= len(in); chunk++ {
				term := New(12, 3)
				for off := 0; off < len(in); off += chunk {
					term.Write(in[off:min(off+chunk, len(in))])
				}
				for i := 0; i < min(count, maxParams); i++ {
					if term.params[i] != want[i] || term.subParam(i) != (separator == ":" && i > 0) {
						t.Fatalf("count=%d separator=%q chunk=%d slot=%d: value=%d sub=%v, want %d", count, separator, chunk, i, term.params[i], term.subParam(i), want[i])
					}
				}
			}
		}
	}
}

func TestCSIColonAtLastSlotDoesNotBecomeMode(t *testing.T) {
	prefix := "\x1b[?" + strings.Repeat("999;", maxParams-2) + "999"
	for _, c := range []struct {
		separator string
		visible   bool
	}{{":", true}, {";", false}} {
		in := []byte(prefix + c.separator + "25l")
		for cut := 0; cut <= len(in); cut++ {
			term := New(12, 3)
			term.Write(in[:cut])
			restored := New(12, 3)
			restored.Write(term.Repaint(true))
			term.Write(in[cut:])
			restored.Write(in[cut:])
			if term.Cursor().Visible != c.visible || restored.Cursor().Visible != c.visible {
				t.Fatalf("separator=%q cut=%d: cursor visibility %v/%v, want %v", c.separator, cut, term.Cursor().Visible, restored.Cursor().Visible, c.visible)
			}
		}
	}
	term := New(12, 3)
	term.Write([]byte(prefix + ":25l\x1b[?25l"))
	if term.Cursor().Visible {
		t.Fatal("a new CSI retained an old colon bit")
	}
}

func TestCSIParameterBoundaryBehavior(t *testing.T) {
	for _, c := range []struct {
		name, in  string
		wantFg    Color
		wantX     int
		wantReply string
	}{
		{"saturated-column", "\x1b[65536G", 0, 11, ""},
		{"overflowed-column", "\x1b[999999999999999999999999G", 0, 11, ""},
		{"last-sgr-slot", "\x1b[" + strings.Repeat("0;", maxParams-1) + "31m", IndexedColor(1), 0, ""},
		{"extra-sgr-slot-ignored", "\x1b[" + strings.Repeat("0;", maxParams-1) + "31;32m", IndexedColor(1), 0, ""},
		{"colon-rgb-saturation", "\x1b[38:2::65536:65535:65537m", RGBColor(255, 255, 255), 0, ""},
		{"semicolon-rgb-saturation", "\x1b[38;2;65536;65535;65537m", RGBColor(255, 255, 255), 0, ""},
		{"saturated-mode-reply", "\x1b[?999999999999999999999999$p", 0, 0, "\x1b[?65536;0$y"},
	} {
		t.Run(c.name, func(t *testing.T) {
			in := []byte(c.in)
			for cut := 0; cut <= len(in); cut++ {
				var replies, restoredReplies bytes.Buffer
				term := New(12, 3, WithReply(&replies))
				term.Write(in[:cut])
				restored := New(12, 3, WithReply(&restoredReplies))
				restored.Write(term.Repaint(true))
				if restoredReplies.Len() != 0 {
					t.Fatalf("cut=%d: repaint emitted reply %q", cut, restoredReplies.Bytes())
				}
				restored.Write(in[cut:])
				term.Write(in[cut:])
				if term.cur.pen.Fg != c.wantFg || term.Cursor().X != c.wantX || replies.String() != c.wantReply {
					t.Fatalf("cut=%d: foreground=%v x=%d reply=%q", cut, term.cur.pen.Fg, term.Cursor().X, replies.Bytes())
				}
				assertContinuationState(t, term, restored, fmt.Sprintf("cut=%d", cut))
			}
		})
	}
}

func TestCSIParameterStorageDoesNotAllocate(t *testing.T) {
	term := New(12, 3)
	in := []byte("\x1b[38:2::10:20:30m\x1b[0;1;4:3;31m\x1b[0m")
	term.Write(in)
	if allocs := testing.AllocsPerRun(100, func() { term.Write(in) }); allocs != 0 {
		t.Fatalf("CSI parsing allocated %.1f times", allocs)
	}
}
