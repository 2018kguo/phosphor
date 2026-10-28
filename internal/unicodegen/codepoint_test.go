package main

import "testing"

func TestCodepointPackingKeepsIndependentProperties(t *testing.T) {
	var widths [0x110000]uint8
	for i := range widths {
		widths[i] = 1
	}
	widths[0x42], widths[0x43] = 2, 0
	pages, classes := codepointProperties(
		[]byte("0041 ; Control\n0042 ; Other\n0043 ; Prepend\n"),
		[]byte("0041..0042 ; Extended_Pictographic\n0041..0042 ; Emoji_Modifier_Base\n"),
		[]byte("0041 ; InCB; Extend\n0042 ; InCB; Consonant\n0043 ; InCB; Linker\n"),
		[]byte("0042 FE0E ; text style\n0042 FE0F ; emoji style\n"), widths,
	)
	for cp, want := range map[int]struct{ property, width, variation uint8 }{
		0x41: {gpModifierBase, 1, 0},
		0x42: {gpOther | gpPictographic | gpConsonant | gpModifierBase, 2, 128},
		0x43: {gpPrepend | gpLinker, 0, 0},
		0x44: {gpOther, 1, 0},
	} {
		info := pages[cp>>8][cp&255]
		if classes[info&31] != want.property || info>>5&3 != want.width || info&128 != want.variation {
			t.Fatalf("U+%04X packed %#x property %#x, want %+v", cp, info, classes[info&31], want)
		}
	}
}
