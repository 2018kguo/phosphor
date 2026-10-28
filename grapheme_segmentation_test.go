package phosphor

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/clipperhouse/uax29/v2/graphemes"
)

func TestGraphemeStringSegmentationKeepsByteBoundaries(t *testing.T) {
	prefixes := []string{
		"", "A", "中", "\x00", "\u200b", "A\u200b\u0301", "👩\u200d", "🇺🇸🇨",
		"가", "\u0600A\u0301", "e\u0301X\u200d", "\xff\xfe",
		strings.Repeat("\x00", maxClusterRunes),
		"👋" + strings.Repeat("\u0301", maxClusterRunes-1),
		"👩" + strings.Repeat("\u200d👩", maxClusterRunes/2),
	}
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
			if got := graphemeContinues(prefix, next); got != want {
				t.Fatalf("%q + %U joins %v, byte iterator %v", prefix, next, got, want)
			}
		}
	}
}
