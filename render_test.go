package phosphor

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestRenderRowsKeepUnicodeAndPalette(t *testing.T) {
	inputs := []string{
		"\x1b[?2027hक्क\x1b[?2027lX",
		"\x1b[?2027h👩‍💻\x1b[?2027lX",
		fmt.Sprintf("\x1b[?2027h\x1b[1;%dH🇧🇨\x1b[?2027l\x1b[1;1H🇦", CodepointWidth('🇦')+1),
		"\x1b[?2027h☔︎1️⃣\x1b[?2027lX",
		"\x1b[48;2;40;44;52m\x1b[2K input \x1b[0m",
		"\x1b]4;1;rgb:12/34/56\a\x1b]10;rgb:ab/cd/ef\a\x1b]11;rgb:28/2c/34\a\x1b[31mX\x1b[0mY",
		"\x1b[1;3;4:3;58;2;50;60;70mstyled\x1b]8;id=x;https://example.com\x1b\\link\x1b]8;;\x1b\\",
	}
	for _, input := range inputs {
		t.Run(fmt.Sprintf("%x", input), func(t *testing.T) {
			source := New(24, 4)
			source.Write([]byte(input))
			before := source.Snapshot()
			host := New(24, 5)
			for y := 0; y < 4; y++ {
				out := []byte(fmt.Sprintf("\x1b[%d;1H", y+1))
				out = source.AppendRowANSI(out, y)
				host.Write(out)
			}
			for y := 0; y < 4; y++ {
				for x := 0; x < 24; x++ {
					want, got := source.Cell(x, y), host.Cell(x, y)
					if want.Rune == 0 {
						want.Rune = ' '
					}
					if got.Rune == 0 {
						got.Rune = ' '
					}
					if want.Rune != got.Rune || want.Combining != got.Combining || want.Width != got.Width || want.Style.Attrs != got.Style.Attrs || want.Style.Underline != got.Style.Underline || want.Link != got.Link {
						t.Fatalf("cell%d,%d got%+v want%+v", x, y, got, want)
					}
					sp, hp := source.Palette(), host.Palette()
					for _, c := range []struct {
						a, b Color
						bg   bool
					}{{want.Style.Fg, got.Style.Fg, false}, {want.Style.Bg, got.Style.Bg, true}} {
						if sp.Resolve(c.a, c.bg) != hp.Resolve(c.b, c.bg) {
							t.Fatalf("palette at%d,%d", x, y)
						}
					}
				}
			}
			if !bytes.Equal(before, source.Snapshot()) {
				t.Fatal("rendering changed source")
			}
		})
	}
}

type renderRows struct{ rows [][]byte }

func (w *renderRows) Write(p []byte) (int, error) {
	w.rows = append(w.rows, bytes.Clone(p))
	return len(p), nil
}
func TestScrollbackOutputSurvivesRetentionAndRestore(t *testing.T) {
	for _, budget := range []int{0, 3} {
		observed := new(renderRows)
		source := New(20, 2, WithScrollback(budget), WithScrollbackOutput(observed))
		for i := 0; i < 12; i++ {
			source.Write([]byte("same\r\n"))
		}
		if len(observed.rows) != 11 {
			t.Fatalf("budget%d lost repeated rows: %d", budget, len(observed.rows))
		}
		restoredRows := new(renderRows)
		restored := New(1, 1, WithScrollbackOutput(restoredRows))
		if err := restored.RestoreSnapshot(source.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
		restored.Write([]byte("another\r\n"))
		if len(restoredRows.rows) != 1 {
			t.Fatal("restore dropped observer")
		}
		retained := new(renderRows)
		if err := source.WriteScrollback(retained); err != nil {
			t.Fatal(err)
		}
		if budget == 0 && len(retained.rows) != 0 || budget == 3 && len(retained.rows) != 3 {
			t.Fatalf("retained=%d", len(retained.rows))
		}
	}
}

type shortRenderWriter struct{}

func (shortRenderWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }
func TestWriteScrollbackStopsOnShortWrite(t *testing.T) {
	source := New(10, 1)
	source.Write([]byte("a\r\nb\r\n"))
	if err := source.WriteScrollback(shortRenderWriter{}); err != io.ErrShortWrite {
		t.Fatal(err)
	}
}

func TestScrollbackObserverDoesNotReplayResizeHistory(t *testing.T) {
	rows := new(renderRows)
	source := New(4, 2, WithScrollbackOutput(rows))
	source.Write([]byte("abcdefghijklmnopq"))
	before := len(rows.rows)
	source.Resize(4, 3)
	if len(rows.rows) != before {
		t.Fatal("resize replayed existing history")
	}
	history := New(8, 1)
	history.Write([]byte("abc\r\n\x1b[?5h"))
	saved := new(renderRows)
	history.WriteScrollback(saved)
	host := New(8, 1)
	host.Write(saved.rows[0])
	if host.Cell(0, 0).Style.Attrs&AttrInverse == 0 {
		t.Fatal("history lost reverse video")
	}
}
