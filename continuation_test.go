package phosphor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A repaint taken at any byte boundary must leave the next bytes in the
// same parser, including queries and controls which already took effect.
func TestRepaintContinuation(t *testing.T) {
	cases := []struct{ name, stream string }{
		{"csi-style", "\x1b[38:2::10:20:30mX\x1b[0mY"},
		{"csi-style-missing-final", "\x1b[58;5;9m\x1b[58:5:mX\x1b[58;mY\x1b[58:2:1:2mZ"},
		{"csi-mode", "\x1b[?2004hX"},
		{"csi-intermediate", "\x1b[4 qX"},
		{"csi-control", "abc\x1b[3\b\a1mX"},
		{"csi-ignore", "\x1b[?1?2004hX"},
		{"csi-overflow", "\x1b[" + strings.Repeat("1;", maxParams+8) + "31mX"},
		{"csi-saturated", "\x1b[999999999999999999999mX"},
		{"csi-cancel", "\x1b[3\x18X"},
		{"escape-intermediate", "\x1b(0lqk\x1b(BX"},
		{"escape-last-intermediate", "\x1b()0\x0elqk\x0fX"},
		{"osc-title", "\x1b]2;hello world\aX"},
		{"osc-hyperlink", "\x1b]8;id=a;https://example.com\x1b\\L\x1b]8;;\x1b\\X"},
		{"osc-overflow", "\x1b]2;" + strings.Repeat("x", maxOSC) + "\aX"},
		{"osc-escape-abort", "\x1b]2;ignored\x1b[31mX"},
		{"dcs-query", "\x1b[1;31m\x1bP$qm\x1b\\X"},
		{"dcs-cap", "\x1bP$q" + strings.Repeat("x", 300) + "\x1b\\X"},
		{"dcs-ignored", "\x1bPqignored\a\x1b\\X"},
		{"sos", "\x1bXignored\a\x1b\\X"},
		{"pm", "\x1b^ignored\x1b\\X"},
		{"apc", "\x1b_ignored\x1b\\X"},
		{"string-escape-abort", "\x1b_ignored\x1b[31mX"},
		{"utf8", "😀X"},
		{"utf8-truncated", "\xf0\x9f\x1b[31mX"},
		{"utf8-c1", "\xc2\x9b31mX"},
		{"synchronized-update", "\x1b[?2026hX\x1b[?2026lY"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := []byte(c.stream)
			for cut := 0; cut <= len(in); cut++ {
				var originalReply, restoredReply bytes.Buffer
				originalBells, restoredBells := 0, 0
				original := New(12, 5, WithReply(&originalReply), WithBell(func() { originalBells++ }))
				restored := New(12, 5, WithReply(&restoredReply), WithBell(func() { restoredBells++ }))
				original.Write(in[:cut])
				restored.Write(original.Repaint(true))
				if restoredReply.Len() != 0 || restoredBells != 0 {
					t.Fatalf("cut %d: repaint repeated callbacks: reply %q, bells %d", cut, restoredReply.Bytes(), restoredBells)
				}
				assertContinuationState(t, original, restored, fmt.Sprintf("cut %d", cut))
				// Repeated checkpoints must retain the same continuation.
				another := New(12, 5)
				another.Write(restored.Repaint(true))
				assertContinuationState(t, original, another, fmt.Sprintf("cut %d then second repaint", cut))
				originalReply.Reset()
				originalBells = 0
				original.Write(in[cut:])
				restored.Write(in[cut:])
				assertContinuationState(t, original, restored, fmt.Sprintf("cut %d then suffix", cut))
				if !bytes.Equal(originalReply.Bytes(), restoredReply.Bytes()) || originalBells != restoredBells {
					t.Fatalf("cut %d: continuation callbacks differ: replies %q / %q; bells %d / %d", cut, originalReply.Bytes(), restoredReply.Bytes(), originalBells, restoredBells)
				}
			}
		})
	}
}

func assertContinuationState(t *testing.T, original, restored *Terminal, context string) {
	t.Helper()
	if got, want := state(restored), state(original); got != want {
		t.Fatalf("%s: %s", context, lineDiff(got, want))
	}
	if got, want := restored.Modes(), original.Modes(); got != want {
		t.Fatalf("%s: modes %+v, want %+v", context, got, want)
	}
	if got, want := restored.appendContinuation(nil), original.appendContinuation(nil); !bytes.Equal(got, want) {
		t.Fatalf("%s: parser continuation %q, want %q", context, got, want)
	}
}

func TestRepaintRecordingContinuation(t *testing.T) {
	paths, err := filepath.Glob("testdata/*.raw")
	if err != nil || len(paths) == 0 {
		t.Fatalf("recordings: %v (%d files)", err, len(paths))
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			in, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			// Use the original recording geometry and every byte boundary,
			// including those inside CSI, string and UTF-8 sequences.
			cuts := 0
			for cut := 0; cut <= len(in); cut++ {
				original := New(100, 30)
				original.Write(in[:cut])
				restored := New(100, 30)
				restored.Write(original.Repaint(true))
				// Continue just far enough to exercise this sequence before
				// a later full-screen redraw could hide a bad continuation.
				end := min(cut+32, len(in))
				original.Write(in[cut:end])
				restored.Write(in[cut:end])
				if got, want := restored.Repaint(true), original.Repaint(true); !bytes.Equal(got, want) {
					assertContinuationState(t, original, restored, fmt.Sprintf("cut %d then %d bytes", cut, end-cut))
				}
				cuts++
			}
			t.Logf("checked all %d boundaries across %d bytes", cuts, len(in))
		})
	}
}

// The parser model is bounded even when hostile input never terminates a
// string or supplies arbitrarily many CSI parameters.
func TestContinuationBound(t *testing.T) {
	for _, in := range []string{
		"\x1b]2;" + strings.Repeat("x", 1<<20),
		"\x1bP$q" + strings.Repeat("x", 1<<20),
		"\x1b[" + strings.Repeat("99999999999;", 1<<16),
	} {
		term := New(12, 5)
		term.Write([]byte(in))
		if n := len(term.appendContinuation(nil)); n > maxOSC+4 {
			t.Fatalf("continuation has %d bytes for %d bytes of input", n, len(in))
		}
		if cap(term.osc) > maxOSC*2 {
			t.Fatalf("string buffer grew to %d", cap(term.osc))
		}
	}
}

func FuzzRepaintContinuation(f *testing.F) {
	for _, in := range []string{
		"\x1b[38:2::1:2:3mX\x1b[?2004hY",
		"\x1b[58;5;9m\x1b[58:5:mX\x1b[58;mY\x1b[58:2:1:2mZ",
		"\x1b[?" + strings.Repeat("999;", maxParams-2) + "999:25lX",
		"\x1b[" + strings.Repeat("0;", maxParams-1) + "31;32mX",
		"\x1b[38:2::65536:65535:999999999999999999999mX",
		"\x1b]8;id=a;https://example.com\x1b\\X",
		"\x1bP$qm\x1b\\X",
		"\x1b_ignored\x1b\\X",
		"abc\x1b[3\b1mX",
		"😀\x1b(0lqk\x1b(B",
	} {
		f.Add([]byte(in), uint16(len(in)/2))
	}
	f.Fuzz(func(t *testing.T, in []byte, at uint16) {
		if len(in) > 4096 {
			in = in[:4096]
		}
		cut := int(at) % (len(in) + 1)
		original := New(12, 5)
		original.Write(in[:cut])
		restored := New(12, 5)
		restored.Write(original.Repaint(true))
		assertContinuationState(t, original, restored, "before suffix")
		original.Write(in[cut:])
		restored.Write(in[cut:])
		assertContinuationState(t, original, restored, "after suffix")
	})
}
