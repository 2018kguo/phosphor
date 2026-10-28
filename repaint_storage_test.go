package phosphor

import (
	"fmt"
	"testing"
)

// The CI token crasher enters 47, leaves with 1049, then resizes. Resize
// releases the empty inactive ALT; repaint must visit ALT to set mode 47.
func TestRepaintImplicitAlternateContinuation(t *testing.T) {
	for _, suffix := range []string{"", "\x1b8X", "\x1b[?47h", "\x1b[?47h\x1b8X", "\x1b[?1047h\x1b8X", "\x1b[?1049h\x1b8X\x1b[?1049l\x1b8Y"} {
		t.Run(fmt.Sprintf("%q", suffix), func(t *testing.T) {
			source := New(3, 1, WithScrollback(20))
			source.Write([]byte("\x1b[?47h\x1b[?1049l"))
			source.Resize(15, 7)
			replay := New(15, 7, WithScrollback(20))
			replay.Write(source.Repaint(false))
			if source.alt != nil || replay.alt == nil {
				t.Fatal("expected implicit source ALT and allocated replay ALT")
			}
			check := func() {
				t.Helper()
				if source.Modes() != replay.Modes() || source.Cursor() != replay.Cursor() || !snapshotCursorsEqual(source, replay, source.cur, replay.cur) {
					t.Fatal("modes or current cursor differ")
				}
				for i := range 2 {
					var a, b savedCursor
					if source.saved != nil {
						a = source.saved.screen[i]
					}
					if replay.saved != nil {
						b = replay.saved.screen[i]
					}
					if a.valid != b.valid || !snapshotCursorsEqual(source, replay, a.cursor, b.cursor) {
						t.Fatalf("saved slot %d differs", i)
					}
				}
				for y := range source.rows {
					if source.Wrapped(y) != replay.Wrapped(y) {
						t.Fatalf("row %d wrap differs", y)
					}
					for x := range source.cols {
						if source.Cell(x, y) != replay.Cell(x, y) {
							t.Fatalf("cell %d,%d differs", x, y)
						}
					}
				}
			}
			check()
			source.Write([]byte(suffix))
			replay.Write([]byte(suffix))
			check()
			source.Resize(9, 4)
			replay.Resize(9, 4)
			source.Write([]byte("\x1b[?47h\x1b8A\x1b[?47l\x1b8B"))
			replay.Write([]byte("\x1b[?47h\x1b8A\x1b[?47l\x1b8B"))
			check()
		})
	}
}
