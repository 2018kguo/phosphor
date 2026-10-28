package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

const gpModifierBase = 128

func codepointProperties(breaks, emoji, derived, variation []byte, widths [0x110000]uint8) (pages [0x1100][256]uint8, classes []uint8) {
	properties := graphemeProperties(breaks, emoji, derived)
	// Modifier-base status is independent of Control normalization: it affects
	// UTS51 filtering, whereas the normalized low seven bits govern boundaries.
	visitProperty(emoji, "Emoji_Modifier_Base", func(cp uint32) { properties[cp>>8][cp&255] |= gpModifierBase })
	var used [256]bool
	for _, page := range properties {
		for _, p := range page {
			used[p] = true
		}
	}
	var classIDs [256]uint8
	for property, keep := range used {
		if keep {
			classIDs[property] = uint8(len(classes))
			classes = append(classes, uint8(property))
		}
	}
	if len(classes) > 32 {
		panic("Unicode classes exceed five bits")
	}
	for cp, width := range widths {
		if width > 2 {
			panic("Unicode width exceeds two bits")
		}
		pages[cp>>8][cp&255] = classIDs[properties[cp>>8][cp&255]] | width<<5
	}
	// A text-variation sequence identifies the base that accepts either VS.
	scanner := bufio.NewScanner(bytes.NewReader(variation))
	for scanner.Scan() {
		line, _, _ := strings.Cut(scanner.Text(), "#")
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "FE0E" {
			cp, err := strconv.ParseUint(fields[0], 16, 32)
			must(err)
			if cp > 0x10ffff {
				panic("invalid variation base")
			}
			pages[cp>>8][cp&255] |= 128
		}
	}
	must(scanner.Err())
	return pages, classes
}

func emitCodepointProperties(out *bytes.Buffer, breaks, emoji, derived, variation []byte, widths [0x110000]uint8) {
	pages, classes := codepointProperties(breaks, emoji, derived, variation, widths)
	fmt.Fprintln(out, "type graphemeProperty uint8\n\nconst (")
	for _, kind := range graphemeKinds {
		fmt.Fprintf(out, "gp%s graphemeProperty = %d\n", kind.goName, kind.value)
	}
	for _, flag := range graphemeFlags {
		fmt.Fprintf(out, "gp%s graphemeProperty = %d\n", flag.name, flag.value)
	}
	fmt.Fprintln(out, "gpModifierBase graphemeProperty = 128\n)")
	fmt.Fprintf(out, "// %d classes retain normalized GCB/EP/InCB and independent modifier-base status.\nvar graphemeClassProperties = [32]graphemeProperty{\n", len(classes))
	for _, p := range classes {
		fmt.Fprintf(out, "%d,", p)
	}
	out.WriteString("}\n\n")
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
	fmt.Fprintf(out, "// Five class bits, two width bits and one variation-base bit per scalar.\n// %d deduplicated pages; %d bytes of shared lookup data.\nvar codepointPageIDs = [0x1100]%s{\n", len(unique), len(ids)*idBytes+256*len(unique), idType)
	for i, id := range ids {
		fmt.Fprintf(out, "%d,", id)
		if i%32 == 31 {
			out.WriteByte('\n')
		}
	}
	out.WriteString("}\n\nvar codepointPages = [...][256]uint8{\n")
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
	emitGraphemeTransitions(out, classes)
}

func visitProperty(data []byte, name string, visit func(uint32)) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line, _, _ := strings.Cut(scanner.Text(), "#")
		fields := strings.Split(line, ";")
		if len(fields) != 2 || strings.TrimSpace(fields[1]) != name {
			continue
		}
		first, last, isRange := strings.Cut(strings.TrimSpace(fields[0]), "..")
		lo, err := strconv.ParseUint(first, 16, 32)
		must(err)
		hi := lo
		if isRange {
			hi, err = strconv.ParseUint(last, 16, 32)
			must(err)
		}
		if lo > hi || hi > 0x10ffff {
			panic("invalid Unicode property range")
		}
		for cp := lo; cp <= hi; cp++ {
			visit(uint32(cp))
		}
	}
	must(scanner.Err())
}
