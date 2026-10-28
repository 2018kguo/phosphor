package phosphor

import (
	"testing"
	"unicode/utf8"
)

// Check every accepted codepoint against the independent segmenter, including
// terminal Control tailoring. The shortcut only applies to two scalar cells.
func TestGraphemeSimpleScalarBoundaries(t *testing.T) {
	checked := 0
	for r := rune(0); r <= utf8.MaxRune; r++ {
		if r >= 0xd800 && r <= 0xdfff || !graphemeSimpleScalar(r) {
			continue
		}
		forward := graphemeContinues("a", r)
		reverse := graphemeContinues(string(r), 'a')
		if forward || reverse {
			t.Fatalf("shortcut accepts a continuing scalar %U", r)
		}
		checked++
	}
	if checked < 1000000 {
		t.Fatalf("unexpected scalar coverage: %d", checked)
	}
	for _, r := range []rune{-1, utf8.MaxRune + 1, clusterBase, '\r', '\n', 0x0300, 0x0600, 0x0903, 0x1100, 0x1160, 0x11a8, 0x200d, 0x1f1e6} {
		if graphemeSimpleScalar(r) {
			t.Fatalf("complex or invalid scalar %U accepted", r)
		}
	}
}
