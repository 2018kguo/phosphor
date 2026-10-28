package main

type graphemeProperty = uint8

const (
	gpOther graphemeProperty = iota
	gpCR
	gpLF
	gpExtend
	gpRI
	gpPrepend
	gpSpacing
	gpL
	gpV
	gpT
	gpLV
	gpLVT
	gpZWJ
)

const (
	gpKindMask      graphemeProperty = 15
	gpPictographic  graphemeProperty = 16
	gpConsonant     graphemeProperty = 32
	gpInCBExtend    graphemeProperty = 64
	gpLinker        graphemeProperty = 96
	gpInCBMask      graphemeProperty = 96
	gpControlMarker graphemeProperty = 128
)

// Unicode 17 boundary rules adapted from github.com/clipperhouse/uax29/v2,
// Copyright (c) 2020 Matt Sherman. See licenses/uax29.txt.
// These rules generate bounded transition entries. Runtime replay restarts
// that context at earlier boundaries in stored mode-off text.
type graphemeBreakState struct {
	lastNonExtendPictographic bool
	regionalPairOdd           bool
	indic                     uint8
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
