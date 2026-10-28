package phosphor

import "testing"

func TestResizeToOneColumnDestroysWideCellMetadata(t *testing.T) {
	for _, style := range []string{
		"\x1b[31;44;1m",
		"\x1b[4;58;5;9m",
		"\x1b]8;id=x;https://x\x1b\\",
		"\x1b]8;;https://x\x1b\\\x1b[4:3;58;2;1;2;3m",
	} {
		term := New(4, 3)
		term.Write([]byte(style + "界"))
		term.Resize(1, 3)
		if c := term.Cell(0, 0); c.Text() != "" || c.Style != (Style{}) || c.Link.URI != "" || c.Width != 1 {
			t.Fatalf("style %q survived native wide clipping: %+v", style, c)
		}
		if cur := term.Cursor(); cur.X != 0 || cur.Y != 1 || cur.PendingWrap || !term.Wrapped(1) {
			t.Fatalf("style %q lost native clipped-wide cursor mapping: %+v", style, cur)
		}
		term.Compact()
		copy := New(1, 3)
		if err := copy.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
		for _, e := range []*Terminal{term, copy} {
			e.Write([]byte("\r\nX"))
			if e.Cell(0, 2).Text() != "X" || e.Cursor().Y != 2 {
				t.Fatal("clipped-wide snapshot changed future cursor behavior")
			}
		}
	}
}

func TestResizeToOneColumnKeepsNarrowedClusterMetadata(t *testing.T) {
	term := New(4, 3)
	term.Write([]byte("\x1b]8;;https://x\x1b\\\x1b[4:3;58;2;1;2;3m\x1b[?2027h☔︎"))
	want := term.Cell(0, 0)
	term.Resize(1, 3)
	if got := term.Cell(0, 0); got != want {
		t.Fatalf("narrowed cluster lost metadata: got %+v want %+v", got, want)
	}
}
