package phosphor

import (
	"bytes"
	"fmt"
	"testing"
	"unsafe"
)

func TestCursorLifetimeCheckpoint(t *testing.T) {
	for _, input := range []string{
		"\x1b]8;;https://old\aA\x1b7\x1b]8;;\a\x1b8B",
		"\x1b[2;2H\x1b7\x1b[3;3H\x1b[?1049h\x1b[?1049l\x1b8X",
		"\x1b]8;;https://main\aA\x1b[?47h\x1b]8;;https://alt\aB\x1b[?47lC",
		"\x1b*A\x1bN\x1b7#\x1b8#",
		"\x1b*A\x1bN\x1b7",
		"\x1b*A\x1bN\x1b[?1049h\x1b7",
		"\x1b[?47h\x1b[2;2H\x1b[?1049h\x1b[3;3H\x1b[?1049h",
		"\x1b[?1049hALT\x1b[?1049l",
	} {
		t.Run(fmt.Sprintf("%q", input), func(t *testing.T) {
			source := New(8, 4)
			source.Write([]byte(input))
			source.Compact()
			var snapshot Terminal
			if err := snapshot.RestoreSnapshot(source.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			repaint := New(8, 4)
			repaint.Write(source.Repaint(true))
			for _, next := range []string{"", "\x1b8#", "\x1b[?47l\x1b8M", "\x1b[?47h\x1b8A", "\x1b[?1049l\x1b8P"} {
				for _, term := range []*Terminal{source, &snapshot, repaint} {
					term.Write([]byte(next))
				}
				assertSnapshotState(t, source, &snapshot)
				for y := range 4 {
					for x := range 8 {
						if source.Cell(x, y) != repaint.Cell(x, y) {
							t.Fatalf("after %q cell %d,%d: %+v / %+v", next, x, y, source.Cell(x, y), repaint.Cell(x, y))
						}
					}
				}
				if source.Cursor() != repaint.Cursor() {
					t.Fatalf("after %q cursor", next)
				}
			}
		})
	}
}

func TestIndependentScreenModesAndAllSavedModes(t *testing.T) {
	var replies bytes.Buffer
	term := New(8, 4, WithReply(&replies))
	term.Write([]byte("\x1b[?47h\x1b[?1049h\x1b[?1049l\x1b[?47$p\x1b[?1047$p\x1b[?1049$p"))
	if want := "\x1b[?47;1$y\x1b[?1047;2$y\x1b[?1049;2$y"; replies.String() != want {
		t.Fatalf("%q", replies.String())
	}
	for _, mode := range savedModeNumbers {
		if _, ok := term.modeValue(mode); !ok {
			t.Fatalf("unknown canonical saved mode %d", mode)
		}
		term.Write(fmt.Appendf(nil, "\x1b[?%ds", mode))
	}
	if len(term.x.savedModes) != len(savedModeNumbers) {
		t.Fatal("saved mode inventory truncated")
	}
	var restored Terminal
	if err := restored.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	assertSnapshotState(t, term, &restored)
	if unsafe.Sizeof(cursor{}) != 40 || unsafe.Sizeof(Terminal{}) > 560 || unsafe.Sizeof(cell{}) != 16 {
		t.Fatal("cursor changes enlarged idle storage")
	}
}

func TestSnapshotRetainsResizedInactiveAlternate(t *testing.T) {
	source := New(8, 3)
	source.Write([]byte("\x1b[?47h\x1b[?1049h\x1b[41m\x1b[?1047l"))
	source.Resize(8, 5)
	var restored Terminal
	if err := restored.RestoreSnapshot(source.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	for _, term := range []*Terminal{source, &restored} {
		term.Write([]byte("\x1b[?47h"))
	}
	assertSnapshotState(t, source, &restored)
	if source.Cell(0, 0).Style.Bg != IndexedColor(1) || source.Cell(0, 4).Style.Bg != 0 {
		t.Fatal("resize did not retain mixed old/new backgrounds")
	}
}
