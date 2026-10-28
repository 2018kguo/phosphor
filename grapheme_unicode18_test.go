package phosphor

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestOfficialUnicode18GraphemeCorpus(t *testing.T) {
	// Exact upstream source: https://www.unicode.org/Public/18.0.0/ucd/auxiliary/GraphemeBreakTest.txt
	// Unicode data license: testdata/conformance/LICENSE.unicode.
	data, err := os.ReadFile("testdata/unicode/GraphemeBreakTest.txt")
	if err != nil {
		t.Fatal(err)
	}
	if sum := fmt.Sprintf("%x", sha256.Sum256(data)); sum != "b0cf047ee94485bbdc846de2b902f5f8a815f6b674f9d04223cddadd91c9df31" {
		t.Fatalf("official Unicode corpus checksum changed: %s", sum)
	}
	type boundary struct {
		prefix string
		next   rune
	}
	// Executed controls never reach the segmenter. A stored mode-off
	// grapheme Control is instead treated as Other, as in the terminal.
	// These exact boundaries retain that policy difference in the counts.
	tailored := map[boundary]bool{
		{"\x00", 0x0308}: false, {"\x00", 0x094d}: false, {"\x00", 0x0300}: false,
		{"\x00", 0x200c}: false, {"\x00", 0x200d}: false, {"\x00", 0x0903}: false,
		{"\u06dd", 0}: false,
	}
	lineNumber, cases, total, official, tailoring := 0, 0, 0, 0, 0
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		lineNumber++
		line, _, _ := strings.Cut(scanner.Text(), "#")
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) < 3 || len(fields)%2 != 1 || fields[0] != "÷" || fields[len(fields)-1] != "÷" {
			t.Fatalf("line %d: invalid official boundary syntax", lineNumber)
		}
		prefix := ""
		for i := 1; i < len(fields); i += 2 {
			value, err := strconv.ParseUint(fields[i], 16, 32)
			if err != nil || !utf8.ValidRune(rune(value)) || fields[i-1] != "÷" && fields[i-1] != "×" {
				t.Fatalf("line %d: invalid codepoint or boundary", lineNumber)
			}
			r := rune(value)
			if i == 1 {
				prefix = string(r)
				continue
			}
			want := fields[i-1] == "×"
			got := graphemeContinues(prefix, r)
			if reference := referenceGraphemeContinues(prefix, r); reference != got {
				t.Fatalf("line %d: generated table joins %v, suffix-lookback reference %v", lineNumber, got, reference)
			}
			total++
			key := boundary{prefix, r}
			if _, ok := tailored[key]; ok {
				if !got || want {
					t.Fatalf("line %d: Control tailoring changed: %q + %U join %v, Unicode join %v", lineNumber, prefix, r, got, want)
				}
				tailored[key] = true
				tailoring++
			} else {
				if got != want {
					t.Fatalf("line %d: %q + %U joins %v, Unicode joins %v", lineNumber, prefix, r, got, want)
				}
				official++
			}
			if got {
				prefix += string(r)
			} else {
				prefix = string(r)
			}
		}
		cases++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	for key, seen := range tailored {
		if !seen {
			t.Fatalf("unexercised Control tailoring: %+v", key)
		}
	}
	if cases != 853 || total != 1355 || official != 1329 || tailoring != 26 {
		t.Fatalf("Unicode18 coverage changed: %d cases, %d boundaries, %d official, %d tailored", cases, total, official, tailoring)
	}
	t.Logf("Unicode18: %d cases, %d boundaries, %d official matches, %d explicit Control-tailoring differences", cases, total, official, tailoring)
}

func TestUnicode18TerminalContinuations(t *testing.T) {
	for _, tc := range []struct {
		name, prefix, suffix, cluster string
		x, width                      int
	}{
		{"linker-without-consonant", "\x1b[?2027ha\u094d", "\u0915", "a\u094d\u0915", 0, 2},
		{"linker-through-extend", "\x1b[?2027ha\u094d\u0300\u200d", "\u0915", "a\u094d\u0300\u200d\u0915", 0, 2},
		{"other-linker", "\x1b[?2027ha\u1cf5", "\u0915", "\u1cf5\u0915", 1, 2},
		{"other-linker-supplementary", "\x1b[?2027ha\U00011a3a", "\U00011a0b", "\U00011a3a\U00011a0b", 1, 2},
		{"mode-off-prefix", "a\u094d\x1b[?2027h", "\u0915", "a\u094d\u0915", 0, 2},
		{"new-hebrew-extend", "\x1b[?2027hA", "\u05c8", "A\u05c8", 0, 1},
		{"new-oriya-extend", "\x1b[?2027hA", "\u0b53", "A\u0b53", 0, 1},
		{"new-musical-extend", "\x1b[?2027hA", "\U0001d127", "A\U0001d127", 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			term := New(12, 3, WithScrollback(4))
			term.Write([]byte(tc.prefix))
			term.Compact()
			term.Resize(10, 4)
			if err := term.RestoreSnapshot(term.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			for _, b := range []byte(tc.suffix) {
				term.Write([]byte{b})
			}
			if got := term.cellString(term.cellAt(tc.x, 0)); got != tc.cluster || term.Cursor().X != tc.x+tc.width {
				t.Fatalf("cluster %q cursor %+v; want %q at %d with width %d", got, term.Cursor(), tc.cluster, tc.x, tc.width)
			}
			checkRepaint(t, term)
			term.Write([]byte("\r\nB\r\nC\r\nD\r\nE"))
			if term.ScrollbackLen() == 0 {
				t.Fatal("Unicode18 cluster did not reach history")
			}
			assertGraphemeInvariants(t, term)
			checkRepaint(t, term)
		})
	}
}
