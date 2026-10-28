package phosphor

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCodepointWidthsMatchAuditedNativeStream(t *testing.T) {
	f, err := os.Open("internal/unicodegen/data/native-widths.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	next, checked, nonDefault, summaries := 0, 0, 0, 0
	check := func(first, last, want int) {
		t.Helper()
		for cp := first; cp <= last; cp++ {
			if got := CodepointWidth(rune(cp)); got != want {
				t.Fatalf("%U width %d, audited native width %d", cp, got, want)
			}
			checked++
		}
	}
	for {
		var row struct {
			Kind        string `json:"kind"`
			First, Last int
			Width       int    `json:"ghostty_width"`
			Scalars     int    `json:"unicode_scalars"`
			NonDefault  int    `json:"non_default_scalars"`
			Commit      string `json:"ghostty_commit"`
			Version     string `json:"unicode_version"`
		}
		if err := decoder.Decode(&row); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if summaries != 0 {
			t.Fatal("data follows native summary")
		}
		switch row.Kind {
		case "native_scalar_width_range":
			if row.First < next || row.Last < row.First || row.Last > utf8.MaxRune || row.Width != 0 && row.Width != 2 {
				t.Fatal("invalid native width range")
			}
			check(next, row.First-1, 1)
			check(row.First, row.Last, row.Width)
			next = row.Last + 1
			nonDefault += row.Last - row.First + 1
		case "native_scalar_width_summary":
			check(next, utf8.MaxRune, 1)
			if checked != 0x110000 || row.Scalars != 1112064 || row.NonDefault != nonDefault || row.Commit != "6ba144158577d033ccb6ed4b574e4013957c9161" || row.Version != "18.0.0" {
				t.Fatal("native scalar coverage changed")
			}
			summaries++
		default:
			t.Fatal("unknown native width record")
		}
	}
	if summaries != 1 {
		t.Fatal("missing native scalar coverage")
	}
	for _, cp := range []rune{-1, utf8.MaxRune + 1, clusterBase} {
		if CodepointWidth(cp) != 1 || emojiVariationBase(cp) || graphemeSimpleScalar(cp) {
			t.Fatalf("invalid scalar %U changed properties", cp)
		}
	}
}

func TestGraphemeSegmentationKeepsByteBoundaries(t *testing.T) {
	prefixes := []string{
		"", "A", "中", "\x00", "\u200b", "A\u200b\u0301", "👩\u200d", "🇺🇸🇨",
		"가", "\u0600A\u0301", "e\u0301X\u200d", "\xff\xfe",
		"क्", "a्", "क्\u200c", "क\u0300्", "\u0600्",
		"क\x00्", "क\u200b्", "क्\x00", "👩\x00\u200d", "👩्\x00\u200d",
		strings.Repeat("\x00", maxClusterRunes),
		"👋" + strings.Repeat("\u0301", maxClusterRunes-1),
		"👩" + strings.Repeat("\u200d👩", maxClusterRunes/2),
	}
	for _, prefix := range prefixes {
		for _, next := range []rune{0, 'A', '中', 0xad, 0x301, 0x200d, 0xfe0e, 0xfe0f, 0x1f3ff, 0x1f469, 0x1f1fa, 0x1100, 0x1161, 0x11a8, 0x61c, -1, utf8.MaxRune + 1} {
			want := referenceGraphemeContinues(prefix, next)
			got := graphemeContinues(prefix, next)
			if got != want {
				t.Fatalf("%q + %U joins %v, lookback reference %v", prefix, next, got, want)
			}
		}
	}
}

// Use suffix lookback instead of the generated forward-state table. The
// official corpus independently checks the properties and Unicode rules;
// this control checks table replay over arbitrary mode-off cell contents.
func referenceGraphemeContinues(prefix string, next rune) bool {
	var cluster []graphemeProperty
	for _, cp := range prefix {
		property := graphemeProps(cp)
		if !referenceGraphemeBoundary(cluster, property) {
			cluster = cluster[:0]
		}
		cluster = append(cluster, property)
	}
	return referenceGraphemeBoundary(cluster, graphemeProps(next))
}

func referenceGraphemeBoundary(prefix []graphemeProperty, next graphemeProperty) bool {
	if len(prefix) == 0 {
		return false
	}
	last := prefix[len(prefix)-1]
	a, b := last&gpKindMask, next&gpKindMask
	if a == gpCR && b == gpLF {
		return true
	}
	if a == gpCR || a == gpLF || b == gpCR || b == gpLF {
		return false
	}
	if a == gpL && (b == gpL || b == gpV || b == gpLV || b == gpLVT) ||
		(a == gpLV || a == gpV) && (b == gpV || b == gpT) ||
		(a == gpLVT || a == gpT) && b == gpT {
		return true
	}
	if b == gpExtend || b == gpZWJ || b == gpSpacing || a == gpPrepend {
		return true
	}
	if next&gpInCBMask == gpConsonant {
		for i := len(prefix) - 1; i >= 0; i-- {
			switch prefix[i] & gpInCBMask {
			case gpLinker:
				return true
			case gpInCBExtend:
			default:
				i = -1
			}
		}
	}
	if next&gpPictographic != 0 && a == gpZWJ {
		for i := len(prefix) - 2; i >= 0; i-- {
			if prefix[i]&gpKindMask != gpExtend {
				return prefix[i]&gpPictographic != 0
			}
		}
	}
	if a == gpRI && b == gpRI {
		count := 0
		for i := len(prefix) - 1; i >= 0 && prefix[i]&gpKindMask == gpRI; i-- {
			count++
		}
		return count%2 == 1
	}
	return false
}

func TestGraphemeClassPrefixesMatchLookback(t *testing.T) {
	tokens := []rune{'a', '中', 0, '\r', '\n', 0xad, 0x200b, 0x0300, 0x094d, 0x0915, 0x0903, 0x0600,
		0x1100, 0x1161, 0x11a8, 0xac00, 0xac01, 0x1cf5, 0x11a3a, 0x05c8, 0x200d, 0x200c, 0x1f3ff, 0x1f469, 0x1f1fa, 0x1f1f8, 0xfe0e, 0xfe0f}
	rng := rand.New(rand.NewPCG(347, 991))
	for trial := range 10000 {
		var prefix strings.Builder
		for range rng.IntN(maxClusterRunes + 1) {
			cp := tokens[rng.IntN(len(tokens))]
			if trial%5 == 0 {
				cp = rune(rng.Int32N(utf8.MaxRune + 1))
			}
			prefix.WriteRune(cp)
		}
		next := tokens[rng.IntN(len(tokens))]
		if trial%7 == 0 {
			next = rune(rng.Uint32())
		}
		s := prefix.String()
		got := graphemeContinues(s, next)
		if want := referenceGraphemeContinues(s, next); got != want {
			t.Fatalf("trial %d: %q + %U joins %v, lookback reference %v", trial, s, next, got, want)
		}
	}
}

func FuzzGraphemeBoundaryMatchesLookback(f *testing.F) {
	for _, prefix := range []string{"क्", "a्", "क\u200b्", "a\u1cf5", "A\u05c8", "👩\u200d", "🇺🇸🇨", "\xff\xfe"} {
		f.Add(prefix, rune(0x0915))
	}
	f.Fuzz(func(t *testing.T, prefix string, next rune) {
		if len(prefix) > maxCluster {
			prefix = prefix[:maxCluster]
		}
		got := graphemeContinues(prefix, next)
		if want := referenceGraphemeContinues(prefix, next); got != want {
			t.Fatalf("%q + %U joins %v, lookback reference %v", prefix, next, got, want)
		}
	})
}

func TestGraphemeHistoryAndCheckpointContinuation(t *testing.T) {
	var replies, controlReplies bytes.Buffer
	term := New(9, 3, WithScrollback(3), WithReply(&replies))
	control := New(9, 3, WithScrollback(3), WithReply(&controlReplies))
	check := func(phase string) {
		t.Helper()
		assertContinuationState(t, term, control, phase)
		if !bytes.Equal(term.Snapshot(), control.Snapshot()) || !bytes.Equal(term.Repaint(true), control.Repaint(true)) || replies.String() != controlReplies.String() {
			t.Fatalf("%s: Unicode operations changed encoded state or replies", phase)
		}
	}
	write := func(input string) {
		t.Helper()
		term.Write([]byte(input))
		for i := range len(input) {
			// The byte-at-a-time control never carries a temporary buffer
			// from the previous Unicode, history or reply operation.
			control.scratch = nil
			control.Write([]byte(input[i : i+1]))
		}
		check("write " + input)
	}
	write("\x1b[?2027lA\u200b\u0301\x1b[?2027h\u200d👩\r\n")
	for range 12 {
		write("\x1b[32m\x1b]8;;https://example.test\x1b\\日e\u0301👩‍💻🇺🇸\x1b]8;;\x1b\\\x1b[0m\r\n")
	}
	write("\x1b[5n\x1b[6n\x1b[?2027$p\x1bP$qm\x1b\\")
	for _, current := range []*Terminal{term, control} {
		current.Compact()
		if current.scratch != nil {
			t.Fatal("Compact retained a temporary output buffer")
		}
	}
	check("Compact")
	term.Resize(5, 4)
	control.Resize(5, 4)
	check("resize")
	write("\x1b[3")
	for _, current := range []*Terminal{term, control} {
		if err := current.RestoreSnapshot(current.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
	}
	check("restore unfinished CSI")
	write("1mTAIL\x1b[0m\r\n👩" + strings.Repeat("\u200d👩", maxClusterRunes/2) + "\r\n☔︎\x1b[6n")
	term.Resize(11, 2)
	control.Resize(11, 2)
	check("future resize")
	write("\r\nfinal e\u0301\x1b[5n")
}

func TestGraphemeCJKWithoutHistoryWarmAllocations(t *testing.T) {
	term := New(12, 3, WithScrollback(0))
	data := []byte("\x1b[?2027h" + strings.Repeat("日本語 中文 한글\r\n", 64))
	term.Write(data)
	if n := testing.AllocsPerRun(20, func() { term.Write(data) }); n != 0 {
		t.Fatalf("warm CJK output allocated %.1f times", n)
	}
}

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

// Check every accepted codepoint against suffix lookback, including
// terminal Control tailoring. The shortcut only applies to two scalar cells.
func TestGraphemeSimpleScalarBoundaries(t *testing.T) {
	checked := 0
	for r := rune(0); r <= utf8.MaxRune; r++ {
		if r >= 0xd800 && r <= 0xdfff || !graphemeSimpleScalar(r) {
			continue
		}
		forward := referenceGraphemeContinues("a", r)
		reverse := referenceGraphemeContinues(string(r), 'a')
		if forward || reverse {
			t.Fatalf("shortcut accepts a continuing scalar %U", r)
		}
		checked++
	}
	if checked < 1000000 {
		t.Fatalf("unexpected scalar coverage: %d", checked)
	}
	for _, r := range []rune{-1, utf8.MaxRune + 1, clusterBase, '\r', '\n', 0x0300, 0x0600, 0x0903, 0x1100, 0x1160, 0x11a8, 0x1cf5, 0x1cf6, 0x11a3a, 0x200d, 0x1f1e6} {
		if graphemeSimpleScalar(r) {
			t.Fatalf("complex or invalid scalar %U accepted", r)
		}
	}
}
