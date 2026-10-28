package phosphor

import "testing"

func TestSemanticPromptFreshLine(t *testing.T) {
	for _, command := range []string{"L", "A", "N", "A;redraw=1", "N;aid=abc"} {
		for _, c := range []struct {
			prefix  string
			x, y    int
			history int
		}{
			{"", 0, 0, 0}, {"Hello", 0, 1, 0}, {"12345678", 0, 1, 0},
			{"\x1b[3;1Htext", 0, 2, 1},
			{"\x1b[?69h\x1b[3;7s\x1b[2;3H", 2, 1, 0},
			{"\x1b[?69h\x1b[3;7s\x1b[2;4H", 2, 2, 0},
			{"\x1b[?69h\x1b[3;7s\x1b[2;1H", 0, 1, 0},
			{"\x1b[?69h\x1b[3;7s\x1b[2;2H", 0, 2, 0},
		} {
			term := New(8, 3, WithScrollback(20))
			term.Write([]byte(c.prefix))
			input := []byte("\x1b]133;" + command + "\x1b\\")
			for _, b := range input {
				term.Write([]byte{b})
			}
			cur := term.Cursor()
			if cur.X != c.x || cur.Y != c.y || term.ScrollbackLen() != c.history {
				t.Fatalf("%q after%q: cursor%+v history%d", command, c.prefix, cur, term.ScrollbackLen())
			}
			var dst Terminal
			if err := dst.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			dst.Compact()
			assertSnapshotState(t, term, &dst)
		}
	}
}

func TestSemanticPromptNonmovingMarkers(t *testing.T) {
	for _, cmd := range []string{"P", "B", "I", "C", "D", "Lextra", "Aextra", "Nextra", ""} {
		term := New(8, 3)
		term.Write([]byte("abc\x1b]133;" + cmd + "\a"))
		if cur := term.Cursor(); cur.X != 3 || cur.Y != 0 {
			t.Fatalf("%q moved cursor%+v", cmd, cur)
		}
	}
}
