package phosphor

import "testing"

func TestModifierRequiresImmediateBase(t *testing.T) {
	for _, tc := range []struct {
		name, prefix string
		cursor       int
	}{
		{"direct", "👋\x1b[?2027h", 3},
		{"control", "👋\u200b\x1b[?2027h", 5},
		{"extend", "👋\u0301\x1b[?2027h", 5},
		{"selector-ignored", "👋\ufe0f\x1b[?2027h", 3},
		{"joiner", "👋\u200d\x1b[?2027h", 5},
		{"prepend", "👋\u0600\x1b[?2027h", 4},
		{"executed-NUL", "👋\x00\x1b[?2027h", 3},
		{"text-selector-mode-off", "✍\ufe0f\x1b[?2027h", 2},
		{"text-selector-mode-on", "\x1b[?2027h✍\ufe0f", 5},
		{"extend-mode-on", "\x1b[?2027h👋\u0301", 5},
		{"repeated-modifier", "\x1b[?2027h👋🏿", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := []byte(tc.prefix + "🏿X")
			for cut := 0; cut <= len(input); cut++ {
				term := New(12, 3)
				term.Write(input[:cut])
				term.Write(input[cut:])
				if term.Cursor().X != tc.cursor {
					t.Fatalf("cut%d cursor%+v wantX%d", cut, term.Cursor(), tc.cursor)
				}
				if tc.cursor == 5 && term.Cell(2, 0).Rune != '🏿' {
					t.Fatalf("modifier crossed intervening scalar: %+v", term.Cell(0, 0))
				}
				assertGraphemeInvariants(t, term)
				checkRepaint(t, term)
			}
		})
	}
}
