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

func emitGraphemeProperties(out *bytes.Buffer, breaks, emoji, derived []byte) {
	pages := graphemeProperties(breaks, emoji, derived)
	fmt.Fprintln(out, "type graphemeProperty uint8\n\nconst (")
	for _, kind := range graphemeKinds {
		fmt.Fprintf(out, "gp%s graphemeProperty = %d\n", kind.goName, kind.value)
	}
	for _, flag := range graphemeFlags {
		fmt.Fprintf(out, "gp%s graphemeProperty = %d\n", flag.name, flag.value)
	}
	out.WriteString(")\n")
	index := make(map[[256]uint8]int)
	var unique [][256]uint8
	var ids [0x1100]int
	for i, page := range pages {
		id, ok := index[page]
		if !ok {
			id = len(unique)
			index[page] = id
			unique = append(unique, page)
		}
		ids[i] = id
	}
	idType, idBytes := "uint8", 1
	if len(unique) > 256 {
		idType, idBytes = "uint16", 2
	}
	fmt.Fprintf(out, "// %d deduplicated pages; %d bytes of shared lookup data.\nvar graphemePropertyPageIDs = [0x1100]%s{\n", len(unique), len(ids)*idBytes+256*len(unique), idType)
	for i, id := range ids {
		fmt.Fprintf(out, "%d,", id)
		if i%32 == 31 {
			out.WriteByte('\n')
		}
	}
	out.WriteString("}\n\nvar graphemePropertyPages = [...][256]uint8{\n")
	for _, page := range unique {
		out.WriteString("{\n")
		for i, p := range page {
			fmt.Fprintf(out, "%d,", p)
			if i%32 == 31 {
				out.WriteByte('\n')
			}
		}
		out.WriteString("},\n")
	}
	out.WriteString("}\n")
	emitGraphemeTransitions(out, pages)
}

func emitGraphemeTransitions(out *bytes.Buffer, pages [0x1100][256]uint8) {
	var used [128]bool
	for _, page := range pages {
		for _, property := range page {
			used[property] = true
		}
	}
	var classes []uint8
	var classIDs [128]uint8
	for property, keep := range used {
		if keep {
			classIDs[property] = uint8(len(classes))
			classes = append(classes, uint8(property))
		}
	}
	if len(classes) > 32 {
		panic("grapheme classes exceed five bits")
	}
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
	out.WriteString("var graphemePropertyClasses = [128]uint8{\n")
	for i, id := range classIDs {
		fmt.Fprintf(out, "%d,", id)
		if i%32 == 31 {
			out.WriteByte('\n')
		}
	}
	out.WriteString("}\n\nvar graphemeTransitions = [16 << 10]uint8{\n")
	for i, transition := range transitions {
		fmt.Fprintf(out, "%d,", transition)
		if i%32 == 31 {
			out.WriteByte('\n')
		}
	}
	out.WriteString("}\n")
}
