package phosphor

import (
	"strings"
	"testing"
)

func TestFullClusterStillChangesGeometry(t *testing.T) {
	wide := "☔" + strings.Repeat("\u0301", 62) + "\u200d☔"
	narrow := "❤" + strings.Repeat("\u0301", 63) + "\u200d"
	for _, c := range []struct {
		mode, base, overflow string
		width                int
	}{
		{"\x1b[?2027h", wide, "\ufe0e", 1},
		{"\x1b[?2027h", narrow, "☔", 2},
		{"", narrow, "\x1b[?2027h☔", 2},
		{"\x1b[?2027h", wide, "\ufe0e\ufe0f\ufe0e", 1},
	} {
		term := New(8, 3)
		term.Write([]byte(c.mode + c.base))
		term.Write([]byte(c.overflow))
		if got := term.Cell(0, 0); got.Text() != c.base || got.Width != c.width || term.Cursor().X != c.width {
			t.Fatalf("fullcluster overflow%q: cell%+v cursor%+v", c.overflow, got, term.Cursor())
		}
		var binary Terminal
		if err := binary.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
		binary.Compact()
		assertSnapshotState(t, term, &binary)
		painted := New(8, 3)
		painted.Write(term.Repaint(false))
		if painted.Cell(0, 0) != term.Cell(0, 0) || painted.Cursor() != term.Cursor() {
			t.Fatalf("fullcluster geometry repaint: cell%+v/%+v cursor%+v/%+v", painted.Cell(0, 0), term.Cell(0, 0), painted.Cursor(), term.Cursor())
		}
		for _, suffix := range []string{"\u0302", "\x1b[?2027lX"} {
			term.Write([]byte(suffix))
			binary.Write([]byte(suffix))
			painted.Write([]byte(suffix))
			assertSnapshotState(t, term, &binary)
			for y := range 3 {
				for x := range 8 {
					if term.Cell(x, y) != painted.Cell(x, y) {
						t.Fatalf("fullcluster continued cell%d,%d changed", x, y)
					}
				}
			}
		}
	}
}
