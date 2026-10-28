package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

var graphemeKinds = [...]struct {
	ucd, goName string
	value       graphemeProperty
}{
	{"Other", "Other", gpOther}, {"CR", "CR", gpCR}, {"LF", "LF", gpLF}, {"Extend", "Extend", gpExtend},
	{"Regional_Indicator", "RI", gpRI}, {"Prepend", "Prepend", gpPrepend}, {"SpacingMark", "Spacing", gpSpacing},
	{"L", "L", gpL}, {"V", "V", gpV}, {"T", "T", gpT}, {"LV", "LV", gpLV}, {"LVT", "LVT", gpLVT}, {"ZWJ", "ZWJ", gpZWJ},
}

var graphemeFlags = [...]struct {
	name  string
	value graphemeProperty
}{
	{"KindMask", gpKindMask}, {"Pictographic", gpPictographic}, {"Consonant", gpConsonant},
	{"InCBExtend", gpInCBExtend}, {"Linker", gpLinker}, {"InCBMask", gpInCBMask},
}

// The low nibble is Grapheme_Cluster_Break, followed by Extended_Pictographic
// and the two-bit Indic_Conjunct_Break value. Control becomes Other under the
// terminal's existing tailoring, while CR and LF keep their distinct classes.
func graphemeProperties(breaks, emoji, derived []byte) (pages [0x1100][256]uint8) {
	visit := func(data []byte, property func([]string) (uint8, bool)) {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line, _, _ := strings.Cut(scanner.Text(), "#")
			fields := strings.Split(strings.TrimSpace(line), ";")
			if len(fields) < 2 {
				continue
			}
			for i := range fields {
				fields[i] = strings.TrimSpace(fields[i])
			}
			bits, keep := property(fields)
			if !keep {
				continue
			}
			first, last, isRange := strings.Cut(fields[0], "..")
			lo, err := strconv.ParseUint(first, 16, 32)
			must(err)
			hi := lo
			if isRange {
				hi, err = strconv.ParseUint(last, 16, 32)
				must(err)
			}
			if lo > hi || hi > 0x10ffff {
				panic("invalid grapheme property range")
			}
			for cp := lo; cp <= hi; cp++ {
				pages[cp>>8][cp&255] |= bits
			}
		}
		must(scanner.Err())
	}
	visit(breaks, func(fields []string) (uint8, bool) {
		if fields[1] == "Control" {
			return gpControlMarker, true // cleared after the independent properties merge
		}
		for _, kind := range graphemeKinds {
			if kind.ucd == fields[1] {
				return kind.value, true
			}
		}
		panic("unknown grapheme break property: " + fields[1])
	})
	visit(emoji, func(fields []string) (uint8, bool) {
		return gpPictographic, fields[1] == "Extended_Pictographic"
	})
	visit(derived, func(fields []string) (uint8, bool) {
		if fields[1] != "InCB" {
			return 0, false
		}
		if len(fields) != 3 {
			panic("invalid Indic conjunct property")
		}
		switch fields[2] {
		case "Consonant":
			return gpConsonant, true
		case "Extend":
			return gpInCBExtend, true
		case "Linker":
			return gpLinker, true
		default:
			panic("unknown Indic conjunct property: " + fields[2])
		}
	})
	for i := range pages {
		for j, property := range pages[i] {
			if property&gpControlMarker != 0 {
				// Normalization replaces the whole scalar with private-use
				// Other, discarding its EP and InCB properties as well.
				pages[i][j] = 0
			}
		}
	}
	return pages
}

func emitGraphemeTransitions(out *bytes.Buffer, classes []uint8) {
	var transitions [16 << 10]uint8
	for state := range 16 {
		for i, last := range classes {
			for j, current := range classes {
				s := graphemeBreakState{indic: uint8(state & 3), lastNonExtendPictographic: state&4 != 0, regionalPairOdd: state&8 != 0}
				continues := s.continues(last, current)
				next := s.indic
				if s.lastNonExtendPictographic {
					next |= 4
				}
				if s.regionalPairOdd {
					next |= 8
				}
				if continues {
					next |= 16
				}
				transitions[state<<10|i<<5|j] = next
			}
		}
	}
	fmt.Fprintln(out, "// Boundary transitions use 16 state encodings and five-bit property classes.")
	fmt.Fprintln(out, "// Rules adapted from Copyright (c) 2020 Matt Sherman; see licenses/uax29.txt.")
	out.WriteString("var graphemeTransitions = [16 << 10]uint8{\n")
	for i, transition := range transitions {
		fmt.Fprintf(out, "%d,", transition)
		if i%32 == 31 {
			out.WriteByte('\n')
		}
	}
	out.WriteString("}\n")
}
