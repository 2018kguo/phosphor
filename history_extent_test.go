package phosphor

import "testing"

func TestHistoryNarrowedStyledTailExtent(t *testing.T) {
	term := New(3, 2)
	term.Write([]byte("\x1b[?2027hA\x1b[31;4m\x1b]8;;https://narrow\x1b\\☔︎\x1b]8;;\x1b\\\x1b[0m\x1b[K\r\nX\r\nY"))
	rows := term.ScrollbackCells(0)
	if len(rows) != 1 || rows[0][1].Text() != "☔︎" || rows[0][2] != (Cell{Width: 1}) {
		t.Fatalf("history invented a styled tail: %+v", rows)
	}
	term.Resize(2, 2)
	rows = term.ScrollbackCells(0)
	if len(rows) != 1 || rows[0][0].Text() != "A" || rows[0][1].Text() != "☔︎" || rows[0][1].Width != 1 {
		t.Fatalf("history invented an extra reflow row: %+v", rows)
	}
}
