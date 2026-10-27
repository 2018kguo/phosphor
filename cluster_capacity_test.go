package phosphor

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNativeClusterCapacity(t *testing.T) {
	for _, mode := range []string{"", "\x1b[?2027h"} {
		for _, mark := range []string{"\u0301", "\U000e0100"} {
			term := New(8, 3, WithScrollback(20))
			term.Write([]byte(mode + "A" + strings.Repeat(mark, 64)))
			want := "A" + strings.Repeat(mark, 64)
			if got := term.Cell(0, 0).Text(); got != want {
				t.Fatalf("capacity mode%q mark%q: %d codepoints, want65", mode, mark, utf8.RuneCountInString(got))
			}
			term.ClearDirty()
			term.Write([]byte(mark))
			if term.Cell(0, 0).Text() != want || term.RowDirty(0) {
				t.Fatal("overflow changed a full cluster")
			}
			var restored Terminal
			if err := restored.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			assertSnapshotState(t, term, &restored)
			for _, next := range []string{"\x1b[?2027l\u0302", "\r\nnext\r\nlast\r\nmore"} {
				term.Write([]byte(next))
				restored.Write([]byte(next))
				assertSnapshotState(t, term, &restored)
			}
			term.Resize(4, 8)
			restored.Resize(4, 8)
			restored.Compact()
			assertSnapshotState(t, term, &restored)
		}
	}
}

func TestLongClusterMixedJoinModes(t *testing.T) {
	term := New(12, 3, WithScrollback(20))
	term.Write([]byte("界"))
	for i := range 64 {
		if i%2 == 0 {
			term.Write([]byte("\x1b[?2027h"))
		} else {
			term.Write([]byte("\x1b[?2027l"))
		}
		term.Write([]byte("\U000e0100"))
	}
	c := term.x.clusters[term.cellAt(0, 0).R-clusterBase]
	for i := 1; i < 65; i++ {
		if c.joinModes.has(i) != (i%2 == 1) {
			t.Fatalf("join mode%d lost", i)
		}
	}
	for _, restore := range []func(*Terminal){
		func(dst *Terminal) {
			if err := dst.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
		},
		func(dst *Terminal) { dst.Write(term.Repaint(false)) },
	} {
		dst := New(12, 3, WithScrollback(20))
		restore(dst)
		if dst.Cell(0, 0) != term.Cell(0, 0) {
			t.Fatal("long mixed-mode cluster changed on replay")
		}
		dst.Write([]byte("\x1b[?2027h\u0303"))
		if dst.Cell(0, 0) != term.Cell(0, 0) {
			t.Fatal("overflow append changed replayed cluster")
		}
	}
}
