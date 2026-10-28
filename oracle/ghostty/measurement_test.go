//go:build ghostty

package main

import (
	"encoding/json"
	"fmt"
	"github.com/2018kguo/phosphor"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
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
	return verifyMeasurementBurstModel(s, historyBudgeted)
}

func verifyMeasurementBurstModel(s modelState, history historyPolicy) error {
	if s.Cols != measurementCols || s.Rows != measurementRows || s.Alt || s.Title != measurementTitle(measurementLines-1) {
		return fmt.Errorf("geometry, alternate screen or final title changed")
	}
	if s.Cursor != (modelCursor{X: 0, Y: measurementRows - 1, Shape: 1, Visible: true}) || s.Kitty != 0 || s.Mouse != 0 || s.MouseEncoding != 0 {
		return fmt.Errorf("final cursor or input mode changed: %+v", s.Cursor)
	}
	if s.Modes != ([24]bool{2: true, 4: true}) {
		return fmt.Errorf("final modes = %v", s.Modes)
	}
	wantHistory, firstRow := measurementLines+1-measurementRows, 0
	if history == historyNone {
		wantHistory, firstRow = 0, measurementLines+1-measurementRows
	}
	if len(s.History) != wantHistory || len(s.Screen) != measurementRows {
		return fmt.Errorf("history/screen rows = %d/%d, want %d/%d; output was lost or pruned", len(s.History), len(s.Screen), wantHistory, measurementRows)
	}
	rows := append(slices.Clone(s.History), s.Screen...)
	for offset, row := range rows {
		i := firstRow + offset
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

func TestMeasurementTimedConstructorParsesEveryChunk(t *testing.T) {
	input := measurementBurst()
	for _, engine := range []string{"phosphor", "ghostty"} {
		for _, history := range []historyPolicy{historyBudgeted, historyNone} {
			t.Run(fmt.Sprintf("%s/%s", engine, history), func(t *testing.T) {
				for _, chunk := range []int{0, 7, 1024, 64 << 10} {
					t.Run(fmt.Sprintf("chunk=%d", chunk), func(t *testing.T) {
						e := newEngineWithHistory(engine, measurementCols, measurementRows, history)
						defer e.Close()
						// This wrapper only reads the existing engine. It installs no
						// reply collector and enables no continuation bookkeeping.
						observed := &tracedEngine{Engine: e}
						traceWrite(observed, input, chunk)
						state := traceState(t, observed)
						if state.Replies != "" {
							t.Fatal("timed constructor unexpectedly retained replies")
						}
						if err := verifyMeasurementBurstModel(state, history); err != nil {
							t.Fatal(err)
						}
						// Complete the CSI left unfinished by the large Write. A
						// correct final picture alone cannot prove parser state survived.
						e.Write([]byte("1mTAIL\x1b[0m\x1b[6n"))
						want := state
						want.Screen = slices.Clone(state.Screen)
						want.Screen[measurementRows-1] = slices.Clone(state.Screen[measurementRows-1])
						for x, c := range "TAIL" {
							want.Screen[measurementRows-1][x] = modelCell{Text: string(c), Width: 1, Fg: uint32(phosphor.IndexedColor(1))}
						}
						want.Cursor.X = 4
						if diff := differences(want, traceState(t, observed)); len(diff) != 0 {
							t.Fatalf("timed constructor lost unfinished input: %+v", diff)
						}
					})
				}
				t.Run("discarded-prefix-is-rejected", func(t *testing.T) {
					e := newEngineWithHistory(engine, measurementCols, measurementRows, history)
					defer e.Close()
					e.Write(input[len(input)-(64<<10):])
					if err := verifyMeasurementBurstModel(traceState(t, &tracedEngine{Engine: e}), history); err == nil {
						t.Fatal("timed constructor witness accepted discarded input")
					} else {
						t.Logf("discarding the prefix is detected: %v", err)
					}
				})
			})
		}
	}
}

func TestMeasurementTimedNoHistoryCumulativePrefix(t *testing.T) {
	const cols, rows, moves, kept = 256, 8, 128, 8
	const setup, unfinished = "\x1b[?2027h", "\x1b[3"
	blocks := make([]string, moves)
	for i := range blocks {
		blocks[i] = fmt.Sprintf("\x1b]2;%s\x1b\\\x1b[1C", measurementTitle(i))
	}
	input := []byte(setup + strings.Join(blocks, "") + unfinished)
	lost := []byte(setup + strings.Join(blocks[moves-kept:], "") + unfinished)
	if len(input) <= 64<<10 {
		t.Fatalf("cumulative burst is %d bytes, want more than 64 KiB", len(input))
	}
	t.Logf("cumulative burst: %d bytes, %d blocks; loss control keeps %d bytes and %d complete blocks", len(input), moves, len(lost), kept)
	// Each complete block must advance the cursor, even with no retained
	// history or replies. The final title alone cannot detect missing blocks.
	check := func(s modelState, count int, completed bool) error {
		x := count
		if completed {
			x++
		}
		if s.Cols != cols || s.Rows != rows || s.Alt || len(s.History) != 0 || s.Replies != "" || s.Title != measurementTitle(moves-1) || s.Modes != ([24]bool{4: true}) {
			return fmt.Errorf("geometry, history, replies, final title or modes changed")
		}
		if s.Cursor != (modelCursor{X: x, Shape: 1, Visible: true}) {
			return fmt.Errorf("cursor %+v, want X %d after %d blocks", s.Cursor, x, count)
		}
		if len(s.Screen) != rows {
			return fmt.Errorf("screen has %d rows, want %d", len(s.Screen), rows)
		}
		for y, row := range s.Screen {
			if len(row) != cols {
				return fmt.Errorf("row %d has %d cells, want %d", y, len(row), cols)
			}
			for x, cell := range row {
				want := modelCell{Width: 1}
				if completed && y == 0 && x == count {
					want.Text, want.Fg = "X", uint32(phosphor.IndexedColor(1))
				}
				if cell != want {
					return fmt.Errorf("cell %d,%d = %+v, want %+v", x, y, cell, want)
				}
			}
		}
		return nil
	}
	for _, engine := range []string{"phosphor", "ghostty"} {
		t.Run(engine, func(t *testing.T) {
			for _, chunk := range []int{0, 7, 1024, 64 << 10} {
				t.Run(fmt.Sprintf("chunk=%d", chunk), func(t *testing.T) {
					e := newEngineWithHistory(engine, cols, rows, historyNone)
					defer e.Close()
					observed := &tracedEngine{Engine: e}
					traceWrite(observed, input, chunk)
					if err := check(traceState(t, observed), moves, false); err != nil {
						t.Fatal(err)
					}
					e.Write([]byte("1mX\x1b[0m\x1b[6n"))
					if err := check(traceState(t, observed), moves, true); err != nil {
						t.Fatalf("unfinished CSI or future output changed: %v", err)
					}
				})
			}
			t.Run("preserved-setup-and-final-title-still-detect-loss", func(t *testing.T) {
				e := newEngineWithHistory(engine, cols, rows, historyNone)
				defer e.Close()
				observed := &tracedEngine{Engine: e}
				e.Write(lost)
				state := traceState(t, observed)
				if err := check(state, kept, false); err != nil {
					t.Fatalf("loss control changed more than the block count: %v", err)
				}
				if err := check(state, moves, false); err == nil {
					t.Fatal("cumulative witness accepted 120 missing blocks")
				} else {
					t.Logf("preserving setup and final title still detects loss: %v", err)
				}
				e.Write([]byte("1mX\x1b[0m\x1b[6n"))
				if err := check(traceState(t, observed), kept, true); err != nil {
					t.Fatalf("loss control did not preserve unfinished CSI: %v", err)
				}
			})
		})
	}
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
					// applies to future output and the following cursor query.
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

func TestEqualHistoryMemoryFixture(t *testing.T) {
	a, b := newTracedEngine("phosphor", 80, 24), newTracedEngine("ghostty", 80, 24)
	defer a.Close()
	defer b.Close()
	feed(a, equalHistoryInput(), 1024)
	feed(b, equalHistoryInput(), 1024)
	before := traceState(t, a)
	if len(before.History) != 100 {
		t.Fatalf("expected exactly 100 retained rows, got %d", len(before.History))
	}
	if diff := differences(before, traceState(t, b)); len(diff) != 0 {
		t.Fatalf("memory fixtures retain different content: %+v", diff)
	}
	a.Engine.(*phos).t.Compact()
	b.Engine.(*ghost).compact()
	for _, engine := range []*tracedEngine{a, b} {
		if diff := differences(before, traceState(t, engine)); len(diff) != 0 {
			t.Fatalf("compaction changed retained content: %+v", diff)
		}
	}
}

func TestComparisonSampleMetadata(t *testing.T) {
	if os.Getenv("ORACLE_METADATA_TEST_CHILD") == "1" {
		emit(map[string]any{"kind": "test"})
		return
	}
	for _, tc := range []struct {
		repeat, position string
		valid            bool
	}{
		{"", "", true}, {"0", "0", true}, {"5", "2", true},
		{"-1", "0", false}, {"bad", "1", false}, {"0", "3", false}, {"0", "-1", false},
	} {
		t.Run(tc.repeat+"/"+tc.position, func(t *testing.T) {
			t.Setenv("ORACLE_METADATA_TEST_CHILD", "1")
			t.Setenv("COMPARE_OUTER_REPEAT", tc.repeat)
			t.Setenv("COMPARE_SAMPLE_ORDER", tc.position)
			cmd := exec.Command(os.Args[0], "-test.run=^TestComparisonSampleMetadata$")
			output, err := cmd.CombinedOutput()
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v exit=%v output=%s", tc.valid, err, output)
			}
			if !tc.valid {
				return
			}
			var record map[string]any
			if err := json.NewDecoder(strings.NewReader(string(output))).Decode(&record); err != nil {
				t.Fatal(err)
			}
			for _, field := range []struct{ key, raw string }{{"outer_repeat", tc.repeat}, {"sample_order", tc.position}} {
				value, present := record[field.key]
				if field.raw == "" {
					if present {
						t.Fatalf("unconfigured %s acquired a value", field.key)
					}
				} else {
					want, _ := strconv.Atoi(field.raw)
					if value != float64(want) {
						t.Fatalf("%s=%v, want%d", field.key, value, want)
					}
				}
			}
		})
	}
}

func TestABRunnerBalancesPositionsAndRecordsRepeats(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "fake-driver")
	// The fake driver emits only configuration: this test never measures a core.
	const script = `#!/bin/sh
printf '{"stream":"%s","label":"%s","outer_repeat":%s,"sample_order":%s}\n' "$COMPARE_STREAM" "$COMPARE_LABEL" "$COMPARE_OUTER_REPEAT" "$COMPARE_SAMPLE_ORDER"
`
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PHOSPHOR_DIR", dir)
	t.Setenv("COMPARE_REPEATS", "")
	output := filepath.Join(dir, "samples.jsonl")
	if result, err := exec.Command("bash", "ab.sh", binary, binary, output).CombinedOutput(); err != nil {
		t.Fatalf("runner: %v %s", err, result)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	type sample struct {
		Stream, Label string
		Repeat        int `json:"outer_repeat"`
		Order         int `json:"sample_order"`
	}
	positions := map[string]map[string][3]int{}
	permutations := map[string]map[int][]string{}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		var row sample
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		if row.Repeat < 0 || row.Repeat > 5 || row.Order < 0 || row.Order > 2 {
			t.Fatalf("invalid sample position: %+v", row)
		}
		if positions[row.Stream] == nil {
			positions[row.Stream] = map[string][3]int{}
			permutations[row.Stream] = map[int][]string{}
		}
		counts := positions[row.Stream][row.Label]
		counts[row.Order]++
		positions[row.Stream][row.Label] = counts
		if row.Order != len(permutations[row.Stream][row.Repeat]) {
			t.Fatal("sample positions do not preserve execution order")
		}
		permutations[row.Stream][row.Repeat] = append(permutations[row.Stream][row.Repeat], row.Label)
	}
	if len(lines) != 180 || len(positions) != 10 {
		t.Fatalf("runner emitted %d samples across %d streams", len(lines), len(positions))
	}
	for stream, labels := range positions {
		for _, label := range []string{"baseline", "phosphor", "ghostty"} {
			if labels[label] != ([3]int{2, 2, 2}) {
				t.Fatalf("%s/%s has unbalanced positions: %v", stream, label, labels[label])
			}
		}
		seen := map[string]bool{}
		for _, order := range permutations[stream] {
			seen[strings.Join(order, ",")] = true
		}
		if len(seen) != 6 {
			t.Fatalf("%s repeats an order before all six permutations", stream)
		}
	}
	t.Setenv("COMPARE_REPEATS", "2")
	if result, err := exec.Command("bash", "ab.sh", binary, binary, output).CombinedOutput(); err != nil {
		t.Fatalf("configured repeats: %v %s", err, result)
	}
	data, err = os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(strings.Split(strings.TrimSpace(string(data)), "\n")); got != 60 {
		t.Fatalf("two repeats emitted %d samples, want60", got)
	}
	for _, invalid := range []string{"0", "-1", "01", "not-a-number", "1000000000"} {
		t.Setenv("COMPARE_REPEATS", invalid)
		if err := exec.Command("bash", "ab.sh", binary, binary, output).Run(); err == nil {
			t.Fatalf("runner accepted invalid repeats %q", invalid)
		}
	}
}

func TestNativeBuildInfoLabelsExpectedSource(t *testing.T) {
	info := nativeBuildInfo()
	if info["expected_source_commit"] != pinnedGhosttyCommit {
		t.Fatal("native build info lost its expected pin")
	}
	if _, observed := info["source_commit"]; observed {
		t.Fatal("expected native source was reported as observed")
	}
	if _, verified := info["verified_checkout_commit"]; verified {
		t.Fatal("native build info reported an unverified checkout")
	}
}
