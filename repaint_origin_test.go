package phosphor

import "testing"

func TestRepaintOutsideOriginWithBlankAlternate(t *testing.T) {
	for _, prefix := range []string{
		"",
		"\x1b[?47h\x1b[3;4H\x1b[32m\x1b)A\x0e\x1b7\x1b[?47l",
		"\x1b[?6h\x1b[5;8H\x1b[34m\x1b(0\x1b7\x1b[?6l",
	} {
		for _, suffix := range []string{"", "q\x1b[B\x1b[C\x1b[41;1m\x1b[1\"q"} {
			term := New(10, 6)
			term.Write([]byte(prefix + "\x1b[0m\x1b(B\x0f\x1b[H\x1b[2JA\x1b[3;5r\x1b[?69h\x1b[4;8s\x1b[?6hB\x1b[22J" + suffix))
			if !term.canStageOriginCursor() {
				t.Fatalf("fixture did not reach an outside-origin cursor: %+v", term.cur)
			}
			restored := New(10, 6)
			restored.Write(term.Repaint(true))
			assertSparseObservations(t, term, restored)
			for _, continuation := range []string{"X", "\x1b8Z", "\x1b[?47h\x1b8A\x1b[?47l", "\x1b[?1047h\x1b8B\x1b[?1047l", "\x1b[?1049hQ\x1b[?1049lR"} {
				term.Write([]byte(continuation))
				restored.Write([]byte(continuation))
				assertSparseObservations(t, term, restored)
			}
		}
	}
}

func TestRepaintOutsideOriginKeepsScratchLimitsExplicit(t *testing.T) {
	term := New(10, 6)
	term.Write([]byte("\x1b[?47hkept\x1b[?47lA\x1b[3;5r\x1b[?69h\x1b[4;8s\x1b[?6hB\x1b[22J"))
	if term.canStageOriginCursor() {
		t.Fatal("nonempty alternate screen was treated as scratch")
	}
	term = New(10, 6)
	term.Write([]byte("\x1b[?6h\x1b[6;10H\x1b7\x1b[3;5r\x1b[?69h\x1b[4;8s\x1b8"))
	if term.canStageOriginCursor() {
		t.Fatal("scratch origin claimed an unreachable lower/right cursor")
	}
}
