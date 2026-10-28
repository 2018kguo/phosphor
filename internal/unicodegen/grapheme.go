package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

var graphemeKinds = [...]struct{ ucd, goName string }{
	{"Other", "Other"}, {"CR", "CR"}, {"LF", "LF"}, {"Extend", "Extend"},
	{"Regional_Indicator", "RI"}, {"Prepend", "Prepend"}, {"SpacingMark", "Spacing"},
	{"L", "L"}, {"V", "V"}, {"T", "T"}, {"LV", "LV"}, {"LVT", "LVT"}, {"ZWJ", "ZWJ"},
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
			return 128, true // cleared after the independent properties merge
		}
		for i, kind := range graphemeKinds {
			if kind.ucd == fields[1] {
				return uint8(i), true
			}
		}
		panic("unknown grapheme break property: " + fields[1])
	})
	visit(emoji, func(fields []string) (uint8, bool) {
		return 16, fields[1] == "Extended_Pictographic"
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
			return 32, true
		case "Extend":
			return 64, true
		case "Linker":
			return 96, true
		default:
			panic("unknown Indic conjunct property: " + fields[2])
		}
	})
	for i := range pages {
		for j, property := range pages[i] {
			if property&128 != 0 {
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
	for i, kind := range graphemeKinds {
		fmt.Fprintf(out, "gp%s graphemeProperty = %d\n", kind.goName, i)
	}
	fmt.Fprintln(out, "gpKindMask graphemeProperty = 15\ngpPictographic graphemeProperty = 16\ngpConsonant graphemeProperty = 32\ngpInCBExtend graphemeProperty = 64\ngpLinker graphemeProperty = 96\ngpInCBMask graphemeProperty = 96\n)")
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
}
