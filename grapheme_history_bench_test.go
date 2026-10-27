package phosphor

import (
	"fmt"
	"strings"
	"testing"
)

func richGraphemeHistory() *Terminal {
	term := New(20, 2)
	term.Write([]byte("\x1b[?2027h"))
	for i := range 8 {
		term.Write([]byte(fmt.Sprintf("\x1b]8;;https://example.com/%d/%s\x1b\\❤️", i, strings.Repeat("a", 3500))))
	}
	term.Write([]byte("\x1b]8;;\x1b\\\r\nX\r\nY"))
	return term
}

func TestRichGraphemeHistory(t *testing.T) {
	term := richGraphemeHistory()
	if term.ScrollbackLen() != 1 {
		t.Fatalf("history lines%d", term.ScrollbackLen())
	}
	rows := term.ScrollbackCells(0)
	for i := range 8 {
		c := rows[0][2*i]
		if c.Rune != '❤' || c.Combining != "️" || c.Width != 2 || !strings.Contains(c.Link.URI, fmt.Sprintf("/%d/", i)) {
			t.Fatalf("cell%d:%+v", i, c)
		}
	}
	if term.ScrollbackLine(0) != strings.Repeat("❤️", 8) {
		t.Fatalf("text%q", term.ScrollbackLine(0))
	}
}

func BenchmarkGraphemeRichHistoryDecode(b *testing.B) {
	term := richGraphemeHistory()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = term.ScrollbackCells(0)
	}
	b.ReportMetric(float64(term.sbBytes), "history-B")
}
