package phosphor

import "testing"

func TestRepaintNarrowedClusterAtRightEdge(t *testing.T) {
	for _, tc := range []struct {
		prefix string
		cols   int
	}{
		{"A", 2}, {"AB", 3}, {"A界", 4},
		{"\x1b[32mA\x1b[0m", 2},
	} {
		for _, history := range []bool{false, true} {
			term := New(tc.cols+1, 4)
			term.Write([]byte("\x1b[?2027h" + tc.prefix + "\x1b[31;4m\x1b]8;;https://narrow\x1b\\☔︎\x1b]8;;\x1b\\\x1b[0m\x1b[K"))
			if history {
				term.Write([]byte("\r\n1\r\n2\r\n3\r\n4"))
			}
			term.Resize(tc.cols, 4)
			term.Write([]byte("\x1b[3;1H"))
			restored := New(tc.cols, 4)
			restored.Write(term.Repaint(true))
			assertSparseObservations(t, term, restored)
			for _, suffix := range []string{"Z", "\x1b8Q", "\r\nnext\r\nlast"} {
				term.Write([]byte(suffix))
				restored.Write([]byte(suffix))
				assertSparseObservations(t, term, restored)
			}
		}
	}
}

func TestRepaintNarrowedEdgeKeepsSoftWrap(t *testing.T) {
	term := New(3, 4)
	term.Write([]byte("\x1b[?2027hA☔︎Q"))
	term.Resize(2, 4)
	restored := New(2, 4)
	restored.Write(term.Repaint(true))
	assertSparseObservations(t, term, restored)
	for y := range term.rows {
		if rowFlagsAt(term.mainWrap, y) != rowFlagsAt(restored.mainWrap, y) {
			t.Fatalf("row %d soft wrap changed", y)
		}
	}
}

func TestRepaintNarrowedEdgeScratchLimits(t *testing.T) {
	for _, cols := range []int{1, 2} {
		term := New(4, 4)
		term.Write([]byte("\x1b[?2027h☔︎☔︎"))
		term.Resize(cols, 4)
		if term.canStageNarrowEdge(term.main[0]) {
			t.Fatal("narrowed clusters filling the row have no two-cell scratch area")
		}
	}
}

func TestRepaintNarrowedStyledTailBoundary(t *testing.T) {
	term := New(3, 4)
	term.Write([]byte("\x1b[?2027hA\x1b[31;4m\x1b]8;;https://tail\x1b\\☔︎\x1b]8;;\x1b\\\x1b[0m"))
	term.Resize(2, 4)
	term.Write([]byte("\x1b[3;1H"))
	want := term.Cell(0, 1)
	if want.Rune != 0 || want.Style.Fg != IndexedColor(1) || want.Link.URI != "https://tail" {
		t.Fatalf("fixture lost its styled empty tail: %+v", want)
	}
	replay := New(2, 4)
	replay.Write(term.Repaint(true))
	// ECH creates background-only empty cells. This relocated foreground,
	// underline and link still use the formatter's styled-space fallback.
	if got := replay.Cell(0, 1); got.Rune != ' ' || got.Style != want.Style || got.Link != want.Link {
		t.Fatalf("styled-empty VT boundary changed: %+v / %+v", got, want)
	}
	exact := New(2, 4)
	if err := exact.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	assertSnapshotState(t, term, exact)
}
