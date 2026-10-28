package phosphor

import "unicode/utf8"

// Unicode 17 boundary rules adapted from github.com/clipperhouse/uax29/v2,
// Copyright (c) 2020 Matt Sherman. See licenses/uax29.txt.
// Like Ghostty's rune/property kernel, state lives only for this replay;
// stored mode-off text can contain several earlier grapheme boundaries.
type graphemeBreakState struct {
	lastNonExtendPictographic bool
	regionalPairOdd           bool
	indic                     uint8
}

func graphemeProps(r rune) graphemeProperty {
	u := uint32(r)
	if u > utf8.MaxRune {
		u = utf8.RuneError
	}
	return graphemeProperty(graphemePropertyPages[graphemePropertyPageIDs[u>>8]][u&255])
}

func (s *graphemeBreakState) continues(last, current graphemeProperty) bool {
	lastKind, currentKind := last&gpKindMask, current&gpKindMask
	beforeZWJ := s.lastNonExtendPictographic
	if lastKind != gpExtend {
		s.lastNonExtendPictographic = last&gpPictographic != 0
	}
	switch last & gpInCBMask {
	case gpConsonant:
		if s.indic != 2 {
			s.indic = 1
		}
	case gpLinker:
		if s.indic >= 1 {
			s.indic = 2
		}
	case gpInCBExtend:
	default:
		s.indic = 0
	}
	if lastKind == gpCR && currentKind == gpLF { // GB3
		return true
	}
	if lastKind == gpCR || lastKind == gpLF || currentKind == gpCR || currentKind == gpLF { // GB4/5
		return false
	}
	if lastKind == gpL && (currentKind == gpL || currentKind == gpV || currentKind == gpLV || currentKind == gpLVT) { // GB6
		return true
	}
	if (lastKind == gpLV || lastKind == gpV) && (currentKind == gpV || currentKind == gpT) { // GB7
		return true
	}
	if (lastKind == gpLVT || lastKind == gpT) && currentKind == gpT { // GB8
		return true
	}
	if currentKind == gpExtend || currentKind == gpZWJ || currentKind == gpSpacing || lastKind == gpPrepend { // GB9/9a/9b
		return true
	}
	if s.indic == 2 && current&gpInCBMask == gpConsonant { // GB9c
		s.indic = 1
		return true
	}
	if current&gpPictographic != 0 && lastKind == gpZWJ && beforeZWJ { // GB11
		return true
	}
	if lastKind == gpRI && currentKind == gpRI { // GB12/13
		s.regionalPairOdd = !s.regionalPairOdd
		return s.regionalPairOdd
	}
	return false
}
