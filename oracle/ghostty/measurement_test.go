//go:build ghostty

package main

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/2018kguo/phosphor"
)

const (
	measurementCols  = 64
	measurementRows  = 8
	measurementLines = 128
	measurementMode  = "\x1b[?2027;1$y"
)

func measurementTitle(i int) string {
	prefix := fmt.Sprintf("burst-%03d-", i)
	return prefix + strings.Repeat("t", 512-len(prefix))
}

func measurementURI(i int) string {
	return fmt.Sprintf("https://example.test/burst/%03d", i)
}

func measurementText(i int) string {
	// Sixty ASCII cells, a two-cell ZWJ cluster and one combining cell.
	return fmt.Sprintf("LINE%03d ", i) + strings.Repeat(string(rune('A'+i%26)), 52) + "👩‍💻e\u0301"
}

func measurementBurst() []byte {
	var b strings.Builder
	b.WriteString("\x1b[?2027h\x1b[?2027$p")
	for i := range measurementLines {
		mode := 'l'
		if i%2 == 1 {
			mode = 'h'
		}
		fmt.Fprintf(&b, "\x1b]2;%s\x1b\\\x1b[?2004%c\x1b[?2026h\x1b[%dm\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\\x1b[0m\r\n\x1b[?2026l\x1b[5n",
			measurementTitle(i), mode, 30+i%8, measurementURI(i), measurementText(i))
	}
	// The parser must still retain unfinished input when the large call returns.
	b.WriteString("\x1b[3")
	return []byte(b.String())
}

// verifyMeasurementBurst has independent expectations, so matching final
// screens or two equally lossy feed paths cannot satisfy the witness.
func verifyMeasurementBurst(s modelState) error {
	wantReplies := measurementMode + strings.Repeat("\x1b[0n", measurementLines)
	if s.Replies != wantReplies {
		return fmt.Errorf("reply bytes = %d, want %d with all %d DSR responses", len(s.Replies), len(wantReplies), measurementLines)
	}
	if s.Cols != measurementCols || s.Rows != measurementRows || s.Alt || s.Title != measurementTitle(measurementLines-1) {
		return fmt.Errorf("geometry, alternate screen or final title changed")
	}
	if s.Cursor != (modelCursor{X: 0, Y: measurementRows - 1, Shape: 1, Visible: true}) || s.Kitty != 0 || s.Mouse != 0 || s.MouseEncoding != 0 {
		return fmt.Errorf("final cursor or input mode changed: %+v", s.Cursor)
	}
	if s.Modes != ([24]bool{2: true, 4: true}) {
		return fmt.Errorf("final modes = %v", s.Modes)
	}
	if len(s.History) != measurementLines+1-measurementRows || len(s.Screen) != measurementRows {
		return fmt.Errorf("history/screen rows = %d/%d, want %d/%d; output was lost or pruned", len(s.History), len(s.Screen), measurementLines+1-measurementRows, measurementRows)
	}
	rows := append(slices.Clone(s.History), s.Screen...)
	for i, row := range rows {
		if len(row) != measurementCols {
			return fmt.Errorf("row %d has %d cells", i, len(row))
		}
		ascii := ""
		if i < measurementLines {
			ascii = fmt.Sprintf("LINE%03d ", i) + strings.Repeat(string(rune('A'+i%26)), 52)
		}
		for x, got := range row {
			want := modelCell{Width: 1}
			if i < measurementLines && x < measurementCols-1 {
				want.Fg, want.URI = uint32(phosphor.IndexedColor(uint8(i%8))), measurementURI(i)
				switch x {
				case 60:
					want.Text, want.Width = "👩‍💻", 2
				case 61:
					want.Width = 0
				case 62:
					want.Text = "e\u0301"
				default:
					want.Text = string(ascii[x])
				}
			}
			if got != want {
				return fmt.Errorf("row %d cell %d = %+v, want %+v", i, x, got, want)
			}
		}
	}
	for _, flags := range [][]bool{s.Wrapped, s.HistoryWrapped} {
		for _, wrapped := range flags {
			if wrapped {
				return fmt.Errorf("unwrapped numbered line was marked as wrapped")
			}
		}
	}
	for _, prompt := range s.Prompt {
		if prompt != 0 {
			return fmt.Errorf("unmarked output acquired a prompt annotation")
		}
	}
	return nil
}

func TestMeasurementBurstParsesEveryChunk(t *testing.T) {
	input := measurementBurst()
	if len(input) <= 64<<10 || len(input) > 1<<20 {
		t.Fatalf("burst is %d bytes; witness must exceed 64 KiB within trace input limits", len(input))
	}
	t.Logf("burst: %d input bytes, %d numbered lines, %d total physical rows", len(input), measurementLines, measurementLines+1)
	for _, engine := range []string{"phosphor", "ghostty"} {
		t.Run(engine, func(t *testing.T) {
			var whole *modelState
			for _, chunk := range []int{0, 7, 1024, 64 << 10} {
				t.Run(fmt.Sprintf("chunk=%d", chunk), func(t *testing.T) {
					e := newTracedEngine(engine, measurementCols, measurementRows)
					defer e.Close()
					traceWrite(e, input, chunk)
					state := traceState(t, e)
					if err := verifyMeasurementBurst(state); err != nil {
						t.Fatal(err)
					}
					if whole == nil {
						whole = &state
					} else if diff := differences(*whole, state); len(diff) != 0 {
						t.Fatalf("chunking changed state: %+v", diff)
					}
					// Capture and restore the unfinished CSI, then prove it still
					// applies to future output without repeating prior replies.
					e.replace(e.Snapshot(), measurementCols, measurementRows, true)
					if diff := differences(state, traceState(t, e)); len(diff) != 0 {
						t.Fatalf("snapshot changed burst state: %+v", diff)
					}
					e.Write([]byte("1mTAIL\x1b[0m\x1b[6n"))
					want := state
					want.Screen = slices.Clone(state.Screen)
					want.Screen[measurementRows-1] = slices.Clone(state.Screen[measurementRows-1])
					for x, c := range "TAIL" {
						want.Screen[measurementRows-1][x] = modelCell{Text: string(c), Width: 1, Fg: uint32(phosphor.IndexedColor(1))}
					}
					want.Cursor.X = 4
					want.Replies += "\x1b[8;5R"
					if diff := differences(want, traceState(t, e)); len(diff) != 0 {
						t.Fatalf("unfinished input or future replies changed: %+v", diff)
					}
				})
			}
			t.Run("discarded-prefix-is-rejected", func(t *testing.T) {
				e := newTracedEngine(engine, measurementCols, measurementRows)
				defer e.Close()
				e.Write(input[len(input)-(64<<10):])
				if err := verifyMeasurementBurst(traceState(t, e)); err == nil {
					t.Fatal("witness accepted a burst after discarding its prefix")
				} else {
					t.Logf("discarding the prefix is detected: %v", err)
				}
			})
		})
	}
}
