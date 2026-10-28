package phosphor

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestHistoryFormattingRestoredContinuation(t *testing.T) {
	source := New(24, 3, WithScrollback(4), WithScrollbackBytes(4096))
	source.Write([]byte("\x1b[?2027h" + strings.Repeat("\x1b[2G\x1b]8;;https://history.invalid/\x1b\\\x1b[31m界 á 👩‍💻 \x1b[1\"qP\x1b[0\"q\x1b[0m\x1b]8;;\x1b\\\r\n", 40)))
	data := source.Snapshot()
	var aReplies, bReplies bytes.Buffer
	a, b := New(1, 1, WithReply(&aReplies)), New(1, 1, WithReply(&bReplies))
	for _, term := range []*Terminal{a, b} {
		if err := term.RestoreSnapshot(data, SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
		if term.ScrollbackLen() != term.sbMaxLines {
			t.Fatal("fixture did not restore full history")
		}
	}
	for i := range 100 {
		input := fmt.Appendf(nil, "\x1b[31mline-%d 界 á 👩‍💻\x1b[0m\r\n", i)
		if i%7 == 0 {
			input = append(input, "\x1b]133;A\a\x1b]8;;https://future.invalid/\x1b\\linked\x1b]8;;\x1b\\\r\n"...)
		}
		input = append(input, "\x1b[5n\x1b[6n"...)
		a.Write(input)
		for _, c := range input {
			b.Write([]byte{c})
		}
		assertSnapshotState(t, a, b)
		if !bytes.Equal(a.Snapshot(), b.Snapshot()) || !bytes.Equal(aReplies.Bytes(), bReplies.Bytes()) {
			t.Fatal("restored continuation changed history, checkpoint or replies across chunk sizes")
		}
		if i == 25 || i == 75 {
			for _, term := range []*Terminal{a, b} {
				term.Compact()
				term.Resize(18+i%12, 6)
			}
			assertSnapshotState(t, a, b)
		}
	}
	if aReplies.Len() == 0 {
		t.Fatal("fixture produced no query replies")
	}
}
