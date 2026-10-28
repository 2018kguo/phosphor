package phosphor

import "testing"

func TestRepaintSavedPendingWrap(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		cols        int
	}{
		{"overwritten-first-cell", "abcd\x1b7\rX", 4},
		{"current-also-pending", "abcd\x1b7", 4},
		{"erased-under-saved-cursor", "abcd\x1b7\x1b[2J", 4},
		{"wide-under-saved-cursor", "ab漢\x1b7\rX", 4},
		{"saved-pen-differs", "abcd\x1b[31m\x1b]8;;https://saved\x1b\\\x1b7\x1b[0m\x1b]8;;\x1b\\\rX", 4},
		{"saved-origin", "\x1b[?6habcd\x1b7\x1b[?6l\rX", 4},
		{"old-right-margin", "\x1b[?69h\x1b[1;2sab\x1b7\x1b[?69l\rX", 8},
		{"custom-tabs", "\x1b[3g\x1b[3G\x1bH\x1b[6G\x1bH\rabcdefgh\x1b[Z\x1b7\rX", 8},
		{"tab-to-zero", "abcdefgh\x1b[Z\x1b7\rX", 8},
		{"one-column", "a\x1b7\rX", 1},
		{"wrapped-row", "abcd\x1b7e\rX", 4},
		{"alternate-active", "main\x1b[?1049habcd\x1b7\rX", 4},
		{"alternate-kept", "\x1b[?47habcd\x1b7\rX\x1b[?47l", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			term := New(tc.cols, 3)
			term.Write([]byte(tc.input))
			restored := New(tc.cols, 3)
			restored.Write(term.Repaint(true))
			assertSparseObservations(t, term, restored)
			assertRepaintSavedSlots(t, term, restored)
			for _, suffix := range []string{"\x1b8Z", "\x1b[?47h\x1b8Q\x1b[?47l", "\x1b[?1049lR", "\x1b8S"} {
				term.Write([]byte(suffix))
				restored.Write([]byte(suffix))
				assertSparseObservations(t, term, restored)
			}
		})
	}
}

func assertRepaintSavedSlots(t *testing.T, a, b *Terminal) {
	t.Helper()
	for i := range 2 {
		var left, right savedCursor
		if a.saved != nil {
			left = a.saved.screen[i]
		}
		if b.saved != nil {
			right = b.saved.screen[i]
		}
		if left.valid != right.valid || left.valid && !snapshotCursorsEqual(a, b, left.cursor, right.cursor) {
			t.Fatalf("saved cursor %d changed: %+v / %+v", i, left, right)
		}
	}
}
