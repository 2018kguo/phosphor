package phosphor

import (
	"bytes"
	"strings"
	"testing"
)

func TestCSIPrefixChunkingPreservesSnapshots(t *testing.T) {
	for _, input := range []string{
		"\x1b[",
		"\x1b[31mred\x1b[0mplain",
		"\x1b[38:2::1:2:3mcolor\x1b[4:3munderline",
		"\x1b[?25l\x1b[>4;2m\x1b[1 q",
		"\x1b[5n\x1b[6n\x1b[c",
		"\x1b[99999999999999999999;2H",
		"\x1b[" + strings.Repeat("1;", maxParams+2) + "m",
		"\x1b[31;\x07\x00mtext\x1b[2\x18m",
		"\x1b]2;title\x1b[31mtext\x1bP$qm\x1b\\",
		"\x1b[1;2H\x1b[?1049h\x1b[2Jalt",
	} {
		for cut := 0; cut <= len(input); cut++ {
			var batchedReply, byteReply bytes.Buffer
			batched := New(10, 4, WithReply(&batchedReply))
			bytewise := New(10, 4, WithReply(&byteReply))
			batched.Write([]byte(input[:cut]))
			for i := 0; i < cut; i++ {
				bytewise.Write([]byte(input[i : i+1]))
			}
			if !bytes.Equal(batched.Snapshot(), bytewise.Snapshot()) || !bytes.Equal(batchedReply.Bytes(), byteReply.Bytes()) {
				t.Fatalf("input %q at byte %d: batching changed parser state or replies", input, cut)
			}
			var restored Terminal
			if err := restored.RestoreSnapshot(batched.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatalf("input %q at byte %d: restore: %v", input, cut, err)
			}
			future := []byte(input[cut:] + "\x1b[0mfuture\x1b[1;1Hdone")
			batched.Write(future)
			restored.Write(future)
			if !bytes.Equal(batched.Snapshot(), restored.Snapshot()) {
				t.Fatalf("input %q at byte %d: checkpoint continuation changed", input, cut)
			}
		}
	}
}
