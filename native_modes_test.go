package phosphor

import (
	"bytes"
	"fmt"
	"testing"
)

func TestResizeEndsSynchronizedUpdate(t *testing.T) {
	for _, size := range [][2]int{{8, 4}, {9, 4}, {8, 5}} {
		term := New(8, 4)
		term.Write([]byte("\x1b[?2026h"))
		term.Resize(size[0], size[1])
		if term.Modes().SyncUpdate {
			t.Fatalf("resize%v retained synchronized update", size)
		}
	}
}

func TestNativeFixedModeReports(t *testing.T) {
	for _, mode := range []int{1, 5, 7, 10, 11, 13, 14, 15, 16, 17, 18, 19} {
		var replies bytes.Buffer
		term := New(8, 4, WithReply(&replies))
		term.Write(fmt.Appendf(nil, "\x1b[%dh\x1b[%d$p\x1b[%dl\x1b[%d$p", mode, mode, mode, mode))
		want := fmt.Sprintf("\x1b[%d;4$y\x1b[%d;4$y", mode, mode)
		if replies.String() != want {
			t.Fatalf("ANSI%d: got%q want%q", mode, replies.String(), want)
		}
	}
	var replies bytes.Buffer
	term := New(8, 4, WithReply(&replies))
	term.Write([]byte("\x1b[?117h\x1b[?117$p\x1b[?117l\x1b[?117$p\x1bP+q536D756C78\x1b\\"))
	if want := "\x1b[?117;4$y\x1b[?117;4$y\x1bP1+r536D756C78=5C455B343A25703125646D\x1b\\"; replies.String() != want {
		t.Fatalf("fixed mode/capability replies got%q want%q", replies.String(), want)
	}
}

func TestSnapshotRarePrivateModes(t *testing.T) {
	term := New(80, 4)
	term.Write([]byte("\x1b[?4;40h\x1b[?3h\x1b[?3;4;40s\x1b[?40l"))
	term.Compact()
	var dst Terminal
	if err := dst.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	assertSnapshotState(t, term, &dst)
	for _, next := range []string{"\x1b[!p", "\x1b[?40r\x1b[?3l", "\x1b[?3;4;40r", "\x1bc"} {
		term.Write([]byte(next))
		dst.Write([]byte(next))
		dst.Compact()
		assertSnapshotState(t, term, &dst)
	}
}
