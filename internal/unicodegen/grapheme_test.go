package main

import "testing"

func TestControlTailoringDropsIndependentProperties(t *testing.T) {
	pages := graphemeProperties(
		[]byte("0041 ; Control\n0042 ; Other\n000D ; CR\n000A ; LF\n"),
		[]byte("0041..0042 ; Extended_Pictographic\n"),
		[]byte("0041 ; InCB; Extend\n0042 ; InCB; Consonant\n"),
	)
	for cp, want := range map[int]uint8{0x41: 0, 0x42: 16 | 32, 0x0d: 1, 0x0a: 2} {
		if got := pages[cp>>8][cp&255]; got != want {
			t.Fatalf("U+%04X property = %#x, want %#x", cp, got, want)
		}
	}
}
