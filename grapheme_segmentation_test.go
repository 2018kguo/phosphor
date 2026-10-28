package phosphor

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/clipperhouse/uax29/v2/graphemes"
)

func TestGraphemeSegmentationKeepsByteBoundaries(t *testing.T) {
	prefixes := []string{
		"", "A", "中", "\x00", "\u200b", "A\u200b\u0301", "👩\u200d", "🇺🇸🇨",
		"가", "\u0600A\u0301", "e\u0301X\u200d", "\xff\xfe",
		strings.Repeat("\x00", maxClusterRunes),
		"👋" + strings.Repeat("\u0301", maxClusterRunes-1),
		"👩" + strings.Repeat("\u200d👩", maxClusterRunes/2),
	}
	var scratch []byte
	for _, prefix := range prefixes {
		for _, next := range []rune{0, 'A', '中', 0xad, 0x301, 0x200d, 0xfe0e, 0xfe0f, 0x1f3ff, 0x1f469, 0x1f1fa, 0x1100, 0x1161, 0x11a8, 0x61c, -1, utf8.MaxRune + 1} {
			// Keep the original byte-backed segmentation as a control for
			// stored mode-off text, including multiple old graphemes and
			// controls whose boundary property the terminal treats as Other.
			var normalized []byte
			for _, cp := range prefix {
				if runeInRanges(cp, graphemeControls[:]) {
					cp = 0xe000
				}
				normalized = utf8.AppendRune(normalized, cp)
			}
			boundary := len(normalized)
			cp := next
			if runeInRanges(cp, graphemeControls[:]) {
				cp = 0xe000
			}
			normalized = utf8.AppendRune(normalized, cp)
			iter := graphemes.FromBytes(normalized)
			want := false
			for iter.Next() {
				want = want || iter.Start() < boundary && iter.End() > boundary
			}
			var got bool
			scratch, got = graphemeContinues(scratch, prefix, next)
			if got != want {
				t.Fatalf("%q + %U joins %v, byte iterator %v", prefix, next, got, want)
			}
		}
	}
}

func TestGraphemeScratchAcrossHistoryAndCheckpoints(t *testing.T) {
	var replies, controlReplies bytes.Buffer
	term := New(9, 3, WithScrollback(3), WithReply(&replies))
	control := New(9, 3, WithScrollback(3), WithReply(&controlReplies))
	check := func(phase string) {
		t.Helper()
		assertContinuationState(t, term, control, phase)
		if !bytes.Equal(term.Snapshot(), control.Snapshot()) || !bytes.Equal(term.Repaint(true), control.Repaint(true)) || replies.String() != controlReplies.String() {
			t.Fatalf("%s: scratch reuse changed encoded state or replies", phase)
		}
	}
	write := func(input string) {
		t.Helper()
		term.Write([]byte(input))
		for i := range len(input) {
			// The byte-at-a-time control never carries a temporary buffer
			// from the previous Unicode, history or reply operation.
			control.scratch = nil
			control.Write([]byte(input[i : i+1]))
		}
		check("write " + input)
	}
	write("\x1b[?2027lA\u200b\u0301\x1b[?2027h\u200d👩\r\n")
	for range 12 {
		write("\x1b[32m\x1b]8;;https://example.test\x1b\\日e\u0301👩‍💻🇺🇸\x1b]8;;\x1b\\\x1b[0m\r\n")
	}
	write("\x1b[5n\x1b[6n\x1b[?2027$p\x1bP$qm\x1b\\")
	for _, current := range []*Terminal{term, control} {
		current.Compact()
		if current.scratch != nil {
			t.Fatal("Compact retained the temporary grapheme buffer")
		}
	}
	check("Compact")
	term.Resize(5, 4)
	control.Resize(5, 4)
	check("resize")
	write("\x1b[3")
	for _, current := range []*Terminal{term, control} {
		if err := current.RestoreSnapshot(current.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
	}
	check("restore unfinished CSI")
	write("1mTAIL\x1b[0m\r\n👩" + strings.Repeat("\u200d👩", maxClusterRunes/2) + "\r\n☔︎\x1b[6n")
	term.Resize(11, 2)
	control.Resize(11, 2)
	check("future resize")
	write("\r\nfinal e\u0301\x1b[5n")
}

func TestGraphemeCJKWithoutHistoryWarmAllocations(t *testing.T) {
	term := New(12, 3, WithScrollback(0))
	data := []byte("\x1b[?2027h" + strings.Repeat("日本語 中文 한글\r\n", 64))
	term.Write(data)
	if n := testing.AllocsPerRun(20, func() { term.Write(data) }); n != 0 {
		t.Fatalf("warm CJK output allocated %.1f times", n)
	}
}
