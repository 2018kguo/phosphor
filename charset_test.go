package phosphor

import "testing"

func TestBritishCharacterSet(t *testing.T) {
	for _, c := range []struct{ input, want string }{
		{"\x1b(A#q\x1b(B#", "£q#"},
		{"\x1b*A#\x1bN\u0301#", "#́#"},
		{"\x1b(0\x1b*A\x1bN#q\x1b[2b\x1b(B\x1b[2b", "£───qq"},
		{"\x1b+A\x1bO##\x1bo#\x1b(B\x0f#", "£#£#"},
		{"\x1b)A\x0e#\x1b7\x1b)B#\x1b8#", "££"},
	} {
		term := New(20, 4)
		for i := range len(c.input) {
			term.Write([]byte(c.input[i : i+1]))
		}
		if got := term.Line(0); got != c.want {
			t.Errorf("input%q got%q want%q", c.input, got, c.want)
		}
	}
}

func TestCharacterSetContinuation(t *testing.T) {
	for _, prefix := range []string{
		"\x1b*A\x1bN", "\x1b+A\x1bO", "\x1b)A\x0e\x1b7\x1b)B",
		"\x1b(A#\x1b7\x1b[?1049h\x1b*A\x1bN#\x1b7",
	} {
		term := New(20, 4, WithScrollback(20))
		term.Write([]byte(prefix))
		for _, binary := range []bool{false, true} {
			dst := New(20, 4, WithScrollback(20))
			if binary {
				if err := dst.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
					t.Fatal(err)
				}
			} else {
				dst.Write(term.Repaint(true))
			}
			source := New(1, 1)
			if err := source.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			for _, next := range []string{"#q", "\x1b8#", "\x1b[?1049l\x1b8#", "\x1b[!p#"} {
				source.Write([]byte(next))
				dst.Write([]byte(next))
				dst.Compact()
				if source.Cursor() != dst.Cursor() || source.Modes() != dst.Modes() {
					t.Fatalf("charset continuation cursor/modes changed binary%v prefix%q next%q got%+v want%+v modes%+v/%+v saved%+v/%+v", binary, prefix, next, dst.Cursor(), source.Cursor(), dst.Modes(), source.Modes(), dst.saved, source.saved)
				}
				for y := range 4 {
					for x := range 20 {
						if source.Cell(x, y) != dst.Cell(x, y) {
							t.Fatalf("charset cell%d,%d binary%v prefix%q next%q got%+v want%+v", x, y, binary, prefix, next, dst.Cell(x, y), source.Cell(x, y))
						}
					}
				}
			}
		}
	}
}
