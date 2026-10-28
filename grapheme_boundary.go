package phosphor

import "unicode/utf8"

// Unicode boundary rules are generated from the UAX29 implementation
// adapted from Copyright (c) 2020 Matt Sherman; see licenses/uax29.txt.
func graphemeProps(r rune) graphemeProperty {
	return codepointInfo(r).grapheme() & 127
}

// Every scalar lookup supplies boundary class, native width and selector
// eligibility together. Invalid scalars share RuneError's neutral properties.
type unicodeInfo uint8

const (
	codepointClassMask unicodeInfo = 31
	codepointVSBase    unicodeInfo = 128
)

func codepointInfo(r rune) unicodeInfo {
	u := uint32(r)
	if u > utf8.MaxRune {
		u = utf8.RuneError
	}
	return unicodeInfo(codepointPages[codepointPageIDs[u>>8]][u&255])
}

func (info unicodeInfo) grapheme() graphemeProperty {
	return graphemeClassProperties[info&codepointClassMask]
}
func (info unicodeInfo) width() int { return int(info >> 5 & 3) }
func (info unicodeInfo) simple() bool {
	p := info.grapheme()
	return p&gpKindMask == gpOther && p&gpInCBMask != gpLinker
}

// Replaying each stored scalar preserves cells collected with mode 2027 off,
// which can contain old boundaries. Control is already normalized to Other
// by the property table, while CR and LF keep their distinct properties.
func graphemeContinues(s string, r rune) bool {
	return graphemeTailOf(s).continues(codepointInfo(r))
}

// The count also marks whether a restored cluster's derived context is known.
// Three bytes fit the existing cluster padding on 32- and 64-bit targets.
type graphemeTail struct {
	count, state uint8
	lastInfo     unicodeInfo
}

func (tail graphemeTail) continues(current unicodeInfo) bool {
	if tail.count == 0 {
		return false
	}
	return graphemeTransitions[uint16(tail.state)<<10|uint16(tail.lastInfo&codepointClassMask)<<5|uint16(current&codepointClassMask)]&16 != 0
}

func graphemeTailOf(s string) graphemeTail {
	var tail graphemeTail
	count := 0
	for _, cp := range s {
		current := codepointInfo(cp)
		if count != 0 {
			transition := graphemeTransitions[uint16(tail.state)<<10|uint16(tail.lastInfo&codepointClassMask)<<5|uint16(current&codepointClassMask)]
			tail.state = transition & 15
			if transition&16 == 0 {
				tail.state = 0
			}
		}
		tail.lastInfo = current
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
	return graphemeTail{count: 1, lastInfo: codepointInfo(c.R)}
}
