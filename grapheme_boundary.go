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
	return graphemeTailOf(s).continues(r)
}

// The count also marks whether a restored cluster's derived context is known.
// Three bytes fit the existing cluster padding on 32- and 64-bit targets.
type graphemeTail struct {
	count, state, lastClass uint8
}

func (tail graphemeTail) continues(r rune) bool {
	if tail.count == 0 {
		return false
	}
	current := graphemePropertyClasses[uint8(graphemeProps(r))&127] & 31
	return graphemeTransitions[uint16(tail.state)<<10|uint16(tail.lastClass)<<5|uint16(current)]&16 != 0
}

func graphemeTailOf(s string) graphemeTail {
	var tail graphemeTail
	count := 0
	for _, cp := range s {
		current := graphemePropertyClasses[uint8(graphemeProps(cp))&127] & 31
		if count != 0 {
			transition := graphemeTransitions[uint16(tail.state)<<10|uint16(tail.lastClass)<<5|uint16(current)]
			tail.state = transition & 15
			if transition&16 == 0 {
				tail.state = 0
			}
		}
		tail.lastClass = current
		count++
	}
	// Stored clusters contain at most 65 scalars. Saturation also lets the
	// boundary helper accept arbitrary mode-off text without wrapping to zero.
	tail.count = uint8(min(count, 255))
	return tail
}

func (t *Terminal) clusterTail(c cell) graphemeTail {
	if c.R >= clusterBase {
		cluster := &t.x.clusters[c.R-clusterBase]
		if cluster.tail.count == 0 {
			cluster.tail = graphemeTailOf(cluster.text)
		}
		return cluster.tail
	}
	return graphemeTail{count: 1, lastClass: graphemePropertyClasses[uint8(graphemeProps(c.R))&127] & 31}
}
