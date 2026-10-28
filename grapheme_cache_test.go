package phosphor

import (
	"bytes"
	"strings"
	"testing"
	"unsafe"
)

const originalClusterSize = (unsafe.Sizeof("") + unsafe.Sizeof(clusterModes{}) + 1 + unsafe.Alignof("") - 1) / unsafe.Alignof("") * unsafe.Alignof("")

// Both arrays must compile, including under GOARCH=386: derived state must
// occupy existing padding rather than increasing every retained pool entry.
var _ [originalClusterSize - unsafe.Sizeof(cluster{})]byte
var _ [unsafe.Sizeof(cluster{}) - originalClusterSize]byte

func TestGraphemeTailIsDerived(t *testing.T) {
	if unsafe.Sizeof(graphemeTail{}) != 3 {
		t.Fatal("grapheme context grew beyond existing cluster padding")
	}
	term := New(12, 3)
	term.Write([]byte("\x1b[?2027ha\u094d\u0915"))
	before := term.Snapshot()
	c := term.x.clusters[0]
	c.tail = graphemeTail{}
	if got := term.intern(c); got != clusterBase {
		t.Fatal("derived context changed semantic cluster identity")
	}
	for i := range term.x.clusters {
		term.x.clusters[i].tail = graphemeTail{}
	}
	if !bytes.Equal(before, term.Snapshot()) {
		t.Fatal("derived context changed serialized state")
	}
	if err := term.RestoreSnapshot(before, SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	for _, c := range term.x.clusters {
		if c.tail != (graphemeTail{}) {
			t.Fatal("snapshot unexpectedly persists derived context")
		}
	}
	term.Write([]byte("\u0300\u094d\u0915"))
	if got := term.cellString(term.cellAt(0, 0)); got != "a\u094d\u0915\u0300\u094d\u0915" {
		t.Fatalf("unknown restored context changed continuation: %q", got)
	}
}

func TestGraphemeTailAcrossPoolCompactionAndRestore(t *testing.T) {
	var replies, controlReplies bytes.Buffer
	term := New(9, 4, WithScrollback(3), WithReply(&replies))
	control := New(9, 4, WithScrollback(3), WithReply(&controlReplies))
	clear := func() {
		if control.x != nil {
			for i := range control.x.clusters {
				control.x.clusters[i].tail = graphemeTail{}
			}
		}
	}
	check := func() {
		t.Helper()
		assertSnapshotState(t, term, control)
		if !bytes.Equal(term.Snapshot(), control.Snapshot()) || !bytes.Equal(term.Repaint(true), control.Repaint(true)) || replies.String() != controlReplies.String() {
			t.Fatal("cached context changed checkpoint, history, repaint or replies")
		}
	}
	for _, text := range []string{
		"\x1b[?2027ha\u094d\u0300\u200d\u0915\r\n",
		"\x1b[?2027l👩\u200b\u0300\u200d\x1b[?2027h👩\r\n",
		"\x1b[?2027h🇺🇸🇨🇦👩‍💻\r\n",
		"\x1b[31mA\u05c8\x1b[0m\r\n",
		"\x1b[?1049h☔︎\u0300\x1b[?1049l",
		"☔" + strings.Repeat("\u0300", maxClusterRunes-1) + "︎️\r\n",
		"\x1b[6n\x1b[5n\x1b[?2027$pFINAL",
	} {
		for _, r := range text {
			clear()
			for _, current := range []*Terminal{term, control} {
				if current.x != nil {
					// Interning the next prefix must remap live IDs before
					// installing the result, invalidating any old pool pointer.
					current.x.clusterCap = len(current.x.clusters)
				}
				current.Write([]byte(string(r)))
			}
			check()
		}
		for _, current := range []*Terminal{term, control} {
			current.Compact()
			current.Resize(7, 4)
			if err := current.RestoreSnapshot(current.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
		}
		check()
	}
}
