package phosphor

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestInternAppendMatchesConcatenatedCluster(t *testing.T) {
	term, control := New(12, 3), New(12, 3)
	check := func(prefix string, next rune, width uint8, modes clusterModes) {
		t.Helper()
		want := cluster{text: prefix + string(next), width: width, joinModes: modes}
		got := term.internAppend(prefix, next, width, modes)
		expected := control.intern(want)
		if got != expected || !slices.Equal(term.x.clusters, control.x.clusters) {
			t.Fatalf("%q + %U width %d modes %v: appended lookup changed pool identity", prefix, next, width, modes)
		}
	}
	for round := range 4 {
		for i := range 48 {
			prefix := fmt.Sprintf("%02d日", i)
			next := []rune{'\u0301', '👩', '\u200d', '\ufe0f'}[i%4]
			var modes clusterModes
			modes.set(1 + i%3)
			check(prefix, next, uint8(1+i%2), modes)
			check(prefix, next, uint8(1+i%2), modes)
			// A matching text with different width or mode metadata is
			// a distinct cluster, even while it remains in the recent cache.
			if round%2 == 0 {
				check(prefix, next, uint8(2-i%2), modes)
				check(prefix, next, uint8(1+i%2), clusterModes{})
			}
		}
	}
	for _, prefix := range []string{"e", "👩\u200d", strings.Repeat("👩", maxClusterRunes-1)} {
		for _, next := range []rune{0x301, 0x200d, 0x1f469, 0xfe0e, 0xfe0f} {
			check(prefix, next, 2, clusterModes{})
		}
	}
}

func TestRepeatedCombiningTextWarmAllocations(t *testing.T) {
	for _, mode := range []string{"", "\x1b[?2027h"} {
		term := New(12, 3, WithScrollback(0))
		data := []byte(mode + strings.Repeat("e\u0301 ", 64) + "\r\n")
		term.Write(data)
		if n := testing.AllocsPerRun(20, func() { term.Write(data) }); n != 0 {
			t.Fatalf("mode %q: repeated combining text allocated %.1f times", mode, n)
		}
	}
}
