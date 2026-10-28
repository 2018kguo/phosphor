package phosphor

import "unicode/utf8"

// Unicode boundary rules are generated from the UAX29 implementation
// adapted from Copyright (c) 2020 Matt Sherman; see licenses/uax29.txt.
func graphemeProps(r rune) graphemeProperty {
	u := uint32(r)
	if u > utf8.MaxRune {
		u = utf8.RuneError
	}
	return graphemeProperty(graphemePropertyPages[graphemePropertyPageIDs[u>>8]][u&255])
}

// Replaying each stored scalar preserves cells collected with mode 2027 off,
// which can contain old boundaries. Control is already normalized to Other
// by the property table, while CR and LF keep their distinct properties.
func graphemeContinues(s string, r rune) bool {
	var state, last uint8
	first := true
	for _, cp := range s {
		current := graphemePropertyClasses[uint8(graphemeProps(cp))&127] & 31
		if !first {
			transition := graphemeTransitions[uint16(state)<<10|uint16(last)<<5|uint16(current)]
			state = transition & 15
			if transition&16 == 0 {
				state = 0
			}
		}
		last, first = current, false
	}
	if first {
		return false
	}
	current := graphemePropertyClasses[uint8(graphemeProps(r))&127] & 31
	return graphemeTransitions[uint16(state)<<10|uint16(last)<<5|uint16(current)]&16 != 0
}
