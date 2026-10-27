package phosphor

import (
	"bytes"
	"fmt"
	"testing"
	"unsafe"
)

func TestMouseModesKeepIndependentBits(t *testing.T) {
	var replies bytes.Buffer
	term := New(12, 3, WithReply(&replies))
	term.Write([]byte("\x1b[?9;1000;1002;1003;1005;1006;1015;1016h"))
	m := term.Modes()
	if m.Mouse != MouseAnyEvent || m.MouseTrackingModes != 15 || m.MouseEncoding != MouseEncodingSGRPixels || m.MouseEncodingModes != 15 {
		t.Fatal(m)
	}
	term.Write([]byte("\x1b[?9l\x1b[?1005l"))
	m = term.Modes()
	if m.Mouse != MouseNone || m.MouseTrackingModes != 14 || m.MouseEncoding != MouseEncodingDefault || m.MouseEncodingModes != 14 {
		t.Fatal(m)
	}
	for _, n := range []int{9, 1000, 1002, 1003, 1005, 1006, 1015, 1016} {
		fmt.Fprintf(term, "\x1b[?%d$p", n)
		status := 1
		if n == 9 || n == 1005 {
			status = 2
		}
		want := fmt.Sprintf("\x1b[?%d;%d$y", n, status)
		if got := replies.String(); got != want {
			t.Fatalf("mode %d: reply %q, want %q", n, got, want)
		}
		replies.Reset()
	}
	if unsafe.Sizeof(*term) > 560 {
		t.Fatalf("mouse bits grew the idle Terminal to %d bytes", unsafe.Sizeof(*term))
	}
}

func TestMouseSavedModesAndReset(t *testing.T) {
	term := New(12, 3)
	term.Write([]byte("\x1b[?9;1000;1005;1016h\x1b[?9;1000;1005;1016s\x1b[?9;1000;1005;1016l\x1b[?9;1000;1005;1016r"))
	want := term.Modes()
	if want.MouseTrackingModes != MouseTrackingX10|MouseTrackingNormal || want.Mouse != MouseNormal || want.MouseEncodingModes != MouseEncodingModeUTF8|MouseEncodingModeSGRPixels || want.MouseEncoding != MouseEncodingSGRPixels {
		t.Fatal(want)
	}
	term.Write([]byte("\x1b[!p"))
	if got := term.Modes(); got != want {
		t.Fatalf("DECSTR changed mouse modes: %+v", got)
	}
	term.Write([]byte("\x1bc"))
	if got := term.Modes(); got != (Modes{Autowrap: true}) {
		t.Fatalf("RIS kept mouse modes: %+v", got)
	}
}

func TestMouseRepaintEveryReachableSelection(t *testing.T) {
	for _, numbers := range [][4]int{mouseTrackingNumbers, mouseEncodingNumbers} {
		for bits := uint16(0); bits <= mouseModeBits; bits++ {
			for selected := uint16(0); selected <= 4; selected++ {
				state := bits | selected<<4
				if !validMouseModes(state) {
					continue
				}
				term := New(12, 3)
				for i, mode := range numbers {
					if bits&(1<<i) != 0 {
						fmt.Fprintf(term, "\x1b[?%dh", mode)
					}
				}
				if selected != 0 {
					fmt.Fprintf(term, "\x1b[?%dh", numbers[selected-1])
				} else if bits != 0 {
					for i, mode := range numbers {
						if bits&(1<<i) == 0 {
							fmt.Fprintf(term, "\x1b[?%dl", mode)
							break
						}
					}
				}
				if term.mouse != state && term.mouseEnc != state {
					t.Fatalf("setup failed for bits=%x selected=%d", bits, selected)
				}
				// Saved false/true bits may transiently select other modes.
				fmt.Fprintf(term, "\x1b[?%ds\x1b[?%ds", numbers[0], numbers[3])
				restored := New(12, 3)
				restored.Write(term.Repaint(false))
				if got, want := restored.Modes(), term.Modes(); got != want {
					t.Fatalf("bits=%x selected=%d: got %+v want %+v", bits, selected, got, want)
				}
				fmt.Fprintf(term, "\x1b[?%dr\x1b[?%dr", numbers[3], numbers[0])
				fmt.Fprintf(restored, "\x1b[?%dr\x1b[?%dr", numbers[3], numbers[0])
				if restored.Modes() != term.Modes() {
					t.Fatalf("saved-mode continuation differs for bits=%x selected=%d", bits, selected)
				}
			}
		}
	}
}

func TestResizeTabsResetOnlyOnWidthChange(t *testing.T) {
	term := New(20, 3)
	term.Write([]byte("\x1b[3g\x1b[5G\x1bH\r\t"))
	if term.Cursor().X != 4 {
		t.Fatal(term.Cursor())
	}
	term.Resize(20, 4)
	term.Write([]byte("\r\t"))
	if term.Cursor().X != 4 {
		t.Fatal("height-only resize lost custom stop", term.Cursor())
	}
	for _, cols := range []int{30, 10} {
		term.Resize(cols, 4)
		term.Write([]byte("\r\t"))
		if term.Cursor().X != 8 || term.tabs() != nil {
			t.Fatalf("width %d did not restore implicit defaults: %+v", cols, term.Cursor())
		}
	}
}
