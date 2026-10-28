package phosphor

import "testing"

func TestSavedPendingRepaintKeepsExistingWrappedRow(t *testing.T) {
	for _, input := range []string{"abcd\x1b[?47h\x1b7X", "abcd\x1b[?47h\x1b7XY\x1b[41m\x1b[2;3H\x1b[X"} {
		source := New(4, 3, WithScrollback(20))
		source.Write([]byte(input))
		restored := New(4, 3, WithScrollback(20))
		restored.Write(source.Repaint(true))
		for _, next := range []string{"", "\x1b8Z", "\r\nnext"} {
			source.Write([]byte(next))
			restored.Write([]byte(next))
			if source.Cursor() != restored.Cursor() {
				t.Fatalf("input%q next%q cursor%+v/%+v", input, next, source.Cursor(), restored.Cursor())
			}
			for y := range 3 {
				if source.Wrapped(y) != restored.Wrapped(y) {
					t.Fatalf("input%q next%q row%d lostwrap", input, next, y)
				}
				for x := range 4 {
					if source.Cell(x, y) != restored.Cell(x, y) {
						t.Fatalf("input%q next%q cell%d,%d changed", input, next, x, y)
					}
				}
			}
		}
	}
}
