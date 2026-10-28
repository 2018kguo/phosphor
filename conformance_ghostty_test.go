package phosphor

import (
	"fmt"
	"strings"
)

// Translated from ghostty (MIT), src/terminal/Terminal.zig at commit 521cf2c
// (tests "Terminal: ..." between lines 4296 and 9048).

func init() { registerSuite("ghostty", ghostty1Cases) }

const (
	g1Grid  = "ABC\r\nDEF\r\nGHI"
	g1Osc8  = "\x1b]8;;http://example.com\x1b\\"
	g1Osc8e = "\x1b]8;;\x1b\\"
	g1Fam   = "\U0001F468‍\U0001F469‍\U0001F467" // man ZWJ woman ZWJ girl
	g2027   = "\x1b[?2027h"
	gLRM    = "DECLRMM left/right margins"
	gGraph  = "grapheme clustering (mode 2027, VS15/VS16 width changes)"
)

// g1Cell checks the rune at x, y (0 for blank, wideTail for a right half).
func g1Cell(t *Terminal, x, y int, r rune) string {
	if got := t.baseRune(x, y); got != r {
		return fmt.Sprintf("cell %d,%d rune got %U want %U", x, y, got, r)
	}
	return ""
}

// g1Cells runs several g1Cell checks: triples of x, y, rune.
func g1Cells(t *Terminal, want ...[3]int) string {
	for _, w := range want {
		if msg := g1Cell(t, w[0], w[1], rune(w[2])); msg != "" {
			return msg
		}
	}
	return ""
}

func g1Prompts(want ...PromptKind) func(*Terminal) string {
	return func(t *Terminal) string {
		_, rows := t.Size()
		for y := range rows {
			kind := PromptNone
			if y < len(want) {
				kind = want[y]
			}
			if got := t.Prompt(y); got != kind {
				return fmt.Sprintf("row %d prompt got %d want %d", y, got, kind)
			}
		}
		return ""
	}
}

func g1Hyperlinks(want ...string) func(*Terminal) string {
	return func(t *Terminal) string {
		for x := 0; x <= len(want); x++ {
			uri := ""
			if x < len(want) {
				uri = want[x]
			}
			if got := t.Cell(x, 0).Link.URI; got != uri {
				return fmt.Sprintf("cell %d,0 URI got %q want %q", x, got, uri)
			}
		}
		return ""
	}
}

const g1Tail = int(wideTail)

// g1DenseLinks prints A..J on the current row, each with its own OSC 8 URI.
func g1DenseLinks() string {
	var b strings.Builder
	for i := range 10 {
		fmt.Fprintf(&b, "\x1b]8;;http://example.com/%d\x1b\\%c\x1b]8;;\x1b\\", i, 'A'+i)
	}
	return b.String()
}

// setCursorPos (original test) checkpoints.
const (
	g1SCP3  = "\x1b[0;0H\x1b[81;81H"
	g1SCP4  = g1SCP3 + "\x1b[0;80Hc"
	g1SCP5  = g1SCP4 + "\x1b[0;80H"
	g1SCP6  = g1SCP5 + "\x1b[?6h\x1b[81;81H"
	g1SCP7  = g1SCP6 + "\x1b[10;80r\x1b[0;0H"
	g1SCP8  = g1SCP7 + "\x1b[1;1H"
	g1SCP9  = g1SCP8 + "\x1b[100;0H"
	g1SCP10 = g1SCP9 + "\x1b[10;11r\x1b[2;0H"
)

var ghostty1Cases = []confCase{
	// cursor positioning with origin + left/right margins
	{name: "setCursorPos saturates overflowing origin offsets", cols: 10, rows: 10,
		in: "\x1b[3;8r\x1b[?69h\x1b[4;9s\x1b[?6h\x1b[99999;99999H", cur: at(8, 7), gap: gLRM},
	{name: "setCursorRow and setCursorCol in origin mode", cols: 10, rows: 10,
		in: "\x1b[3;8r\x1b[?69h\x1b[4;9s\x1b[?6h\x1b[2;2H\x1b[1G", cur: at(3, 3), gap: gLRM},
	{name: "setCursorRow and setCursorCol in origin mode #2", cols: 10, rows: 10,
		in: "\x1b[3;8r\x1b[?69h\x1b[4;9s\x1b[?6h\x1b[2;2H\x1b[1G\x1b[1d", cur: at(3, 2), gap: gLRM},
	{name: "cursorRowRelative and cursorColRelative in origin mode", cols: 10, rows: 10,
		in: "\x1b[3;8r\x1b[?69h\x1b[4;9s\x1b[?6h\x1b[2;2H\x1b[2a\x1b[2e\x1b[100a\x1b[100e", cur: at(8, 7), gap: gLRM},
	{name: "cursorRowRelative and cursorColRelative stop at screen edge", cols: 10, rows: 10,
		in: "\x1b[3;4H\x1b[2a\x1b[1e", cur: at(5, 3)},
	{name: "cursorRowRelative and cursorColRelative stop at screen edge #2", cols: 10, rows: 10,
		in: "\x1b[3;4H\x1b[2a\x1b[1e\x1b[65535a\x1b[65535e", cur: at(9, 9)},

	// input
	{name: "input with no control characters", cols: 40, rows: 40,
		in: "hello", cur: at(5, 0), screen: scr("hello")},
	{name: "input with basic wraparound", cols: 5, rows: 40,
		in: "helloworldabc12", cur: at(4, 2), pending: yes(), screen: scr("hello\nworld\nabc12")},
	{name: "input that forces scroll", cols: 1, rows: 5,
		in: "abcdef", cur: at(0, 4), screen: scr("b\nc\nd\ne\nf")},
	{name: "zero-width character at start", cols: 80, rows: 80,
		in: "‍", cur: at(0, 0)},
	{name: "zero-width character attaches to pending wrap cell", cols: 2, rows: 2,
		in: "\x1b[?2027lxå̲", screen: scr("xå̲")},
	{name: "print single very long line", cols: 5, rows: 5,
		in: strings.Repeat("x", 1000)}, // upstream asserts only that it does not crash

	// wide characters
	{name: "print wide char", cols: 80, rows: 80,
		in: "\U0001F600", cur: at(2, 0),
		check: func(t *Terminal) string { return g1Cells(t, [3]int{0, 0, 0x1F600}, [3]int{1, 0, g1Tail}) }},
	{name: "print wide char at edge creates spacer head", cols: 10, rows: 10,
		in: "\x1b[1;10H\U0001F600", cur: at(2, 1),
		check: func(t *Terminal) string {
			return g1Cells(t, [3]int{9, 0, 0}, [3]int{0, 1, 0x1F600}, [3]int{1, 1, g1Tail})
		}},
	{name: "print wide char in single-width terminal", cols: 1, rows: 80,
		in: "\U0001F600", cur: at(0, 0), pending: yes(),
		check: func(t *Terminal) string { return g1Cell(t, 0, 0, 0) }},
	{name: "print over wide char at 0,0", cols: 80, rows: 80,
		in: "\U0001F600\x1b[0;0HA", cur: at(1, 0),
		check: func(t *Terminal) string { return g1Cells(t, [3]int{0, 0, 'A'}, [3]int{1, 0, 0}) }},
	{name: "print over wide char at col 0 corrupts previous row", cols: 10, rows: 3,
		in: strings.Repeat("中", 10) + "\x1b[2;1HA",
		check: func(t *Terminal) string {
			return g1Cells(t, [3]int{0, 1, 'A'}, [3]int{8, 0, 0x4e2d}, [3]int{9, 0, g1Tail})
		}},
	{name: "print over wide spacer tail", cols: 5, rows: 5,
		in: "橋\x1b[1;2HX", screen: scr(" X"),
		check: func(t *Terminal) string { return g1Cells(t, [3]int{0, 0, 0}, [3]int{1, 0, 'X'}) }},
	{name: "print over wide char with bold", cols: 80, rows: 80,
		in: "\x1b[1m\U0001F600\x1b[0;0H\x1b[0mA",
		check: func(t *Terminal) string {
			// upstream: the style map is empty, i.e. no cell keeps the bold style.
			for x := range 2 {
				if c := t.cellAt(x, 0); c.A != 0 {
					return fmt.Sprintf("cell %d,0 attrs %v, want none (bold dropped)", x, c.A)
				}
			}
			return g1Cells(t, [3]int{0, 0, 'A'}, [3]int{1, 0, 0})
		}},
	{name: "print over wide char with bg color", cols: 80, rows: 80,
		in: "\x1b[48;2;255;0;0m\U0001F600\x1b[0;0H\x1b[0mA",
		check: func(t *Terminal) string {
			for x := range 2 {
				if c := t.cellAt(x, 0); c.Bg != 0 {
					return fmt.Sprintf("cell %d,0 bg %#x, want default (bg style dropped)", x, c.Bg)
				}
			}
			return ""
		}},
	{name: "print multicodepoint grapheme, disabled mode 2027", cols: 80, rows: 80,
		in: g1Fam, cur: at(6, 0),
		check: func(t *Terminal) string {
			return g1Cells(t, [3]int{0, 0, 0x1F468}, [3]int{1, 0, g1Tail}, [3]int{2, 0, 0x1F469},
				[3]int{3, 0, g1Tail}, [3]int{4, 0, 0x1F467}, [3]int{5, 0, g1Tail})
		}},
	{name: "enabling grapheme mode handles stored breaks", cols: 5, rows: 1,
		in: "\x1b[?2027la​" + g2027 + "́", screen: scr("a​́"), gap: gGraph},
	{name: "graphemeWidth parity", cols: 80, rows: 5,
		in: g2027 + "❤️", cur: at(2, 0), gap: gGraph},
	{name: "VS16 doesn't make character with 2027 disabled", cols: 5, rows: 5,
		in: "\x1b[?2027l❤️", screen: scr("❤️"),
		check: func(t *Terminal) string { return g1Cell(t, 0, 0, 0x2764) }},
	{name: "print invalid VS16 non-grapheme", cols: 80, rows: 80,
		in: "x️", cur: at(1, 0),
		check: func(t *Terminal) string { return g1Cells(t, [3]int{0, 0, 'x'}, [3]int{1, 0, 0}) }},

	// mode 2027 grapheme clustering
	{name: "variation selectors apply to preceding codepoint", cols: 5, rows: 5,
		in: g2027 + "\U0001F3F4‍☠️", gap: gGraph},
	{name: "print multicodepoint grapheme, mode 2027", cols: 80, rows: 80,
		in: g2027 + g1Fam, cur: at(2, 0), gap: gGraph},
	{name: "keypad sequence VS15", cols: 80, rows: 80, in: g2027 + "#︎", cur: at(1, 0), gap: gGraph},
	{name: "keypad sequence VS16", cols: 80, rows: 80, in: g2027 + "#️", cur: at(2, 0), gap: gGraph},
	{name: "Fitzpatrick skin tone next valid base", cols: 80, rows: 80,
		in: g2027 + "\U0001F44B\U0001F3FF", cur: at(2, 0), gap: gGraph},
	{name: "Fitzpatrick skin tone next to non-base", cols: 80, rows: 80,
		in: g2027 + "\"\U0001F3FF\"", cur: at(4, 0), gap: gGraph},
	{name: "multicodepoint grapheme marks dirty on every codepoint", cols: 80, rows: 80,
		in: g2027 + g1Fam, cur: at(2, 0), gap: gGraph},
	{name: "VS15 to make narrow character", cols: 5, rows: 5,
		in: g2027 + "☔︎", cur: at(1, 0), screen: scr("☔︎"), gap: gGraph},
	{name: "VS15 on already narrow emoji", cols: 5, rows: 5,
		in: g2027 + "⛈︎", cur: at(1, 0), screen: scr("⛈︎"), gap: gGraph},
	{name: "print invalid VS15 following emoji is wide", cols: 80, rows: 80,
		in: g2027 + "\U0001F9E0︎", cur: at(2, 0), gap: gGraph},
	{name: "print invalid VS15 in emoji ZWJ sequence", cols: 80, rows: 80,
		in: g2027 + "\U0001F469︎‍\U0001F466", cur: at(2, 0), gap: gGraph},
	{name: "VS15 to make narrow character with pending wrap", cols: 4, rows: 5,
		in: g2027 + "\U0001F34B☔︎", cur: at(3, 0), pending: no(), gap: gGraph},
	{name: "VS15 narrows wide cell under cursor with wraparound disabled", cols: 5, rows: 5,
		in: g2027 + "\x1b[?7l\x1b[1;4H☔\x1b[?69h\x1b[1;4s\x1b[1;4H︎", cur: at(3, 0), gap: gGraph},
	{name: "VS15 narrows wide cell under restored pending cursor", cols: 5, rows: 5,
		in: g2027 + "\x1b[?69h\x1b[1;4s\x1b[1;4HX\x1b7\x1b[1;5s\x1b[1;4H☔\x1b8︎", cur: at(4, 0), gap: gGraph},
	{name: "VS16 to make wide character on next line", cols: 3, rows: 5,
		in: g2027 + "\x1b[2C#️", cur: at(2, 1), gap: gGraph},
	{name: "VS16 to make wide character on next line with hyperlink", cols: 3, rows: 5,
		in: g2027 + g1Osc8 + "\x1b[2C#️", gap: gGraph},
	{name: "VS16 widening when the spacer tail grows the page", cols: 20, rows: 10,
		in: g2027 + g1Osc8 + "#️", gap: gGraph},
	{name: "grapheme transfer when widening wraps to the next line", cols: 3, rows: 5,
		in: g2027 + "\x1b[2C☺‍❤", gap: gGraph},
	{name: "grapheme transfer when widening wraps below the scroll region", cols: 3, rows: 3,
		in: g2027 + "\x1b[1;2r\x1b[3;3H❤‍\U0001F44D", gap: gGraph},
	{name: "grapheme transfer when widening scrolls a single row", cols: 5, rows: 1,
		in: g2027 + "\x1b[1;5Há\x1bD", gap: gGraph},
	{name: "grapheme transfer when widening wraps to a page without grapheme capacity", cols: 80, rows: 24,
		in: g2027 + "\x1b[24;80H☺‍❤", gap: gGraph},
	{name: "VS16 to make wide character with pending wrap", cols: 3, rows: 5,
		in: g2027 + "\x1b[1C#️", gap: gGraph},
	{name: "VS16 to make wide character with mode 2027", cols: 5, rows: 5,
		in: g2027 + "❤️", cur: at(2, 0), gap: gGraph},
	{name: "VS16 repeated with mode 2027", cols: 5, rows: 5,
		in: g2027 + "❤️❤️", cur: at(4, 0), gap: gGraph},
	{name: "print invalid VS16 grapheme", cols: 80, rows: 80, in: g2027 + "x️", cur: at(1, 0), gap: gGraph},
	{name: "print invalid VS16 with second char", cols: 80, rows: 80, in: g2027 + "x️y", cur: at(2, 0), gap: gGraph},
	{name: "print grapheme o with nonspacing mark should be narrow", cols: 5, rows: 5,
		in: g2027 + "ò", cur: at(1, 0), gap: gGraph},
	{name: "print Devanagari grapheme should be wide", cols: 5, rows: 5,
		in: g2027 + "क्‍ष", cur: at(2, 0), gap: gGraph},
	{name: "print Devanagari grapheme should be wide on next line", cols: 3, rows: 5,
		in: g2027 + "\x1b[2Cक्‍ष", cur: at(2, 1), gap: gGraph},
	{name: "print Devanagari grapheme should be wide on next page", cols: 80, rows: 24,
		in: g2027 + "\x1b[24;80Hक्‍ष", cur: at(2, 23), gap: gGraph},
	{name: "print invalid VS16 with second char (combining)", cols: 80, rows: 80,
		in: g2027 + "n️̃", cur: at(1, 0), gap: gGraph},
	{name: "overwrite grapheme should clear grapheme data", cols: 5, rows: 5,
		in: g2027 + "⛈︎\x1b[1;1HA", screen: scr("A"), gap: gGraph},
	{name: "overwrite multicodepoint grapheme clears grapheme data", cols: 10, rows: 10,
		in: g2027 + g1Fam + "\x1b[1;1HX", cur: at(1, 0), screen: scr("X"), gap: gGraph},
	{name: "overwrite multicodepoint grapheme tail clears grapheme data", cols: 10, rows: 10,
		in: g2027 + g1Fam + "\x1b[1;2HX", cur: at(2, 0), screen: scr(" X"), gap: gGraph},
	{name: "print breaks valid grapheme cluster with Prepend + ASCII for speed", cols: 5, rows: 5,
		in: g2027 + "_؀1", cur: at(3, 0), gap: gGraph},

	// scrollback viewport
	{name: "print writes to bottom if scrolled", cols: 5, rows: 2,
		in: "hello\x1b[H\x1bD\x1bD\x1bD", screen: scr("")},
	{name: "print writes to bottom if scrolled #2", cols: 5, rows: 2,
		in: "hello\x1b[H\x1bD\x1bD\x1bDA", screen: scr("\nA"),
		check: func(t *Terminal) string {
			// upstream: scrolling the viewport to the top shows "hello".
			if sb := t.ScrollbackLines(); len(sb) == 0 || sb[0] != "hello" {
				return fmt.Sprintf("scrollback %q, want first line \"hello\"", sb)
			}
			return ""
		}},

	// charsets
	{name: "print charset", cols: 80, rows: 80,
		in: "\x1b)0\x1b*0\x1b+0`\x1b(B`\x1b(B`\x1b(0`", screen: scr("```◆")},
	{name: "print charset outside of ASCII", cols: 80, rows: 80,
		in: "\x1b)0\x1b*0\x1b+0\x1b(0`\U0001F600", screen: scr("◆\U0001F600")},
	{name: "print wide codepoint in a charset prints it unmapped", cols: 10, rows: 2,
		in: "\x1b(0`\U0001F600a", cur: at(4, 0), screen: scr("◆\U0001F600▒"),
		check: func(t *Terminal) string { return g1Cells(t, [3]int{1, 0, 0x1F600}, [3]int{2, 0, g1Tail}) }},
	{name: "print single shift is used up by a combining character", cols: 10, rows: 2,
		in: "\x1b*A#\x1bŃ#", cur: at(2, 0),
		check: func(t *Terminal) string { return g1Cell(t, 1, 0, '#') }, gap: "national/other charsets"},
	{name: "VS16 moving a character to the next line keeps it unmapped", cols: 3, rows: 5,
		in: g2027 + "\x1b[2C#\x1b(A️", gap: gGraph},
	{name: "print invoke charset", cols: 80, rows: 80,
		in: "\x1b)0`\x0e``\x0f`", screen: scr("`◆◆`")},
	// Upstream single-shifts G1, which has no escape sequence; SS2 with G2 is the equivalent.
	{name: "print invoke charset single", cols: 80, rows: 80,
		in: "\x1b*0`\x1bN``", screen: scr("`◆`")},
	{name: "print kitty unicode placeholder", cols: 10, rows: 10,
		in: "\U0010EEEE", cur: at(1, 0), gap: "images (sixel/kitty)"},

	// wrapping
	{name: "soft wrap", cols: 3, rows: 80, in: "hello", cur: at(2, 1), screen: scr("hel\nlo")},
	// Native Terminal.zig at 46e9dc9a retains primary/continuation row marks.
	{name: "soft wrap with semantic prompt", cols: 3, rows: 80,
		in: "\x1b]133;A\x07hello", cur: at(2, 1), pending: no(), screen: scr("hel\nlo"), scrollback: []string{},
		check: g3all(g1Prompts(PromptPrimary, PromptContinuation),
			g3is("prompt continuation lost its soft wrap", func(t *Terminal) bool { return t.Wrapped(1) && !t.Wrapped(2) }))},
	{name: "disabled wraparound with wide char and one space", cols: 5, rows: 5,
		in: "\x1b[?7lAAAA\U0001F6A8", cur: at(4, 0), screen: scr("AAAA"),
		check: func(t *Terminal) string { return g1Cell(t, 4, 0, 0) }},
	{name: "disabled wraparound with wide char and no space", cols: 5, rows: 5,
		in: "\x1b[?7lAAAAA\U0001F6A8", cur: at(4, 0), screen: scr("AAAAA"),
		check: func(t *Terminal) string { return g1Cell(t, 4, 0, 'A') }},
	{name: "disabled wraparound with wide grapheme and half space", cols: 5, rows: 5,
		in: g2027 + "\x1b[?7lAAAA❤️", cur: at(4, 0), screen: scr("AAAA❤"), gap: gGraph},
	{name: "print with wraparound disabled clears pending wrap after CBT", cols: 6, rows: 2,
		in: "\x1b[?7habcdef\x1b[?7l\x1b[Z", pending: yes()},
	{name: "print with wraparound disabled clears pending wrap after CBT #2", cols: 6, rows: 2,
		in: "\x1b[?7habcdef\x1b[?7l\x1b[Zx", pending: no()},
	{name: "print with wraparound disabled clears pending wrap after CBT #3", cols: 6, rows: 2,
		in: "\x1b[?7habcdef\x1b[?7l\x1b[Zx\x1b[?7hy", cur: at(2, 0), screen: scr("xycdef")},
	{name: "print with wraparound disabled clears pending wrap after CBT #4", cols: 6, rows: 2,
		in: "\x1b[?7labcdef\x1b[?7l\x1b[Z", pending: yes()},
	{name: "print with wraparound disabled clears pending wrap after CBT #5", cols: 6, rows: 2,
		in: "\x1b[?7labcdef\x1b[?7l\x1b[Zx", pending: no()},
	{name: "print with wraparound disabled clears pending wrap after CBT #6", cols: 6, rows: 2,
		in: "\x1b[?7labcdef\x1b[?7l\x1b[Zx\x1b[?7hy", cur: at(2, 0), screen: scr("xycdef")},
	{name: "print right margin wrap", cols: 10, rows: 5,
		in: "123456789\x1b[?69h\x1b[3;5s\x1b[1;5HXY", screen: scr("1234X6789\n  Y"), gap: gLRM},
	{name: "print right margin wrap dirty tracking", cols: 10, rows: 5,
		in: "123456789\x1b[?69h\x1b[3;5s\x1b[1;5HXY", screen: scr("1234X6789\n  Y"), gap: gLRM},
	{name: "print right margin outside", cols: 10, rows: 5,
		in: "123456789\x1b[?69h\x1b[3;5s\x1b[1;6HXY", screen: scr("12345XY89"), gap: gLRM},
	{name: "print right margin outside wrap", cols: 10, rows: 5,
		in: "123456789\x1b[?69h\x1b[3;5s\x1b[1;10HXY", screen: scr("123456789X\n  Y"), gap: gLRM},
	{name: "print wide char at right margin does not create spacer head", cols: 10, rows: 10,
		in: "\x1b[?69h\x1b[3;5s\x1b[1;5H\U0001F600", cur: at(4, 1), gap: gLRM},

	// hyperlinks
	{name: "print with hyperlink", cols: 80, rows: 80, in: g1Osc8 + "123456", screen: scr("123456"),
		check: g1Hyperlinks("http://example.com", "http://example.com", "http://example.com", "http://example.com", "http://example.com", "http://example.com")},
	{name: "print over cell with same hyperlink", cols: 80, rows: 80, in: g1Osc8 + "123456\x1b[1;1H123456", screen: scr("123456"),
		check: g1Hyperlinks("http://example.com", "http://example.com", "http://example.com", "http://example.com", "http://example.com", "http://example.com")},
	{name: "print and end hyperlink", cols: 80, rows: 80, in: g1Osc8 + "123" + g1Osc8e + "456", screen: scr("123456"),
		check: g1Hyperlinks("http://example.com", "http://example.com", "http://example.com", "", "", "")},
	{name: "print and change hyperlink", cols: 80, rows: 80,
		in: "\x1b]8;;http://one.example.com\x1b\\123\x1b]8;;http://two.example.com\x1b\\456", screen: scr("123456"),
		check: g1Hyperlinks("http://one.example.com", "http://one.example.com", "http://one.example.com", "http://two.example.com", "http://two.example.com", "http://two.example.com")},
	{name: "overwrite hyperlink", cols: 80, rows: 80,
		in: "\x1b]8;;http://one.example.com\x1b\\123\x1b[1;1H" + g1Osc8e + "456", screen: scr("456"), check: g1Hyperlinks("", "", "")},
	{name: "print wide char at right edge with hyperlink", cols: 10, rows: 5,
		in: g1Osc8 + "\x1b[1;10H中", cur: at(2, 1),
		check: func(t *Terminal) string {
			return g1Cells(t, [3]int{9, 0, 0}, [3]int{0, 1, 0x4e2d}, [3]int{1, 1, g1Tail})
		}},

	// linefeed / carriage return / backspace
	{name: "linefeed and carriage return", cols: 80, rows: 80,
		in: "hello\r\nworld", cur: at(5, 1), screen: scr("hello\nworld")},
	{name: "linefeed unsets pending wrap", cols: 5, rows: 80, in: "hello", pending: yes()},
	{name: "linefeed unsets pending wrap #2", cols: 5, rows: 80, in: "hello\n", pending: no()},
	{name: "linefeed mode automatic carriage return", cols: 10, rows: 10,
		in: "\x1b[20h123456\nX", screen: scr("123456\nX")},
	{name: "carriage return unsets pending wrap", cols: 5, rows: 80, in: "hello", pending: yes()},
	{name: "carriage return unsets pending wrap #2", cols: 5, rows: 80, in: "hello\r", pending: no()},
	{name: "carriage return origin mode moves to left margin", cols: 5, rows: 80,
		in: "\x1b[?69h\x1b[3;5s\x1b[?6h\x1b[1;1H\r", cur: at(2, 0), gap: gLRM},
	{name: "carriage return left of left margin moves to zero", cols: 5, rows: 80,
		in: "\x1b[?69h\x1b[3;5s\x1b[1;2H\r", cur: at(0, 0), gap: gLRM},
	{name: "carriage return right of left margin moves to left margin", cols: 5, rows: 80,
		in: "\x1b[?69h\x1b[3;5s\x1b[1;4H\r", cur: at(2, 0), gap: gLRM},
	{name: "backspace", cols: 80, rows: 80, in: "hello\by", cur: at(5, 0), screen: scr("helly")},

	// tabs
	{name: "horizontal tabs", cols: 20, rows: 5, in: "1\t", cur: at(8, 0)},
	{name: "horizontal tabs #2", cols: 20, rows: 5, in: "1\t\t", cur: at(16, 0)},
	{name: "horizontal tabs #3", cols: 20, rows: 5, in: "1\t\t\t", cur: at(19, 0)},
	{name: "horizontal tabs #4", cols: 20, rows: 5, in: "1\t\t\t\t", cur: at(19, 0)},
	{name: "horizontal tabs starting on tabstop", cols: 20, rows: 5,
		in: "\x1b[0;9HX\x1b[0;9H\tA", screen: scr("        X       A")},
	{name: "horizontal tabs with right margin", cols: 20, rows: 5,
		in: "\x1b[?69h\x1b[3;6s\x1b[1;1HX\tA", screen: scr("X    A"), gap: gLRM},
	{name: "horizontal tabs back", cols: 20, rows: 5, in: "\x1b[1;20H\x1b[Z", cur: at(16, 0)},
	{name: "horizontal tabs back #2", cols: 20, rows: 5, in: "\x1b[1;20H\x1b[Z\x1b[Z", cur: at(8, 0)},
	{name: "horizontal tabs back #3", cols: 20, rows: 5, in: "\x1b[1;20H\x1b[Z\x1b[Z\x1b[Z", cur: at(0, 0)},
	{name: "horizontal tabs back #4", cols: 20, rows: 5, in: "\x1b[1;20H\x1b[Z\x1b[Z\x1b[Z\x1b[Z", cur: at(0, 0)},
	{name: "horizontal tabs back starting on tabstop", cols: 20, rows: 5,
		in: "\x1b[1;9HX\x1b[1;9H\x1b[ZA", screen: scr("A       X")},
	{name: "horizontal tabs with left margin in origin mode", cols: 20, rows: 5,
		in: "\x1b[?69h\x1b[3;6s\x1b[?6h\x1b[1;2HX\x1b[ZA", screen: scr("  AX"), gap: gLRM},
	{name: "horizontal tab back with cursor before left margin", cols: 20, rows: 5,
		in: "\x1b[?6h\x1b7\x1b[?69h\x1b[5;0s\x1b8\x1b[ZX", screen: scr("X"), gap: gLRM},

	// cursor position
	{name: "cursorPos resets wrap", cols: 5, rows: 5, in: "ABCDE", pending: yes()},
	{name: "cursorPos resets wrap #2", cols: 5, rows: 5, in: "ABCDE\x1b[1;1H", pending: no()},
	{name: "cursorPos resets wrap #3", cols: 5, rows: 5, in: "ABCDE\x1b[1;1HX", screen: scr("XBCDE")},
	{name: "cursorPos off the screen", cols: 5, rows: 5,
		in: "\x1b[500;500HX", screen: scr("\n\n\n\n    X")},
	{name: "cursorPos relative to origin", cols: 5, rows: 5,
		in: "\x1b[3;4r\x1b[?6h\x1b[1;1HX", screen: scr("\n\nX")},
	{name: "cursorPos relative to origin with left/right", cols: 5, rows: 5,
		in: "\x1b[3;4r\x1b[?69h\x1b[3;5s\x1b[?6h\x1b[1;1HX", screen: scr("\n\n  X"), gap: gLRM},
	{name: "cursorPos limits with full scroll region", cols: 5, rows: 5,
		in: "\x1b[3;4r\x1b[?69h\x1b[3;5s\x1b[?6h\x1b[500;500HX", screen: scr("\n\n\n    X"), gap: gLRM},
	{name: "setCursorPos (original test)", cols: 80, rows: 80, in: "", cur: at(0, 0)},
	{name: "setCursorPos (original test) #2", cols: 80, rows: 80, in: "\x1b[0;0H", cur: at(0, 0)},
	{name: "setCursorPos (original test) #3", cols: 80, rows: 80, in: g1SCP3, cur: at(79, 79)},
	{name: "setCursorPos (original test) #4", cols: 80, rows: 80, in: g1SCP4, pending: yes()},
	{name: "setCursorPos (original test) #5", cols: 80, rows: 80, in: g1SCP5, pending: no()},
	{name: "setCursorPos (original test) #6", cols: 80, rows: 80, in: g1SCP6, cur: at(79, 79)},
	{name: "setCursorPos (original test) #7", cols: 80, rows: 80, in: g1SCP7, cur: at(0, 9)},
	{name: "setCursorPos (original test) #8", cols: 80, rows: 80, in: g1SCP8, cur: at(0, 9)},
	{name: "setCursorPos (original test) #9", cols: 80, rows: 80, in: g1SCP9, cur: at(0, 79)},
	{name: "setCursorPos (original test) #10", cols: 80, rows: 80, in: g1SCP10, cur: at(0, 10)},

	// margins
	{name: "setTopAndBottomMargin simple", cols: 5, rows: 5,
		in: g1Grid + "\x1b[0;0r\x1b[1T", screen: scr("\nABC\nDEF\nGHI")},
	{name: "setTopAndBottomMargin top only", cols: 5, rows: 5,
		in: g1Grid + "\x1b[2;0r\x1b[1T", screen: scr("ABC\n\nDEF\nGHI")},
	{name: "setTopAndBottomMargin top and bottom", cols: 5, rows: 5,
		in: g1Grid + "\x1b[1;2r\x1b[1T", screen: scr("\nABC\nGHI")},
	{name: "setTopAndBottomMargin top equal to bottom", cols: 5, rows: 5,
		in: g1Grid + "\x1b[2;2r\x1b[1T", screen: scr("\nABC\nDEF\nGHI")},
	{name: "setLeftAndRightMargin simple", cols: 5, rows: 5,
		in: g1Grid + "\x1b[?69h\x1b[0;0s\x1b[1X", screen: scr(" BC\nDEF\nGHI"), gap: gLRM},
	{name: "setLeftAndRightMargin left only", cols: 5, rows: 5,
		in: g1Grid + "\x1b[?69h\x1b[2;0s\x1b[1;2H\x1b[L", screen: scr("A\nDBC\nGEF\n HI"), gap: gLRM},
	{name: "setLeftAndRightMargin left and right", cols: 5, rows: 5,
		in: g1Grid + "\x1b[?69h\x1b[1;2s\x1b[1;2H\x1b[L", screen: scr("  C\nABF\nDEI\nGH"), gap: gLRM},
	{name: "setLeftAndRightMargin left equal right", cols: 5, rows: 5,
		in: g1Grid + "\x1b[?69h\x1b[2;2s\x1b[1;2H\x1b[L", screen: scr("\nABC\nDEF\nGHI"), gap: gLRM},
	{name: "setLeftAndRightMargin mode 69 unset", cols: 5, rows: 5,
		in: g1Grid + "\x1b[?69l\x1b[1;2s\x1b[1;2H\x1b[L", screen: scr("\nABC\nDEF\nGHI")},

	// insertLines
	{name: "insertLines simple", cols: 5, rows: 5,
		in: g1Grid + "\x1b[2;2H\x1b[L", screen: scr("ABC\n\nDEF\nGHI")},
	{name: "insertLines colors with bg color", cols: 5, rows: 5,
		in: g1Grid + "\x1b[2;2H\x1b[48;2;255;0;0m\x1b[L", screen: scr("ABC\n\nDEF\nGHI"),
		check: func(t *Terminal) string {
			for x := range 5 {
				if c := t.cellAt(x, 1); c.Bg != rgbColor(0xff, 0, 0) {
					return fmt.Sprintf("cell %d,1 bg %#x, want rgb ff0000", x, c.Bg)
				}
			}
			return ""
		}},
	{name: "insertLines handles style refs", cols: 5, rows: 3,
		in: "ABC\r\nDEF\r\n\x1b[1mGHI\x1b[0m\x1b[2;2H\x1b[L", screen: scr("ABC\n\nDEF")},
	{name: "insertLines outside of scroll region", cols: 5, rows: 5,
		in: g1Grid + "\x1b[3;4r\x1b[2;2H\x1b[L", screen: scr("ABC\nDEF\nGHI")},
	{name: "insertLines top/bottom scroll region", cols: 5, rows: 5,
		in: g1Grid + "\r\n123\x1b[1;3r\x1b[2;2H\x1b[L", screen: scr("ABC\n\nDEF\n123")},
	// Upstream linefeeds until the content straddles a page boundary; the
	// count only matters to its page list, so any count >= rows will do.
	{name: "insertLines across page boundary marks all shifted rows dirty", cols: 10, rows: 5,
		in:     strings.Repeat("\n", 300) + "1AAAA\r\n2BBBB\r\n3CCCC\r\n4DDDD\r\n5EEEE\x1b[1;1H\x1b[L",
		screen: scr("\n1AAAA\n2BBBB\n3CCCC\n4DDDD")},
	{name: "insertLines hyperlink-dense row crosses page boundary", cols: 10, rows: 5,
		in: strings.Repeat("\n", 300) + "\x1b[1;1H0\x1b[2;1H1\x1b[4;1H3\x1b[5;1H4\x1b[3;1H" +
			g1DenseLinks() + "\x1b[1;1H\x1b[L",
		screen: scr("\n0\n1\nABCDEFGHIJ\n3")},
	{name: "insertLines (legacy test)", cols: 2, rows: 5,
		in: "A\r\nB\r\nC\r\nD\r\nE\x1b[2;1H\x1b[2L", screen: scr("A\n\n\nB\nC")},
}

// Translated from ghostty (MIT), src/terminal/Terminal.zig at commit 521cf2c
// (tests "insertLines with scroll region" through "insert mode doesn't wrap
// pushed characters").

func init() { registerSuite("ghostty", ghostty2Cases) }

const (
	g2Red     = "\x1b[48;2;255;0;0m"
	g2Link    = "\x1b]8;;http://example.com\x1b\\"
	g2LinkEnd = "\x1b]8;;\x1b\\"
	g2Family  = "\U0001F468‍\U0001F469‍\U0001F467"
)

// g2BgRed checks that cells [from, to) of row y have a red direct-color
// background.
func g2BgRed(y, from, to int) func(t *Terminal) string {
	return func(t *Terminal) string {
		want := rgbColor(0xFF, 0, 0)
		for x := from; x < to; x++ {
			if c := t.cellAt(x, y); c.Bg != want {
				return fmt.Sprintf("cell %d,%d bg %#x, want red %#x", x, y, c.Bg, want)
			}
		}
		return ""
	}
}

// g2Cell checks the rune and attributes of one cell, and that its colors
// are the default.
func g2Cell(x, y int, r rune, a Attr) func(t *Terminal) string {
	return func(t *Terminal) string {
		c := t.cellAt(x, y)
		if c.R != r || c.A != a || c.Fg != 0 || c.Bg != 0 {
			return fmt.Sprintf("cell %d,%d = %+v, want rune %q attr %#x default colors", x, y, c, r, a)
		}
		return ""
	}
}

// g2DenseLinks builds the "hyperlink-dense row crosses page boundary" input.
func g2DenseLinks() string {
	var b strings.Builder
	b.WriteString(strings.Repeat("\n", 300))
	b.WriteString("\x1b[1;1H0\x1b[2;1H1\x1b[3;1H2\x1b[5;1H4\x1b[4;1H")
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&b, "\x1b]8;;http://example.com/%d\x1b\\%c%s", i, 'A'+i, g2LinkEnd)
	}
	b.WriteString("\x1b[1;1H\x1b[M")
	return b.String()
}

const g2ABCDEFGHI = "ABC\r\nDEF\r\nGHI"
const g2Cols123 = "ABC123\r\nDEF456\r\nGHI789"
const g2SpacerHead = "AAAAABBBB\U0001F600CCC"

var ghostty2Cases = []confCase{
	// insertLines
	{name: "insertLines with scroll region", cols: 2, rows: 6,
		in:     "A\r\nB\r\nC\r\nD\r\nE\x1b[1;2r\x1b[1;1H\x1b[LX",
		screen: scr("X\nA\nC\nD\nE")},
	{name: "insertLines more than remaining", cols: 2, rows: 5,
		in:     "A\r\nB\r\nC\r\nD\r\nE\x1b[2;1H\x1b[20L",
		screen: scr("A")},
	{name: "insertLines resets pending wrap", cols: 5, rows: 5,
		in:     "ABCDE\x1b[LB",
		screen: scr("B\nABCDE")},
	{name: "insertLines resets pending wrap#2", cols: 5, rows: 5,
		in: "ABCDE", pending: yes()},
	{name: "insertLines resets pending wrap#3", cols: 5, rows: 5,
		in: "ABCDE\x1b[L", pending: no()},
	{name: "insertLines resets wrap", cols: 3, rows: 3,
		in:     "1\r\nABCDEF\x1b[1;1H\x1b[LX",
		screen: scr("X\n1\nABC")},
	{name: "insertLines multi-codepoint graphemes", cols: 5, rows: 5,
		in:     "\x1b[?2027hABC\r\n" + g2Family + "\r\nGHI\x1b[2;2H\x1b[L",
		screen: scr("ABC\n\n" + g2Family + "\nGHI"),
		gap:    "grapheme clustering (mode 2027, VS15/VS16 width changes)"},
	{name: "insertLines left/right scroll region", cols: 10, rows: 10,
		in:     g2Cols123 + "\x1b[?69h\x1b[2;4s\x1b[2;2H\x1b[L",
		screen: scr("ABC123\nD   56\nGEF489\n HI7"),
		gap:    "DECLRMM left/right margins"},

	// scrollUp (SU)
	{name: "scrollUp simple", cols: 5, rows: 5,
		in:     g2ABCDEFGHI + "\x1b[2;2H\x1b[S",
		screen: scr("DEF\nGHI"), cur: at(1, 1)},
	{name: "scrollUp moves hyperlink", cols: 5, rows: 5,
		in:     "ABC\r\n" + g2Link + "DEF" + g2LinkEnd + "\r\nGHI\x1b[2;2H\x1b[S",
		screen: scr("DEF\nGHI")},
	{name: "scrollUp clears hyperlink", cols: 5, rows: 5,
		in:     g2Link + "ABC" + g2LinkEnd + "\r\nDEF\r\nGHI\x1b[2;2H\x1b[S",
		screen: scr("DEF\nGHI")},
	{name: "scrollUp top/bottom scroll region", cols: 5, rows: 5,
		in:     g2ABCDEFGHI + "\x1b[2;3r\x1b[1;1H\x1b[S",
		screen: scr("ABC\nGHI")},
	{name: "scrollUp left/right scroll region", cols: 10, rows: 10,
		in:     g2Cols123 + "\x1b[?69h\x1b[2;4s\x1b[2;2H\x1b[S",
		screen: scr("AEF423\nDHI756\nG   89"), cur: at(1, 1),
		gap: "DECLRMM left/right margins"},
	{name: "scrollUp left/right scroll region hyperlink", cols: 10, rows: 10,
		in:     "ABC123\r\n" + g2Link + "DEF456" + g2LinkEnd + "\r\nGHI789\x1b[?69h\x1b[2;4s\x1b[2;2H\x1b[S",
		screen: scr("AEF423\nDHI756\nG   89"),
		gap:    "DECLRMM left/right margins"},
	{name: "scrollUp preserves pending wrap", cols: 5, rows: 5,
		in:     "\x1b[1;5HA\x1b[2;5HB\x1b[3;5HC\x1b[SX",
		screen: scr("    B\n    C\n\nX")},
	{name: "scrollUp full top/bottom region", cols: 5, rows: 5,
		in:     "top\x1b[5;1HABCDE\x1b[2;5r\x1b[4S",
		screen: scr("top")},
	{name: "scrollUp full top/bottomleft/right scroll region", cols: 5, rows: 5,
		in:     "top\x1b[5;1HABCDE\x1b[?69h\x1b[2;5r\x1b[2;4s\x1b[4S",
		screen: scr("top\n\n\n\nA   E"),
		gap:    "DECLRMM left/right margins"},
	{name: "scrollUp creates scrollback in primary screen", cols: 5, rows: 5,
		in:         "AAAAA\r\nBBBBB\r\nCCCCC\r\nDDDDD\r\nEEEEE\x1b[S",
		screen:     scr("BBBBB\nCCCCC\nDDDDD\nEEEEE"),
		scrollback: []string{"AAAAA"}},
	{name: "scrollUp with max_scrollback_bytes zero", cols: 5, rows: 5,
		in:     "AAAAA\r\nBBBBB\r\nCCCCC\x1b[S",
		screen: scr("BBBBB\nCCCCC")},
	{name: "scrollUp with max_scrollback_bytes zero and top margin", cols: 5, rows: 5,
		in:     "AAAAA\r\nBBBBB\r\nCCCCC\r\nDDDDD\x1b[2;5r\x1b[S",
		screen: scr("AAAAA\nCCCCC\nDDDDD")},
	{name: "scrollUp with max_scrollback_bytes zero and left/right margin", cols: 10, rows: 5,
		in:     "AAAAABBBBB\r\nCCCCCDDDDD\r\nEEEEEFFFFF\x1b[?69h\x1b[2;6s\x1b[S",
		screen: scr("ACCCCDBBBB\nCEEEEFDDDD\nE     FFFF"),
		gap:    "DECLRMM left/right margins"},

	// scrollDown (SD)
	{name: "scrollDown simple", cols: 5, rows: 5,
		in:     g2ABCDEFGHI + "\x1b[2;2H\x1b[T",
		screen: scr("\nABC\nDEF\nGHI"), cur: at(1, 1)},
	{name: "scrollDown hyperlink moves", cols: 5, rows: 5,
		in:     g2Link + "ABC" + g2LinkEnd + "\r\nDEF\r\nGHI\x1b[2;2H\x1b[T",
		screen: scr("\nABC\nDEF\nGHI")},
	{name: "scrollDown outside of scroll region", cols: 5, rows: 5,
		in:     g2ABCDEFGHI + "\x1b[3;4r\x1b[2;2H\x1b[T",
		screen: scr("ABC\nDEF\n\nGHI"), cur: at(1, 1)},
	{name: "scrollDown left/right scroll region", cols: 10, rows: 10,
		in:     g2Cols123 + "\x1b[?69h\x1b[2;4s\x1b[2;2H\x1b[T",
		screen: scr("A   23\nDBC156\nGEF489\n HI7"), cur: at(1, 1),
		gap: "DECLRMM left/right margins"},
	{name: "scrollDown left/right scroll region hyperlink", cols: 10, rows: 10,
		in:     g2Link + "ABC123" + g2LinkEnd + "\r\nDEF456\r\nGHI789\x1b[?69h\x1b[2;4s\x1b[2;2H\x1b[T",
		screen: scr("A   23\nDBC156\nGEF489\n HI7"),
		gap:    "DECLRMM left/right margins"},
	{name: "scrollDown outside of left/right scroll region", cols: 10, rows: 10,
		in:     g2Cols123 + "\x1b[?69h\x1b[2;4s\x1b[1;1H\x1b[T",
		screen: scr("A   23\nDBC156\nGEF489\n HI7"), cur: at(0, 0),
		gap: "DECLRMM left/right margins"},
	{name: "scrollDown preserves pending wrap", cols: 5, rows: 10,
		in:     "\x1b[1;5HA\x1b[2;5HB\x1b[3;5HC\x1b[TX",
		screen: scr("\n    A\n    B\nX   C")},

	// eraseChars (ECH)
	{name: "eraseChars simple operation", cols: 5, rows: 5,
		in: "ABC\x1b[1;1H\x1b[2XX", screen: scr("X C")},
	{name: "eraseChars minimum one", cols: 5, rows: 5,
		in: "ABC\x1b[1;1H\x1b[0XX", screen: scr("XBC")},
	{name: "eraseChars beyond screen edge", cols: 5, rows: 5,
		in: "  ABC\x1b[1;4H\x1b[10X", screen: scr("  A")},
	{name: "eraseChars wide character", cols: 5, rows: 5,
		in: "橋BC\x1b[1;1H\x1b[1XX", screen: scr("X BC")},
	{name: "eraseChars resets pending wrap", cols: 5, rows: 5,
		in: "ABCDE\x1b[1XX", screen: scr("ABCDX")},
	{name: "eraseChars resets pending wrap#2", cols: 5, rows: 5,
		in: "ABCDE", pending: yes()},
	{name: "eraseChars resets pending wrap#3", cols: 5, rows: 5,
		in: "ABCDE\x1b[1X", pending: no()},
	{name: "eraseChars resets wrap", cols: 5, rows: 5,
		in: "ABCDE123\x1b[1;1H\x1b[1XX", screen: scr("XBCDE\n123")},
	{name: "eraseChars preserves background sgr", cols: 10, rows: 10,
		in:     "ABC\x1b[1;1H" + g2Red + "\x1b[2X",
		screen: scr("  C"), check: g2BgRed(0, 0, 2)},
	{name: "eraseChars protected attributes respected with iso", cols: 5, rows: 5,
		in: "\x1bVABC\x1b[1;1H\x1b[2X", screen: scr("ABC"),
		gap: "protected cells (DECSCA/SPA, DECSED/DECSEL)"},
	{name: "eraseChars protected attributes ignored with dec most recent", cols: 5, rows: 5,
		in: "\x1bVABC\x1b[1\"q\x1b[0\"q\x1b[1;1H\x1b[2X", screen: scr("  C"),
		gap: "protected cells (DECSCA/SPA, DECSED/DECSEL)"},
	{name: "eraseChars protected attributes ignored with dec set", cols: 5, rows: 5,
		in: "\x1b[1\"qABC\x1b[1;1H\x1b[2X", screen: scr("  C"),
		gap: "protected cells (DECSCA/SPA, DECSED/DECSEL)"},
	{name: "eraseChars wide char boundary conditions", cols: 8, rows: 1,
		in: "😀a😀b😀\x1b[1;2H\x1b[3X", screen: scr("     b😀")},
	{name: "eraseChars wide char boundary conditions#2", cols: 8, rows: 1,
		in: "😀a😀b😀", screen: scr("😀a😀b😀")},
	{name: "eraseChars wide char splits proper cell boundaries", cols: 30, rows: 1,
		in: "x食べて下さい\x1b[1;6H\x1b[4X", screen: scr("x食べ    さい")},
	{name: "eraseChars wide char splits proper cell boundaries#2", cols: 30, rows: 1,
		in: "x食べて下さい", screen: scr("x食べて下さい")},
	{name: "eraseChars wide char wrap boundary conditions", cols: 8, rows: 3,
		in:     ".......😀abcde😀......\x1b[2;2H\x1b[3X",
		screen: scr(".......\n    cde\n😀......")},
	{name: "eraseChars wide char wrap boundary conditions#2", cols: 8, rows: 3,
		in:     ".......😀abcde😀......",
		screen: scr(".......\n😀abcde\n😀......")},

	// reverseIndex (RI)
	{name: "reverseIndex", cols: 2, rows: 5,
		in: "A\r\nB\r\nC\x1bMD\r\n\r\n", screen: scr("A\nBD\nC")},
	{name: "reverseIndex from the top", cols: 2, rows: 5,
		in:     "A\r\nB\r\n\r\n\x1b[1;1H\x1bMD\r\n\x1b[1;1H\x1bME\r\n",
		screen: scr("E\nD\nA\nB")},
	{name: "reverseIndex top of scrolling region", cols: 2, rows: 10,
		in:     "\x1b[2;1HA\r\nB\r\nC\r\nD\r\n\x1b[2;5r\x1b[2;1H\x1bMX",
		screen: scr("\nX\nA\nB\nC")},
	{name: "reverseIndex top of screen", cols: 5, rows: 5,
		in:     "A\x1b[2;1HB\x1b[3;1HC\x1b[1;1H\x1bMX",
		screen: scr("X\nA\nB\nC")},
	{name: "reverseIndex not top of screen", cols: 5, rows: 5,
		in:     "A\x1b[2;1HB\x1b[3;1HC\x1b[2;1H\x1bMX",
		screen: scr("X\nB\nC")},
	{name: "reverseIndex top/bottom margins", cols: 5, rows: 5,
		in:     "A\x1b[2;1HB\x1b[3;1HC\x1b[2;3r\x1b[2;1H\x1bM",
		screen: scr("A\n\nB")},
	{name: "reverseIndex outside top/bottom margins", cols: 5, rows: 5,
		in:     "A\x1b[2;1HB\x1b[3;1HC\x1b[2;3r\x1b[1;1H\x1bM",
		screen: scr("A\nB\nC")},
	{name: "reverseIndex left/right margins", cols: 5, rows: 5,
		in:     "ABC\x1b[2;1HDEF\x1b[3;1HGHI\x1b[?69h\x1b[2;3s\x1b[1;2H\x1bM",
		screen: scr("A\nDBC\nGEF\n HI"),
		gap:    "DECLRMM left/right margins"},
	{name: "reverseIndex outside left/right margins", cols: 5, rows: 5,
		in:     "ABC\x1b[2;1HDEF\x1b[3;1HGHI\x1b[?69h\x1b[2;3s\x1b[1;1H\x1bM",
		screen: scr("ABC\nDEF\nGHI"),
		gap:    "DECLRMM left/right margins"},

	// index (IND)
	{name: "index", cols: 2, rows: 5,
		in: "\x1bDA", screen: scr("\nA")},
	{name: "index from the bottom", cols: 2, rows: 5,
		in: "\x1b[5;1HA\x1b[1D\x1bDB", screen: scr("\n\n\nA\nB")},
	{name: "index scrolling with hyperlink", cols: 2, rows: 5,
		in:     "\x1b[5;1H" + g2Link + "A" + g2LinkEnd + "\x1b[1D\x1bDB",
		screen: scr("\n\n\nA\nB")},
	{name: "index outside of scrolling region", cols: 2, rows: 5,
		in: "\x1b[2;5r\x1bD", cur: at(0, 1)},
	{name: "index from the bottom outside of scroll region", cols: 2, rows: 5,
		in: "\x1b[1;2r\x1b[5;1HA\x1bDB", screen: scr("\n\n\n\nAB")},
	{name: "index no scroll region, top of screen", cols: 5, rows: 5,
		in: "A\x1bDX", screen: scr("A\n X")},
	{name: "index bottom of primary screen", cols: 5, rows: 5,
		in: "\x1b[5;1HA\x1bDX", screen: scr("\n\n\nA\n X")},
	{name: "index bottom of primary screen background sgr", cols: 5, rows: 5,
		in:     "\x1b[5;1HA" + g2Red + "\x1bD",
		screen: scr("\n\n\nA"), check: g2BgRed(4, 0, 5)},
	{name: "index inside scroll region", cols: 5, rows: 5,
		in: "\x1b[1;3rA\x1bDX", screen: scr("A\n X")},
	{name: "index bottom of scroll region with hyperlinks", cols: 5, rows: 5,
		in:     "\x1b[1;2rA\x1bD\r" + g2Link + "B" + g2LinkEnd + "\x1bD\rC",
		screen: scr("B\nC")},
	{name: "index bottom of scroll region clear hyperlinks", cols: 5, rows: 5,
		in:     "\x1b[2;3r\x1b[2;1H" + g2Link + "A" + g2LinkEnd + "\x1bD\rB\x1bD\rC",
		screen: scr("\nB\nC")},
	{name: "index bottom of scroll region with background SGR", cols: 5, rows: 5,
		in:     "\x1b[1;3r\x1b[4;1HB\x1b[3;1HA" + g2Red + "\x1bD",
		screen: scr("\nA\n\nB"), check: g2BgRed(2, 0, 5)},
	{name: "index bottom of primary screen with scroll region", cols: 5, rows: 5,
		in:     "\x1b[1;3r\x1b[3;1HA\x1b[5;1H\x1bD\x1bD\x1bDX",
		screen: scr("\n\nA\n\nX")},
	{name: "index outside left/right margin", cols: 10, rows: 5,
		in:     "\x1b[1;3r\x1b[?69h\x1b[4;6s\x1b[3;3HA\x1b[3;1H\x1bDX",
		screen: scr("\n\nX A"),
		gap:    "DECLRMM left/right margins"},
	{name: "index inside left/right margin", cols: 10, rows: 5,
		in:     "AAAAAA\r\nAAAAAA\r\nAAAAAA\x1b[?69h\x1b[1;3r\x1b[1;3s\x1b[3;1H\x1bD",
		screen: scr("AAAAAA\nAAAAAA\n   AAA"), cur: at(0, 2),
		gap: "DECLRMM left/right margins"},
	{name: "index bottom of scroll region creates scrollback", cols: 5, rows: 5,
		in:         "\x1b[1;3r1\r\n2\r\n3\x1b[4;1HX\x1b[3;1H\x1bDY",
		screen:     scr("2\n3\nY\nX"),
		scrollback: []string{"1"}},
	{name: "index bottom of scroll region no scrollback", cols: 5, rows: 5,
		in:     "\x1b[1;3r\x1b[4;1HB\x1b[3;1HA\x1bDX",
		screen: scr("\nA\n X\nB")},
	{name: "index bottom of scroll region blank line preserves SGR", cols: 5, rows: 5,
		in:         "\x1b[1;3r1\r\n2\r\n3\x1b[4;1HX\x1b[3;1H" + g2Red + "\x1bD",
		screen:     scr("2\n3\n\nX"),
		scrollback: []string{"1"},
		check:      g2BgRed(2, 0, 5)},
	{name: "index bottom of scroll region with top margin and background SGR", cols: 5, rows: 5,
		in:     "1\r\n2\r\n3\r\n4\r\n5\x1b[2;4r\x1b[4;1H" + g2Red + "\x1bD",
		screen: scr("1\n3\n4\n\n5"), cur: at(0, 3), check: g2BgRed(3, 0, 5)},
	{name: "index bottom of alt screen full region", cols: 5, rows: 3,
		in:         "\x1b[?1049hA\r\nB\r\nC\x1bD\rD",
		screen:     scr("B\nC\nD"),
		scrollback: []string{}, alt: yes()},
	{name: "index bottom of alt screen full region#2", cols: 5, rows: 3,
		in:     "\x1b[?1049hA\r\nB\r\nC\x1bD\rD\x1b[?1049l",
		screen: scr(""), alt: no()},
	{name: "index bottom of alt screen top region", cols: 5, rows: 5,
		in:         "\x1b[?1049h1\r\n2\r\n3\r\n4\r\n5\x1b[1;4r\x1b[4;1H\x1bDX",
		screen:     scr("2\n3\n4\nX\n5"),
		scrollback: []string{}},
	{name: "scrollUp top region no scrollback", cols: 5, rows: 5,
		in:     "A\r\nB\r\nC\r\nD\r\nE\x1b[1;3r\x1b[S",
		screen: scr("B\nC\n\nD\nE")},

	// cursorUp (CUU)
	{name: "cursorUp basic", cols: 5, rows: 5,
		in: "\x1b[3;1HA\x1b[10AX", screen: scr(" X\n\nA")},
	{name: "cursorUp below top scroll margin", cols: 5, rows: 5,
		in: "\x1b[2;4r\x1b[3;1HA\x1b[5AX", screen: scr("\n X\nA")},
	{name: "cursorUp above top scroll margin", cols: 5, rows: 5,
		in: "\x1b[3;5r\x1b[3;1HA\x1b[2;1H\x1b[10AX", screen: scr("X\n\nA")},
	{name: "cursorUp resets wrap", cols: 5, rows: 5,
		in: "ABCDE\x1b[1AX", screen: scr("ABCDX")},
	{name: "cursorUp resets wrap#2", cols: 5, rows: 5,
		in: "ABCDE\x1b[1A", pending: no()},

	// cursorLeft (CUB) and reverse wrap
	{name: "cursorLeft no wrap", cols: 10, rows: 5,
		in: "A\r\nB\x1b[10D", screen: scr("A\nB")},
	{name: "cursorLeft stops at left margin", cols: 5, rows: 5,
		in: "\x1b[?69h\x1b[3;5s\x1b[1;4H\x1b[100DX", screen: scr("  X"),
		gap: "DECLRMM left/right margins"},
	{name: "cursorLeft left of left margin", cols: 5, rows: 5,
		in: "\x1b[?69h\x1b[4;5s\x1b[1;3H\x1b[100DX", screen: scr("X"),
		gap: "DECLRMM left/right margins"},
	{name: "backspace at left margin", cols: 5, rows: 5,
		in: "\x1b[?69h\x1b[3;5s\x1b[1;3H\bX", screen: scr("  X"),
		gap: "DECLRMM left/right margins"},
	{name: "cursorLeft unsets pending wrap state", cols: 5, rows: 5,
		in: "ABCDE\x1b[1DX", screen: scr("ABCXE")},
	{name: "cursorLeft unsets pending wrap state#2", cols: 5, rows: 5,
		in: "ABCDE\x1b[1D", pending: no()},
	{name: "cursorLeft unsets pending wrap state with longer jump", cols: 5, rows: 5,
		in: "ABCDE\x1b[3DX", screen: scr("AXCDE")},
	{name: "cursorLeft unsets pending wrap state with longer jump#2", cols: 5, rows: 5,
		in: "ABCDE\x1b[3D", pending: no()},
	{name: "cursorLeft reverse wrap with pending wrap state", cols: 5, rows: 5,
		in: "\x1b[?7h\x1b[?45hABCDE\x1b[1DX", screen: scr("ABCDX")},
	{name: "cursorLeft reverse wrap with pending wrap state#2", cols: 5, rows: 5,
		in: "\x1b[?7h\x1b[?45hABCDE\x1b[1D", pending: no()},
	{name: "cursorLeft reverse wrap with pending wrap above top margin", cols: 5, rows: 5,
		in:     "\x1b[?7h\x1b[?45h\x1b[?69h\x1b[1;2sAB\x1b7\x1b[2;5s\x1b[3;5r\x1b8\x1b[1DX",
		screen: scr("AX"),
		gap:    "DECLRMM left/right margins"},
	{name: "cursorLeft reverse wrap extended with pending wrap state", cols: 5, rows: 5,
		in: "\x1b[?7h\x1b[?1045hABCDE\x1b[1DX", screen: scr("ABCDX")},
	{name: "cursorLeft reverse wrap extended with pending wrap state#2", cols: 5, rows: 5,
		in: "\x1b[?7h\x1b[?1045hABCDE\x1b[1D", pending: no()},
	{name: "cursorLeft reverse wrap", cols: 5, rows: 5,
		in:     "\x1b[?7h\x1b[?45hABCDE1\x1b[2DX",
		screen: scr("ABCDX\n1"), pending: yes()},
	{name: "cursorLeft reverse wrap with no soft wrap", cols: 5, rows: 5,
		in:     "\x1b[?7h\x1b[?45hABCDE\r\n1\x1b[2DX",
		screen: scr("ABCDE\nX")},
	{name: "cursorLeft reverse wrap before left margin", cols: 5, rows: 5,
		in:     "\x1b[?7h\x1b[?45h\x1b[3r\x1b[1DX",
		screen: scr("\n\nX")},
	{name: "cursorLeft extended reverse wrap", cols: 5, rows: 5,
		in:     "\x1b[?7h\x1b[?1045hABCDE\r\n1\x1b[2DX",
		screen: scr("ABCDX\n1")},
	{name: "cursorLeft extended reverse wrap bottom wraparound", cols: 5, rows: 3,
		in:     "\x1b[?7h\x1b[?1045hABCDE\r\n1\x1b[7DX",
		screen: scr("ABCDE\n1\n    X")},
	{name: "cursorLeft extended reverse wrap is priority if both set", cols: 5, rows: 3,
		in:     "\x1b[?7h\x1b[?45h\x1b[?1045hABCDE\r\n1\x1b[7DX",
		screen: scr("ABCDE\n1\n    X")},
	{name: "cursorLeft extended reverse wrap above top scroll region", cols: 5, rows: 5,
		in:  "\x1b[?7h\x1b[?1045h\x1b[3r\x1b[2;1H\x1b[1000D",
		cur: at(0, 0)},
	{name: "cursorLeft reverse wrap on first row", cols: 5, rows: 5,
		in:  "\x1b[?7h\x1b[?45h\x1b[3r\x1b[1;2H\x1b[1000D",
		cur: at(0, 0)},

	// cursorDown (CUD) / cursorRight (CUF)
	{name: "cursorDown basic", cols: 5, rows: 5,
		in: "A\x1b[10BX", screen: scr("A\n\n\n\n X")},
	{name: "cursorDown above bottom scroll margin", cols: 5, rows: 5,
		in: "\x1b[1;3rA\x1b[10BX", screen: scr("A\n\n X")},
	{name: "cursorDown below bottom scroll margin", cols: 5, rows: 5,
		in: "\x1b[1;3rA\x1b[4;1H\x1b[10BX", screen: scr("A\n\n\n\nX")},
	{name: "cursorDown resets wrap", cols: 5, rows: 5,
		in: "ABCDE\x1b[1BX", screen: scr("ABCDE\n    X")},
	{name: "cursorDown resets wrap#2", cols: 5, rows: 5,
		in: "ABCDE\x1b[1B", pending: no()},
	{name: "cursorRight resets wrap", cols: 5, rows: 5,
		in: "ABCDE\x1b[1CX", screen: scr("ABCDX")},
	{name: "cursorRight resets wrap#2", cols: 5, rows: 5,
		in: "ABCDE\x1b[1C", pending: no()},
	{name: "cursorRight to the edge of screen", cols: 5, rows: 5,
		in: "\x1b[100CX", screen: scr("    X")},
	{name: "cursorRight left of right margin", cols: 5, rows: 5,
		in: "\x1b[?69h\x1b[1;3s\x1b[100CX", screen: scr("  X"),
		gap: "DECLRMM left/right margins"},
	{name: "cursorRight right of right margin", cols: 5, rows: 5,
		in: "\x1b[?69h\x1b[1;3s\x1b[1;4H\x1b[100CX", screen: scr("    X"),
		gap: "DECLRMM left/right margins"},

	// deleteLines (DL)
	{name: "deleteLines simple", cols: 5, rows: 5,
		in: g2ABCDEFGHI + "\x1b[2;2H\x1b[M", screen: scr("ABC\nGHI")},
	{name: "deleteLines colors with bg color", cols: 5, rows: 5,
		in:     g2ABCDEFGHI + "\x1b[2;2H" + g2Red + "\x1b[M",
		screen: scr("ABC\nGHI"), check: g2BgRed(4, 0, 5)},
	{name: "deleteLines across page boundary marks all shifted rows dirty", cols: 10, rows: 5,
		in:     strings.Repeat("\n", 300) + "1AAAA\r\n2BBBB\r\n3CCCC\r\n4DDDD\r\n5EEEE\x1b[1;1H\x1b[M",
		screen: scr("2BBBB\n3CCCC\n4DDDD\n5EEEE")},
	{name: "deleteLines hyperlink-dense row crosses page boundary", cols: 10, rows: 5,
		in:     g2DenseLinks(),
		screen: scr("1\n2\nABCDEFGHIJ\n4")},
	{name: "deleteLines (legacy)", cols: 80, rows: 80,
		in:     "A\r\nB\r\nC\r\nD\x1b[2A\x1b[ME\r\n",
		screen: scr("A\nE\nD"), cur: at(0, 2)},
	{name: "deleteLines with scroll region", cols: 80, rows: 80,
		in:     "A\r\nB\r\nC\r\nD\x1b[1;3r\x1b[1;1H\x1b[ME\r\n",
		screen: scr("E\nC\n\nD")},
	{name: "deleteLines with scroll region, large count", cols: 80, rows: 80,
		in:     "A\r\nB\r\nC\r\nD\x1b[1;3r\x1b[1;1H\x1b[5ME\r\n",
		screen: scr("E\n\n\nD")},
	{name: "deleteLines with scroll region, cursor outside of region", cols: 80, rows: 80,
		in:     "A\r\nB\r\nC\r\nD\x1b[1;3r\x1b[4;1H\x1b[M",
		screen: scr("A\nB\nC\nD")},
	{name: "deleteLines resets pending wrap", cols: 5, rows: 5,
		in: "ABCDE\x1b[MB", screen: scr("B")},
	{name: "deleteLines resets pending wrap#2", cols: 5, rows: 5,
		in: "ABCDE\x1b[M", pending: no()},
	{name: "deleteLines resets wrap", cols: 3, rows: 3,
		in:     "1\r\nABCDEF\x1b[1;2r\x1b[1;1H\x1b[MX",
		screen: scr("XBC\n\nDEF")},
	{name: "deleteLines left/right scroll region", cols: 10, rows: 10,
		in:     g2Cols123 + "\x1b[?69h\x1b[2;4s\x1b[2;2H\x1b[M",
		screen: scr("ABC123\nDHI756\nG   89"),
		gap:    "DECLRMM left/right margins"},
	{name: "deleteLines left/right scroll region from top", cols: 10, rows: 10,
		in:     g2Cols123 + "\x1b[?69h\x1b[2;4s\x1b[1;2H\x1b[M",
		screen: scr("AEF423\nDHI756\nG   89"),
		gap:    "DECLRMM left/right margins"},
	{name: "deleteLines left/right scroll region high count", cols: 10, rows: 10,
		in:     g2Cols123 + "\x1b[?69h\x1b[2;4s\x1b[2;2H\x1b[100M",
		screen: scr("ABC123\nD   56\nG   89"),
		gap:    "DECLRMM left/right margins"},
	{name: "deleteLines wide character spacer head", cols: 5, rows: 3,
		in:     g2SpacerHead + "\x1b[1;1H\x1b[M",
		screen: scr("BBBB\n\U0001F600CCC")},
	{name: "deleteLines wide character spacer head left scroll margin", cols: 5, rows: 3,
		in:     g2SpacerHead + "\x1b[?69h\x1b[3;5s\x1b[1;3H\x1b[M",
		screen: scr("AABB\nBBCCC\n\U0001F600"),
		gap:    "DECLRMM left/right margins"},
	{name: "deleteLines wide character spacer head right scroll margin", cols: 5, rows: 3,
		in:     g2SpacerHead + "\x1b[?69h\x1b[1;4s\x1b[1;1H\x1b[M",
		screen: scr("BBBBA\n\U0001F600CC\n    C"),
		gap:    "DECLRMM left/right margins"},
	{name: "deleteLines wide character spacer head left and right scroll margin", cols: 5, rows: 3,
		in:     g2SpacerHead + "\x1b[?69h\x1b[3;4s\x1b[1;3H\x1b[M",
		screen: scr("AABBA\nBBCC\n\U0001F600  C"),
		gap:    "DECLRMM left/right margins"},
	{name: "deleteLines wide character spacer head left (< 2) and right scroll margin", cols: 5, rows: 3,
		in:     g2SpacerHead + "\x1b[?69h\x1b[2;4s\x1b[1;2H\x1b[M",
		screen: scr("ABBBA\nB CC\n    C"),
		gap:    "DECLRMM left/right margins"},
	{name: "deleteLines wide characters split by left/right scroll region boundaries", cols: 5, rows: 2,
		in:     "AAAAA\r\n\U0001F600B\U0001F600\x1b[?69h\x1b[2;4s\x1b[1;2H\x1b[M",
		screen: scr("A B A"),
		gap:    "DECLRMM left/right margins"},

	// styles
	{name: "default style is empty", cols: 5, rows: 5,
		in: "A", check: g2Cell(0, 0, 'A', 0)},
	{name: "bold style", cols: 5, rows: 5,
		in: "\x1b[1mA", check: g2Cell(0, 0, 'A', AttrBold)},
	{name: "garbage collect overwritten", cols: 5, rows: 5,
		in: "\x1b[1mA\x1b[1;1H\x1b[0mB", check: g2Cell(0, 0, 'B', 0)},
	{name: "do not garbage collect old styles in use", cols: 5, rows: 5,
		in: "\x1b[1mA\x1b[0mB", check: g2Cell(1, 0, 'B', 0)},

	// DECALN
	{name: "DECALN", cols: 2, rows: 2,
		in: "A\r\nB\x1b#8", screen: scr("EE\nEE"), cur: at(0, 0)},
	{name: "decaln reset margins", cols: 3, rows: 3,
		in: "\x1b[?6h\x1b[2;3r\x1b#8\x1b[T", screen: scr("\nEEE\nEEE")},
	{name: "decaln preserves color", cols: 3, rows: 3,
		in:     g2Red + "\x1b[?6h\x1b[2;3r\x1b#8\x1b[T",
		screen: scr("\nEEE\nEEE"), check: g2BgRed(0, 0, 1)},
	{name: "DECALN resets graphemes with protected mode", cols: 3, rows: 3,
		in:     "\x1bV\x1b[?2027h" + g2Family + "\x1b#8",
		screen: scr("EEE\nEEE\nEEE"), cur: at(0, 0),
		gap: "protected cells (DECSCA/SPA, DECSED/DECSEL)"},

	// insertBlanks (ICH) and insert mode
	{name: "insertBlanks", cols: 5, rows: 2,
		in: "ABC\x1b[1;1H\x1b[2@", screen: scr("  ABC")},
	{name: "insertBlanks pushes off end", cols: 3, rows: 2,
		in: "ABC\x1b[1;1H\x1b[2@", screen: scr("  A")},
	{name: "insertBlanks more than size", cols: 3, rows: 2,
		in: "ABC\x1b[1;1H\x1b[5@", screen: scr("")},
	{name: "insertBlanks no scroll region, fits", cols: 10, rows: 10,
		in: "ABC\x1b[1;1H\x1b[2@", screen: scr("  ABC")},
	{name: "insertBlanks preserves background sgr", cols: 10, rows: 10,
		in:     "ABC\x1b[1;1H" + g2Red + "\x1b[2@",
		screen: scr("  ABC"), check: g2BgRed(0, 0, 1)},
	{name: "insertBlanks shift off screen", cols: 5, rows: 10,
		in: "  ABC\x1b[1;3H\x1b[2@X", screen: scr("  X A")},
	{name: "insertBlanks split multi-cell character", cols: 5, rows: 10,
		in: "123橋\x1b[1;1H\x1b[1@", screen: scr(" 123")},
	{name: "insertBlanks inside left/right scroll region", cols: 10, rows: 10,
		in:     "\x1b[?69h\x1b[3;5s\x1b[1;3HABC\x1b[1;3H\x1b[2@X",
		screen: scr("  X A"),
		gap:    "DECLRMM left/right margins"},
	{name: "insertBlanks outside left/right scroll region", cols: 6, rows: 10,
		// Upstream sets the margins directly; DECSLRM homes the cursor, so
		// put it back on the last column.
		in:     "\x1b[1;4HABC\x1b[?69h\x1b[3;5s\x1b[1;6H\x1b[2@X",
		screen: scr("   ABX"),
		gap:    "DECLRMM left/right margins"},
	{name: "insertBlanks left/right scroll region large count", cols: 10, rows: 10,
		in:     "\x1b[?6h\x1b[?69h\x1b[3;5s\x1b[1;1H\x1b[140@X",
		screen: scr("  X"),
		gap:    "DECLRMM left/right margins"},
	{name: "insertBlanks deleting graphemes", cols: 5, rows: 5,
		in:     "\x1b[?2027hABC" + g2Family + "\x1b[1;1H\x1b[4@",
		screen: scr("    A"),
		gap:    "grapheme clustering (mode 2027, VS15/VS16 width changes)"},
	{name: "insertBlanks shift graphemes", cols: 5, rows: 5,
		in:     "\x1b[?2027hA" + g2Family + "\x1b[1;1H\x1b[1@",
		screen: scr(" A" + g2Family),
		gap:    "grapheme clustering (mode 2027, VS15/VS16 width changes)"},
	{name: "insertBlanks split multi-cell character from tail", cols: 5, rows: 10,
		in: "橋123\x1b[1;2H\x1b[1@", screen: scr("   12")},
	{name: "insertBlanks shifts hyperlinks", cols: 10, rows: 2,
		in: g2Link + "ABC\x1b[1;1H\x1b[2@", screen: scr("  ABC")},
	{name: "insertBlanks pushes hyperlink off end completely", cols: 3, rows: 2,
		in: g2Link + "ABC\x1b[1;1H\x1b[3@", screen: scr("")},
	{name: "insertBlanks wide char straddling right margin", cols: 10, rows: 5,
		in:     "\x1b[1;1HABCD橋\x1b[?69h\x1b[1;5s\x1b[1;3H\x1b[1@",
		screen: scr("AB CD"),
		gap:    "DECLRMM left/right margins"},
	{name: "insertBlanks wide char spacer_tail orphaned beyond right margin", cols: 10, rows: 5,
		in:     "中中中中中\x1b[?69h\x1b[1;9sa\x1b[8@",
		screen: scr("a"),
		gap:    "DECLRMM left/right margins"},
	{name: "insert mode with space", cols: 10, rows: 2,
		in: "hello\x1b[1;2H\x1b[4hX", screen: scr("hXello")},
	{name: "insert mode doesn't wrap pushed characters", cols: 5, rows: 2,
		in: "hello\x1b[1;2H\x1b[4hX", screen: scr("hXell")},
}

// Translated from ghostty (MIT), src/terminal/Terminal.zig (tests from
// "insert mode does nothing at the end of the line" to the end of the file)
// and src/terminal/stream_terminal.zig (byte-stream tests) at commit 521cf2c.

func init() { registerSuite("ghostty", ghostty3Cases) }

// g3all runs several checks and returns the first failure.
func g3all(fs ...func(*Terminal) string) func(*Terminal) string {
	return func(t *Terminal) string {
		for _, f := range fs {
			if msg := f(t); msg != "" {
				return msg
			}
		}
		return ""
	}
}

// g3rune checks the rune stored at x, y (0 blank, wideTail right half).
func g3rune(x, y int, r rune) func(*Terminal) string {
	return func(t *Terminal) string {
		if got := t.baseRune(x, y); got != r {
			return fmt.Sprintf("cell %d,%d rune got %q (%d) want %q (%d)", x, y, got, got, r, r)
		}
		return ""
	}
}

// g3curRune checks the rune under the cursor.
func g3curRune(r rune) func(*Terminal) string {
	return func(t *Terminal) string {
		x, y := t.Cursor().X, t.Cursor().Y
		if got := t.baseRune(x, y); got != r {
			return fmt.Sprintf("cell under cursor %d,%d got %q want %q", x, y, got, r)
		}
		return ""
	}
}

// g3bg checks the background of cells x0..x1-1 on row y.
func g3bg(x0, x1, y int, c Color) func(*Terminal) string {
	return func(t *Terminal) string {
		for x := x0; x < x1; x++ {
			if got := t.cellAt(x, y).Bg; got != c {
				return fmt.Sprintf("cell %d,%d bg got %#x want %#x", x, y, got, c)
			}
		}
		return ""
	}
}

// g3cell checks fg, bg and attributes of one cell.
func g3cell(x, y int, fg, bg Color, a Attr) func(*Terminal) string {
	return func(t *Terminal) string {
		c := t.cellAt(x, y)
		if c.Fg != fg || c.Bg != bg || c.A != a {
			return fmt.Sprintf("cell %d,%d style got fg %#x bg %#x attr %#x want fg %#x bg %#x attr %#x",
				x, y, c.Fg, c.Bg, c.A, fg, bg, a)
		}
		return ""
	}
}

// g3is checks an arbitrary predicate.
func g3is(msg string, ok func(*Terminal) bool) func(*Terminal) string {
	return func(t *Terminal) string {
		if !ok(t) {
			return msg
		}
		return ""
	}
}

// g3uniqueStyles reproduces "resize with high unique style per cell":
// every cell gets its own direct background color.
func g3uniqueStyles(wrapping bool) string {
	var b strings.Builder
	if wrapping {
		for i := 0; i < 30*30; i++ {
			fmt.Fprintf(&b, "\x1b[48;2;%d;%d;0mx", i>>8, i&0xff)
		}
	} else {
		for y := 0; y < 30; y++ {
			for x := 0; x < 30; x++ {
				// setCursorPos(y, x) with 0 clamped to 1 like ghostty.
				fmt.Fprintf(&b, "\x1b[%d;%dH\x1b[48;2;%d;%d;0mx", max(y, 1), max(x, 1), x, y)
			}
		}
	}
	return b.String() + rz(60, 30)
}

// g3resizePendingWrap expands "resize pending wrap live and saved cursors".
func g3resizePendingWrap() []confCase {
	type rc struct {
		text    string
		cols    int
		pending bool
		want    string
	}
	cases := []rc{
		{"ABCD", 6, false, "ABCDX"},
		{"ABCD", 3, false, "ABC\nDX"},
		{"ABCD", 2, true, "AB\nCD\nX"},
		{"ABCD", 4, true, "ABCD\nX"},
		{"ABCDEFGH", 6, false, "ABCDEF\nGHX"},
		{"AB界", 6, false, "AB界X"},
		{"ABC", 6, false, "ABCX"},
	}
	var out []confCase
	for _, c := range cases {
		for _, restore := range []bool{false, true} {
			pre, post := "", ""
			if restore {
				pre, post = "\x1b7", "\x1b8"
			}
			base := fmt.Sprintf("resize pending wrap live and saved cursors (%s to %d cols, saved %v)", c.text, c.cols, restore)
			// Checkpoint 1: pending wrap after the resize.
			out = append(out, confCase{
				name: base, cols: 4, rows: 5,
				in:      c.text + pre + rz(c.cols, 6) + post,
				pending: &c.pending,
			})
			// Checkpoint 2: where the next character lands.
			out = append(out, confCase{
				name: base + "#2", cols: 4, rows: 5,
				in:     c.text + pre + rz(c.cols, 6) + post + "X",
				screen: scr(c.want),
			})
		}
	}
	return out
}

const (
	g3gapLR    = "DECLRMM left/right margins"
	g3gapProt  = "protected cells (DECSCA/SPA, DECSED/DECSEL)"
	g3gapCS    = "national/other charsets"
	g3gapPal   = "palette and dynamic colors"
	g3gapRep   = "device reports / replies"
	g3gapImg   = "images (sixel/kitty)"
	g3red      = "\x1b[48;2;255;0;0m"
	g3redFg7f  = "\x1b[38;2;255;0;127m"
	g3redBg7f  = "\x1b[48;2;255;0;127m"
	g3abcDefGh = "ABC\r\nDEF\r\nGHI"
)

var ghostty3Cases = append([]confCase{
	// ---- Terminal.zig: insert mode ----
	{name: "insert mode does nothing at the end of the line", cols: 5, rows: 2,
		in: "hello\x1b[4hX", screen: scr("hello\nX")},
	{name: "insert mode with wide characters", cols: 5, rows: 2,
		in: "hello\x1b[1;2H\x1b[4h😀", screen: scr("h😀el")},
	{name: "insert mode with wide characters at end", cols: 5, rows: 2,
		in: "well\x1b[4h😀", screen: scr("well\n😀")},
	{name: "insert mode pushing off wide character", cols: 5, rows: 2,
		in: "123😀\x1b[4h\x1b[1;1HX", screen: scr("X123")},

	// ---- deleteChars (DCH) ----
	{name: "deleteChars", cols: 5, rows: 5,
		in: "ABCDE\x1b[1;2H\x1b[2P", screen: scr("ADE")},
	// "deleteChars zero count" calls the API with 0, which has no byte
	// form (CSI 0 P means 1); not translated.
	{name: "deleteChars more than half", cols: 5, rows: 5,
		in: "ABCDE\x1b[1;2H\x1b[3P", screen: scr("AE")},
	{name: "deleteChars more than line width", cols: 5, rows: 5,
		in: "ABCDE\x1b[1;2H\x1b[10P", screen: scr("A")},
	{name: "deleteChars should shift left", cols: 5, rows: 5,
		in: "ABCDE\x1b[1;2H\x1b[1P", screen: scr("ACDE")},
	{name: "deleteChars resets pending wrap", cols: 5, rows: 5,
		in: "ABCDE", pending: yes()},
	{name: "deleteChars resets pending wrap#2", cols: 5, rows: 5,
		in: "ABCDE\x1b[1P", pending: no()},
	{name: "deleteChars resets pending wrap#3", cols: 5, rows: 5,
		in: "ABCDE\x1b[1PX", screen: scr("ABCDX")},
	{name: "deleteChars resets wrap", cols: 5, rows: 5,
		in: "ABCDE123\x1b[1;1H\x1b[1PX", screen: scr("XCDE\n123")},
	{name: "deleteChars simple operation", cols: 10, rows: 10,
		in: "ABC123\x1b[1;3H\x1b[2P", screen: scr("AB23")},
	{name: "deleteChars preserves background sgr", cols: 10, rows: 10,
		in: "ABC123\x1b[1;3H" + g3red + "\x1b[2P", screen: scr("AB23"),
		check: g3bg(8, 10, 0, rgbColor(0xff, 0, 0))},
	{name: "deleteChars outside scroll region", cols: 6, rows: 10,
		in: "ABC123\x1b[?69h\x1b[3;5s\x1b[2P", screen: scr("ABC123"), gap: g3gapLR},
	{name: "deleteChars inside scroll region", cols: 6, rows: 10,
		in: "ABC123\x1b[?69h\x1b[3;5s\x1b[1;4H\x1b[1P", screen: scr("ABC2 3"), gap: g3gapLR},
	{name: "deleteChars split wide character from spacer tail", cols: 6, rows: 10,
		in: "A橋123\x1b[1;3H\x1b[1P", screen: scr("A 123")},
	{name: "deleteChars split wide character from wide", cols: 6, rows: 10,
		in:    "橋123\x1b[1;1H\x1b[1P",
		check: g3all(g3rune(0, 0, 0), g3rune(1, 0, '1'))},
	{name: "deleteChars split wide character from end", cols: 6, rows: 10,
		in:    "A橋123\x1b[1;1H\x1b[1P",
		check: g3all(g3rune(0, 0, 0x6A4B), g3rune(1, 0, wideTail))},
	{name: "deleteChars with a spacer head at the end", cols: 5, rows: 10,
		in:    "0123橋123\x1b[1;1H\x1b[1P",
		check: g3rune(3, 0, 0)},
	{name: "deleteChars split wide character tail", cols: 5, rows: 5,
		in: "\x1b[1;4H橋\r\x1b[4P0", screen: scr("0")},
	{name: "deleteChars wide char boundary conditions", cols: 8, rows: 1,
		in: "😀a😀b😀", screen: scr("😀a😀b😀")},
	{name: "deleteChars wide char boundary conditions#2", cols: 8, rows: 1,
		in: "😀a😀b😀\x1b[1;2H\x1b[3P", screen: scr("  b😀")},
	{name: "deleteChars wide char wrap boundary conditions", cols: 8, rows: 3,
		in: ".......😀abcde😀......", screen: scr(".......\n😀abcde\n😀......")},
	{name: "deleteChars wide char wrap boundary conditions#2", cols: 8, rows: 3,
		in: ".......😀abcde😀......\x1b[2;2H\x1b[3P", screen: scr(".......\n cde\n😀......")},
	{name: "deleteChars wide char across right margin", cols: 8, rows: 3,
		in: "123456橋\x1b[?69h\x1b[2;7s\x1b[1;2H\x1b[1P", screen: scr("13456"), gap: g3gapLR},

	// ---- saveCursor / restoreCursor ----
	// Upstream also saves GR=G3; only the pen and origin mode are translated.
	{name: "saveCursor", cols: 3, rows: 3,
		in: "\x1b[1m\x1b[?6h\x1b7\x1b[0m\x1b[?6l\x1b8",
		check: g3all(
			g3is("pen not bold after restore", func(t *Terminal) bool { return t.cur.pen.A&AttrBold != 0 }),
			g3is("origin mode not restored", func(t *Terminal) bool { return t.cur.origin }))},
	{name: "saveCursor position", cols: 10, rows: 5,
		in: "\x1b[1;5HA\x1b7\x1b[1;1HB\x1b8X", screen: scr("B   AX")},
	{name: "saveCursor pending wrap state", cols: 5, rows: 5,
		in: "\x1b[1;5HA\x1b7\x1b[1;1HB\x1b8X", screen: scr("B   A\nX")},
	{name: "saveCursor origin mode", cols: 10, rows: 5,
		in: "\x1b[?6h\x1b7\x1b[?69h\x1b[3;5s\x1b[2;4r\x1b8X", screen: scr("X"), gap: g3gapLR},
	{name: "saveCursor resize", cols: 10, rows: 5,
		in: "\x1b[1;10H\x1b7" + rz(5, 5) + "\x1b8X", screen: scr("    X")},
	{name: "saveCursor protected pen", cols: 10, rows: 5,
		in: "\x1bV\x1b[1;10H\x1b7\x1bW\x1b8", gap: g3gapProt},
	{name: "setProtectedMode", cols: 3, rows: 3,
		in: "\x1bV\x1b[1\"q\x1b[0\"q", gap: g3gapProt},

	// ---- eraseLine (EL) ----
	{name: "eraseLine simple erase right", cols: 5, rows: 5,
		in: "ABCDE\x1b[1;3H\x1b[0K", screen: scr("AB")},
	{name: "eraseLine resets pending wrap", cols: 5, rows: 5,
		in: "ABCDE\x1b[0K", pending: no()},
	{name: "eraseLine resets pending wrap#2", cols: 5, rows: 5,
		in: "ABCDE\x1b[0KB", screen: scr("ABCDB")},
	{name: "eraseLine resets wrap", cols: 5, rows: 5,
		in: "ABCDE123\x1b[1;1H\x1b[0KX", screen: scr("X\n123")},
	{name: "eraseLine right preserves background sgr", cols: 5, rows: 5,
		in: "ABCDE\x1b[1;2H" + g3red + "\x1b[0K", screen: scr("A"),
		check: g3bg(1, 5, 0, rgbColor(0xff, 0, 0))},
	{name: "eraseLine right wide character", cols: 10, rows: 5,
		in: "AB橋DE\x1b[1;4H\x1b[0K", screen: scr("AB")},
	{name: "eraseLine right protected attributes respected with iso", cols: 5, rows: 5,
		in: "\x1bVABC\x1b[1;1H\x1b[0K", screen: scr("ABC"), gap: g3gapProt},
	{name: "eraseLine right protected attributes ignored with dec most recent", cols: 5, rows: 5,
		in: "\x1bVABC\x1b[1\"q\x1b[0\"q\x1b[1;2H\x1b[0K", screen: scr("A"), gap: g3gapProt},
	{name: "eraseLine right protected attributes ignored with dec set", cols: 5, rows: 5,
		in: "\x1b[1\"qABC\x1b[1;2H\x1b[0K", screen: scr("A"), gap: g3gapProt},
	{name: "eraseLine right protected requested", cols: 10, rows: 5,
		in: "12345678\x1b[1;6H\x1b[1\"qX\x1b[1;4H\x1b[?0K", screen: scr("123  X"), gap: g3gapProt},
	{name: "eraseLine simple erase left", cols: 5, rows: 5,
		in: "ABCDE\x1b[1;3H\x1b[1K", screen: scr("   DE")},
	{name: "eraseLine left resets wrap", cols: 5, rows: 5,
		in: "ABCDE\x1b[1K", pending: no()},
	{name: "eraseLine left resets wrap#2", cols: 5, rows: 5,
		in: "ABCDE\x1b[1KB", screen: scr("    B")},
	{name: "eraseLine left preserves background sgr", cols: 5, rows: 5,
		in: "ABCDE\x1b[1;2H" + g3red + "\x1b[1K", screen: scr("  CDE"),
		check: g3bg(0, 2, 0, rgbColor(0xff, 0, 0))},
	{name: "eraseLine left wide character", cols: 10, rows: 5,
		in: "AB橋DE\x1b[1;3H\x1b[1K", screen: scr("    DE")},
	{name: "eraseLine left protected attributes respected with iso", cols: 5, rows: 5,
		in: "\x1bVABC\x1b[1;1H\x1b[1K", screen: scr("ABC"), gap: g3gapProt},
	{name: "eraseLine left protected attributes ignored with dec most recent", cols: 5, rows: 5,
		in: "\x1bVABC\x1b[1\"q\x1b[0\"q\x1b[1;2H\x1b[1K", screen: scr("  C"), gap: g3gapProt},
	{name: "eraseLine left protected attributes ignored with dec set", cols: 5, rows: 5,
		in: "\x1b[1\"qABC\x1b[1;2H\x1b[1K", screen: scr("  C"), gap: g3gapProt},
	{name: "eraseLine left protected requested", cols: 10, rows: 5,
		in: "123456789\x1b[1;6H\x1b[1\"qX\x1b[1;8H\x1b[?1K", screen: scr("     X  9"), gap: g3gapProt},
	{name: "eraseLine complete preserves background sgr", cols: 5, rows: 5,
		in: "ABCDE\x1b[1;2H" + g3red + "\x1b[2K", screen: scr(""),
		check: g3bg(0, 5, 0, rgbColor(0xff, 0, 0))},
	{name: "eraseLine complete resets wrap", cols: 5, rows: 5,
		in: "ABCDE123\x1b[1;1H\x1b[2KX" + rz(10, 5), screen: scr("X\n123")},
	{name: "eraseLine complete clears kitty placeholder flag", cols: 5, rows: 5,
		in: "\U0010EEEE\x1b[2K", gap: g3gapImg},
	{name: "eraseLine complete protected attributes respected with iso", cols: 5, rows: 5,
		in: "\x1bVABC\x1b[1;1H\x1b[2K", screen: scr("ABC"), gap: g3gapProt},
	{name: "eraseLine complete protected attributes ignored with dec most recent", cols: 5, rows: 5,
		in: "\x1bVABC\x1b[1\"q\x1b[0\"q\x1b[1;2H\x1b[2K", screen: scr(""), gap: g3gapProt},
	{name: "eraseLine complete protected attributes ignored with dec set", cols: 5, rows: 5,
		in: "\x1b[1\"qABC\x1b[1;2H\x1b[2K", screen: scr(""), gap: g3gapProt},
	{name: "eraseLine complete protected requested", cols: 10, rows: 5,
		in: "123456789\x1b[1;6H\x1b[1\"qX\x1b[1;8H\x1b[?2K", screen: scr("     X"), gap: g3gapProt},

	// ---- tabs ----
	{name: "tabClear single", cols: 30, rows: 5,
		in: "\t\x1b[0g\x1b[1;1H\t", cur: at(16, 0)},
	{name: "tabClear all", cols: 30, rows: 5,
		in: "\x1b[3g\x1b[1;1H\t", cur: at(29, 0)},

	// ---- printRepeat (REP) ----
	{name: "printRepeat simple", cols: 5, rows: 5,
		in: "A\x1b[1b", screen: scr("AA")},
	{name: "printRepeat wrap", cols: 5, rows: 5,
		in: "    A\x1b[1b", screen: scr("    A\nA")},
	{name: "printRepeat no previous character", cols: 5, rows: 5,
		in: "\x1b[1b", screen: scr("")},

	// ---- printSlice ----
	{name: "printSlice simple ascii", cols: 10, rows: 3,
		in: "hello", cur: at(5, 0), screen: scr("hello")},
	// Upstream also enables mode 2027; it does not affect these cells.
	{name: "printSlice charset batched fill", cols: 5, rows: 2,
		in: "\x1b)0\x0e\x1b[41m\x1b[2J" +
			"\x1b[1;1H\x1b[0mlqqqkmqqqj" +
			"\x1b[1;1H\x1b[0mlqqqkmqqqj" +
			"\x1b[1;1H\x1b[1mlqqqkmqqqj" +
			"\x1b[1;1H\x1b[1mlqqqkmqqqj" +
			"\x1b[1;1H\x1b[0mlqqqkmqqqj",
		screen: scr("┌───┐\n└───┘"), pending: yes(),
		check: func(t *Terminal) string {
			for y := 0; y < 2; y++ {
				for x := 0; x < 5; x++ {
					if msg := g3cell(x, y, 0, 0, 0)(t); msg != "" {
						return msg
					}
				}
			}
			return ""
		}},
	{name: "printSlice charset batched fill (bold pass)", cols: 5, rows: 2,
		in: "\x1b)0\x0e\x1b[41m\x1b[2J" +
			"\x1b[1;1H\x1b[0mlqqqkmqqqj" +
			"\x1b[1;1H\x1b[1mlqqqkmqqqj",
		screen: scr("┌───┐\n└───┘"), pending: yes(),
		check: func(t *Terminal) string {
			for y := 0; y < 2; y++ {
				for x := 0; x < 5; x++ {
					if msg := g3cell(x, y, 0, 0, AttrBold)(t); msg != "" {
						return msg
					}
				}
			}
			return ""
		}},
	{name: "printSlice charset single shift and repeat", cols: 10, rows: 2,
		in: "\x1b(0\x1b*A\x1bN#q\x1b[2b\x1b(B\x1b[2b", screen: scr("£───qq"), gap: g3gapCS},
	{name: "printSlice wraps and scrolls", cols: 5, rows: 2,
		in: "abcdefghijkl", screen: scr("fghij\nkl"), cur: at(2, 1), pending: no()},
	{name: "printSlice pending wrap state", cols: 5, rows: 2,
		in: "abcde", cur: at(4, 0), pending: yes(), screen: scr("abcde")},

	// ---- printAttributes: the pen it reports, checked on a printed cell ----
	{name: "printAttributes", cols: 5, rows: 5,
		in: "\x1b[38;2;1;2;3mX", check: g3cell(0, 0, rgbColor(1, 2, 3), 0, 0)},
	{name: "printAttributes#2", cols: 5, rows: 5,
		in: "\x1b[1m\x1b[48;2;1;2;3mX", check: g3cell(0, 0, 0, rgbColor(1, 2, 3), AttrBold)},
	{name: "printAttributes#3", cols: 5, rows: 5,
		in: "\x1b[1m\x1b[2m\x1b[3m\x1b[4m\x1b[5m\x1b[7m\x1b[8m\x1b[9m\x1b[53m\x1b[38;2;100;200;255m\x1b[48;2;101;102;103mX",
		check: g3cell(0, 0, rgbColor(100, 200, 255), rgbColor(101, 102, 103),
			AttrBold|AttrFaint|AttrItalic|AttrBlink|AttrInverse|AttrHidden|AttrStrikethrough|AttrOverline|1<<ulShift)},
	{name: "printAttributes#4", cols: 5, rows: 5,
		in: "\x1b[4mX", check: g3cell(0, 0, 0, 0, 1<<ulShift)},
	{name: "printAttributes#5", cols: 5, rows: 5,
		in: "\x1b[4:2mX", check: g3cell(0, 0, 0, 0, 2<<ulShift)},
	{name: "printAttributes#6", cols: 5, rows: 5,
		in: "\x1b[4:3mX", check: g3cell(0, 0, 0, 0, 3<<ulShift)},
	{name: "printAttributes#7", cols: 5, rows: 5,
		in: "\x1b[4:4mX", check: g3cell(0, 0, 0, 0, 4<<ulShift)},
	{name: "printAttributes#8", cols: 5, rows: 5,
		in: "\x1b[4:5mX", check: g3cell(0, 0, 0, 0, 5<<ulShift)},
	{name: "printAttributes#9", cols: 5, rows: 5,
		in: "\x1b[1;4:5;38;2;1;2;3m\x1b[0mX", check: g3cell(0, 0, 0, 0, 0)},

	// ---- eraseDisplay (ED) ----
	{name: "eraseDisplay simple erase below", cols: 5, rows: 5,
		in: g3abcDefGh + "\x1b[2;2H\x1b[0J", screen: scr("ABC\nD")},
	{name: "eraseDisplay erase below preserves SGR bg", cols: 5, rows: 5,
		in: g3abcDefGh + "\x1b[2;2H" + g3red + "\x1b[0J", screen: scr("ABC\nD"),
		check: g3bg(1, 5, 1, rgbColor(0xff, 0, 0))},
	{name: "eraseDisplay below split multi-cell", cols: 5, rows: 5,
		in: "AB橋C\r\nDE橋F\r\nGH橋I\x1b[2;4H\x1b[0J", screen: scr("AB橋C\nDE")},
	{name: "eraseDisplay below protected attributes respected with iso", cols: 5, rows: 5,
		in: "\x1bV" + g3abcDefGh + "\x1b[2;2H\x1b[0J", screen: scr("ABC\nDEF\nGHI"), gap: g3gapProt},
	{name: "eraseDisplay below protected attributes ignored with dec most recent", cols: 5, rows: 5,
		in: "\x1bV" + g3abcDefGh + "\x1b[1\"q\x1b[0\"q\x1b[2;2H\x1b[0J", screen: scr("ABC\nD"), gap: g3gapProt},
	{name: "eraseDisplay below protected attributes ignored with dec set", cols: 5, rows: 5,
		in: "\x1b[1\"q" + g3abcDefGh + "\x1b[2;2H\x1b[0J", screen: scr("ABC\nD"), gap: g3gapProt},
	{name: "eraseDisplay below protected attributes respected with force", cols: 5, rows: 5,
		in: "\x1b[1\"q" + g3abcDefGh + "\x1b[2;2H\x1b[?0J", screen: scr("ABC\nDEF\nGHI"), gap: g3gapProt},
	{name: "eraseDisplay simple erase above", cols: 5, rows: 5,
		in: g3abcDefGh + "\x1b[2;2H\x1b[1J", screen: scr("\n  F\nGHI")},
	{name: "eraseDisplay erase above preserves SGR bg", cols: 5, rows: 5,
		in: g3abcDefGh + "\x1b[2;2H" + g3red + "\x1b[1J", screen: scr("\n  F\nGHI"),
		check: g3bg(0, 2, 1, rgbColor(0xff, 0, 0))},
	{name: "eraseDisplay above split multi-cell", cols: 5, rows: 5,
		in: "AB橋C\r\nDE橋F\r\nGH橋I\x1b[2;3H\x1b[1J", screen: scr("\n    F\nGH橋I")},
	{name: "eraseDisplay above protected attributes respected with iso", cols: 5, rows: 5,
		in: "\x1bV" + g3abcDefGh + "\x1b[2;2H\x1b[1J", screen: scr("ABC\nDEF\nGHI"), gap: g3gapProt},
	{name: "eraseDisplay above protected attributes ignored with dec most recent", cols: 5, rows: 5,
		in: "\x1bV" + g3abcDefGh + "\x1b[1\"q\x1b[0\"q\x1b[2;2H\x1b[1J", screen: scr("\n  F\nGHI"), gap: g3gapProt},
	{name: "eraseDisplay above protected attributes ignored with dec set", cols: 5, rows: 5,
		in: "\x1b[1\"q" + g3abcDefGh + "\x1b[2;2H\x1b[1J", screen: scr("\n  F\nGHI"), gap: g3gapProt},
	{name: "eraseDisplay above protected attributes respected with force", cols: 5, rows: 5,
		in: "\x1b[1\"q" + g3abcDefGh + "\x1b[2;2H\x1b[?1J", screen: scr("ABC\nDEF\nGHI"), gap: g3gapProt},
	{name: "eraseDisplay protected complete", cols: 10, rows: 5,
		in: "A\r\n123456789\x1b[2;6H\x1b[1\"qX\x1b[2;4H\x1b[?2J", screen: scr("\n     X"), gap: g3gapProt},
	{name: "eraseDisplay protected below", cols: 10, rows: 5,
		in: "A\r\n123456789\x1b[2;6H\x1b[1\"qX\x1b[2;4H\x1b[?0J", screen: scr("A\n123  X"), gap: g3gapProt},
	// .scroll_complete is CSI 22 J (kitty's "scroll screen into scrollback, then clear").
	{name: "eraseDisplay scroll complete", cols: 10, rows: 5,
		in: "A\r\n\x1b[22J", screen: scr("")},
	{name: "eraseDisplay protected above", cols: 10, rows: 3,
		in: "A\r\n123456789\x1b[2;6H\x1b[1\"qX\x1b[2;8H\x1b[?1J", screen: scr("\n     X  9"), gap: g3gapProt},
	// Upstream asserts the cursor style survives ED 2; checked by printing.
	{name: "eraseDisplay complete preserves cursor", cols: 5, rows: 5,
		in: "\x1b[1mAAAA\x1b[2JX", check: g3all(g3rune(4, 0, 'X'), g3cell(4, 0, 0, 0, AttrBold))},

	// ---- fullReset (RIS) ----
	{name: "fullReset with a non-empty pen", cols: 80, rows: 80,
		in: g3redFg7f + g3redBg7f + "\x1bc",
		check: g3all(g3cell(0, 0, 0, 0, 0),
			g3is("pen not default after RIS", func(t *Terminal) bool { return t.cur.pen == cell{} }))},
	{name: "fullReset with a non-empty saved cursor", cols: 80, rows: 80,
		in: g3redFg7f + g3redBg7f + "\x1b7\x1bc",
		check: g3all(g3cell(0, 0, 0, 0, 0),
			g3is("pen not default after RIS", func(t *Terminal) bool { return t.cur.pen == cell{} }))},
	{name: "fullReset origin mode", cols: 10, rows: 10,
		in: "\x1b[3;5H\x1b[?6h\x1bc", cur: at(0, 0),
		check: g3is("origin mode still set after RIS", func(t *Terminal) bool { return !t.cur.origin })},

	// ---- softReset (DECSTR) ----
	// Upstream sets keypad_keys (DECNKM); ESC = sets the same keypad state.
	// reverse_wrap (?45) is set but its reset is not checked here.
	{name: "softReset modes", cols: 10, rows: 10,
		in: "\x1b[4h\x1b[?6h\x1b[?7l\x1b[?45h\x1b[?25l\x1b[?1h\x1b=\x1b[?2004h\x1b[!p",
		check: g3all(
			g3is("insert still set", func(t *Terminal) bool { return !t.insert }),
			g3is("origin still set", func(t *Terminal) bool { return !t.cur.origin }),
			g3is("wraparound not set", func(t *Terminal) bool { return t.autowrap }),
			g3is("cursor not visible", func(t *Terminal) bool { return !t.cursorHidden }),
			g3is("cursor keys still set", func(t *Terminal) bool { return !t.appCursor }),
			g3is("keypad keys still set", func(t *Terminal) bool { return !t.appKeypad }),
			g3is("bracketed paste was reset", func(t *Terminal) bool { return t.bracketed }))},
	{name: "softReset margins", cols: 10, rows: 10,
		in:    "\x1b[3;4r\x1b[!p",
		check: g3is("scroll region not reset", func(t *Terminal) bool { return t.top == 0 && t.bot == 9 })},
	{name: "softReset margins#2", cols: 10, rows: 10,
		in: "\x1b[3;4r\x1b[?69h\x1b[5;6s\x1b[!p", gap: g3gapLR},
	{name: "softReset keeps the cursor and screen", cols: 10, rows: 10,
		in: "hello\x1b[6;5H\x1b[!p", cur: at(4, 5), screen: scr("hello")},
	{name: "softReset saved cursor", cols: 10, rows: 10,
		in: "\x1b[6;5H\x1b[1m\x1b7\x1b[!p\x1b8", cur: at(0, 0),
		check: g3is("pen still bold", func(t *Terminal) bool { return t.cur.pen.A&AttrBold == 0 })},
	{name: "softReset pen, protection, and charsets", cols: 10, rows: 10,
		in: "\x1b[1m\x1b[1\"q\x1b(0\x1b[!p",
		check: g3all(
			g3is("pen not default", func(t *Terminal) bool { return t.cur.pen == cell{} }),
			g3is("G0 still DEC special graphics", func(t *Terminal) bool { return t.cur.g[0] == charsetASCII }))},
	{name: "softReset ends ISO protection for erase", cols: 10, rows: 10,
		in: "\x1bVA\x1bW\x1b[!p\x1b[2J", screen: scr(""), gap: g3gapProt},
	{name: "softReset palette", cols: 10, rows: 10,
		in: "\x1b]4;1;rgb:ff/00/00\x1b\\\x1b]4;200;rgb:ff/00/00\x1b\\\x1b[!p", gap: g3gapPal},

	// ---- resize ----
	{name: "resize less cols with wide char then print", cols: 3, rows: 3,
		in: "x😀" + rz(2, 3) + "\x1b[1;2H😀"},
	{name: "resize less cols without reflow cutting wide char tail", cols: 3, rows: 1,
		in: "a一\x1b[?7l" + rz(2, 1), check: g3rune(1, 0, 0)},
	{name: "resize with left and right margin set", cols: 70, rows: 23,
		in: "\x1b[?69h0\x1b[?40h" + rz(70, 23) + "\x1b[2;0s\x1b[1850b" + rz(70, 23), gap: g3gapLR},
	{name: "resize with wraparound off", cols: 4, rows: 2,
		in: "\x1b[?7l0123" + rz(2, 2), screen: scr("01")},
	{name: "resize with wraparound on", cols: 4, rows: 2,
		in: "\x1b[?7h0123" + rz(2, 2), screen: scr("01\n23")},
	{name: "resize with high unique style per cell", cols: 30, rows: 30,
		in: g3uniqueStyles(false)},
	{name: "resize with high unique style per cell with wrapping", cols: 30, rows: 30,
		in: g3uniqueStyles(true)},
	{name: "resize with reflow and saved cursor", cols: 2, rows: 3,
		in: "1A2B\x1b[2;2H", screen: scr("1A\n2B"), check: g3curRune('B')},
	{name: "resize with reflow and saved cursor#2", cols: 2, rows: 3,
		in: "1A2B\x1b[2;2H\x1b7" + rz(5, 3) + "\x1b8", screen: scr("1A2B"), check: g3curRune('B')},
	{name: "resize with reflow and saved cursor pending wrap", cols: 2, rows: 3,
		in: "1A2B", screen: scr("1A\n2B"), check: g3curRune('B')},
	{name: "resize with reflow and saved cursor pending wrap#2", cols: 2, rows: 3,
		in: "1A2B\x1b7" + rz(5, 3) + "\x1b8", screen: scr("1A2B")},
	{name: "resize with reflow and saved cursor pending wrap#3", cols: 2, rows: 3,
		in: "1A2B\x1b7" + rz(5, 3) + "\x1b8X", screen: scr("1A2BX")},
	{name: "saved cursor survives repeated widening", cols: 4, rows: 5,
		in: "abc\r\nAAA|\x1b7" + rz(5, 5) + rz(6, 5) + "\x1b8X", screen: scr("abc\nAAA|X")},

	// ---- DECCOLM ----
	{name: "DECCOLM without DEC mode 40", cols: 5, rows: 5, in: "\x1b[?3h", screen: scr(""),
		check: g3is("disabled DECCOLM changed width or modes", func(t *Terminal) bool { return t.cols == 5 && !t.Modes().Columns132 && !t.Modes().AllowColumnSwitching })},
	{name: "DECCOLM unset", cols: 5, rows: 5, in: "\x1b[?40h\x1b[?3l", screen: scr(""), cur: at(0, 0),
		check: g3is("DECCOLM did not select80 columns", func(t *Terminal) bool { return t.cols == 80 && !t.Modes().Columns132 && t.Modes().AllowColumnSwitching })},
	{name: "DECCOLM resets pending wrap", cols: 5, rows: 5, in: "ABCDE\x1b[?40h\x1b[?3l", screen: scr(""), cur: at(0, 0), pending: no(),
		check: g3is("DECCOLM did not resize", func(t *Terminal) bool { return t.cols == 80 })},
	{name: "DECCOLM preserves SGR bg", cols: 5, rows: 5, in: g3red + "\x1b[?40h\x1b[?3l", screen: scr(""),
		check: g3all(g3bg(0, 80, 0, rgbColor(255, 0, 0)), g3is("DECCOLM lost background pen", func(t *Terminal) bool { return t.cols == 80 && t.cur.pen.Bg == rgbColor(255, 0, 0) }))},
	{name: "DECCOLM resets scroll region", cols: 5, rows: 5,
		in: "\x1b[?69h\x1b[2;3r\x1b[3;5s\x1b[?40h\x1b[?3l", cur: at(0, 0),
		check: g3is("DECCOLM did not reset margins", func(t *Terminal) bool {
			return t.cols == 80 && t.top == 0 && t.bot == 4 && t.left == 0 && t.right == 79
		})},

	// ---- alternate screen modes ----
	{name: "mode 47 alt screen plain", cols: 5, rows: 5,
		in: "1A\x1b[?47h", alt: yes(), screen: scr("")},
	{name: "mode 47 alt screen plain#2", cols: 5, rows: 5,
		in: "1A\x1b[?47h2B", alt: yes(), screen: scr("  2B")},
	{name: "mode 47 alt screen plain#3", cols: 5, rows: 5,
		in: "1A\x1b[?47h2B\x1b[?47l", alt: no(), screen: scr("1A")},
	{name: "mode 47 alt screen plain#4", cols: 5, rows: 5,
		in: "1A\x1b[?47h2B\x1b[?47l\x1b[?47h", alt: yes(), screen: scr("  2B")},
	{name: "mode 47 copies cursor both directions", cols: 5, rows: 5,
		in: g3redFg7f + "\x1b[?47h", alt: yes(),
		check: g3is("pen lost its style entering alt", func(t *Terminal) bool { return t.cur.pen.Fg != 0 })},
	{name: "mode 47 copies cursor both directions#2", cols: 5, rows: 5,
		in: g3redFg7f + "\x1b[?47h\x1b[38;2;0;255;0m\x1b[?47l", alt: no(),
		check: g3is("pen lost its style leaving alt", func(t *Terminal) bool { return t.cur.pen.Fg != 0 })},
	{name: "mode 1047 alt screen plain", cols: 5, rows: 5,
		in: "1A\x1b[?1047h", alt: yes(), screen: scr("")},
	{name: "mode 1047 alt screen plain#2", cols: 5, rows: 5,
		in: "1A\x1b[?1047h2B", alt: yes(), screen: scr("  2B")},
	{name: "mode 1047 alt screen plain#3", cols: 5, rows: 5,
		in: "1A\x1b[?1047h2B\x1b[?1047l", alt: no(), screen: scr("1A")},
	{name: "mode 1047 alt screen plain#4", cols: 5, rows: 5,
		in: "1A\x1b[?1047h2B\x1b[?1047l\x1b[?1047h", alt: yes(), screen: scr("")},
	{name: "mode 1047 copies cursor both directions", cols: 5, rows: 5,
		in: g3redFg7f + "\x1b[?1047h", alt: yes(),
		check: g3is("pen lost its style entering alt", func(t *Terminal) bool { return t.cur.pen.Fg != 0 })},
	{name: "mode 1047 copies cursor both directions#2", cols: 5, rows: 5,
		in: g3redFg7f + "\x1b[?1047h\x1b[38;2;0;255;0m\x1b[?1047l", alt: no(),
		check: g3is("pen lost its style leaving alt", func(t *Terminal) bool { return t.cur.pen.Fg != 0 })},
	{name: "mode 1049 alt screen plain", cols: 5, rows: 5,
		in: "1A\x1b[?1049h", alt: yes(), screen: scr("")},
	{name: "mode 1049 alt screen plain#2", cols: 5, rows: 5,
		in: "1A\x1b[?1049h2B", alt: yes(), screen: scr("  2B")},
	{name: "mode 1049 alt screen plain#3", cols: 5, rows: 5,
		in: "1A\x1b[?1049h2B\x1b[?1049l", alt: no(), screen: scr("1A")},
	{name: "mode 1049 alt screen plain#4", cols: 5, rows: 5,
		in: "1A\x1b[?1049h2B\x1b[?1049lC", alt: no(), screen: scr("1AC")},
	{name: "mode 1049 alt screen plain#5", cols: 5, rows: 5,
		in: "1A\x1b[?1049h2B\x1b[?1049lC\x1b[?1049h", alt: yes(), screen: scr("")},

	{name: "deleteLines wide char at right margin with full clear", cols: 80, rows: 24,
		in: "\x1b[10;39H中\x1b[?69h\x1b[5;39s\x1b[24S", gap: g3gapLR},
	// Upstream marks row 1 as a prompt first (not expressible as bytes
	// beyond OSC 133); the observable fact is that ED 2 with no prompt on
	// screen pushes nothing into the scrollback.
	{name: "eraseDisplay complete ignores stale prompt on recycled row", cols: 10, rows: 3,
		in: "hello\x1b[2;3r\x1b[3;1H\n\x1b[r\x1b[2J", scrollback: []string{}},
	{name: "eraseDisplay complete at prompt not on last row scrolls", cols: 10, rows: 15,
		in:  strings.Repeat("x\r\n", 10) + "\x1b]133;A\x07$ \x1b]133;B\x07\x1b[1;1H\x1b[2J",
		cur: at(0, 0), pending: no(), screen: scr(""),
		scrollback: []string{"x", "x", "x", "x", "x", "x", "x", "x", "x", "x", "$"}, check: g1Prompts()},

	// ---- stream_terminal.zig byte-stream tests ----
	{name: "stream: basic print", cols: 10, rows: 10,
		in: "Hello", cur: at(5, 0), screen: scr("Hello")},
	{name: "stream: semantic failure is sticky while processing continues", cols: 10, rows: 2,
		in: "\x1b]2;unavailable\x1b\\ignored\x1bcOK", screen: scr("OK")},
	{name: "stream: cursor movement", cols: 10, rows: 10,
		in: "Hello\x1b[1;1H", cur: at(0, 0)},
	{name: "stream: cursor movement#2", cols: 10, rows: 10,
		in: "Hello\x1b[1;1H\x1b[2;3H", cur: at(2, 1)},
	{name: "stream: erase operations", cols: 20, rows: 10,
		in: "Hello World", cur: at(11, 0)},
	{name: "stream: erase operations#2", cols: 20, rows: 10,
		in: "Hello World\x1b[1;6H\x1b[K", screen: scr("Hello")},
	{name: "stream: tabs", cols: 80, rows: 10,
		in: "A\tB", cur: at(9, 0), screen: scr("A       B")},
	{name: "stream: modes", cols: 80, rows: 24,
		in: "\x1b[?7l", check: g3is("wraparound still set", func(t *Terminal) bool { return !t.autowrap })},
	{name: "stream: modes#2", cols: 80, rows: 24,
		in: "\x1b[?7l\x1b[?7h", check: g3is("wraparound not set", func(t *Terminal) bool { return t.autowrap })},
	{name: "stream: scrolling regions", cols: 80, rows: 24,
		in:    "\x1b[5;20r",
		check: g3is("scroll region not 4..19", func(t *Terminal) bool { return t.top == 4 && t.bot == 19 })},
	{name: "stream: charsets", cols: 80, rows: 24,
		in: "\x1b(0`", screen: scr("◆")},
	{name: "stream: alt screen", cols: 10, rows: 5,
		in: "Primary\x1b[?1049h", alt: yes()},
	{name: "stream: alt screen#2", cols: 10, rows: 5,
		in: "Primary\x1b[?1049hAlt\x1b[?1049l", alt: no(), screen: scr("Primary")},
	{name: "stream: cursor save and restore", cols: 80, rows: 24,
		in: "\x1b[10;15H", cur: at(14, 9)},
	{name: "stream: cursor save and restore#2", cols: 80, rows: 24,
		in: "\x1b[10;15H\x1b7\x1b[1;1H", cur: at(0, 0)},
	{name: "stream: cursor save and restore#3", cols: 80, rows: 24,
		in: "\x1b[10;15H\x1b7\x1b[1;1H\x1b8", cur: at(14, 9)},
	{name: "stream: attributes", cols: 80, rows: 24,
		in: "\x1b[1mBold\x1b[0m", screen: scr("Bold")},
	{name: "stream: DECRQSS responses", cols: 80, rows: 24,
		in: "\x1b[1m\x1bP$qm\x1b\\\x1b[0;53m\x1bP$qm\x1b\\", gap: g3gapRep},
	{name: "stream: DECRQSS without write effect is ignored", cols: 80, rows: 24,
		in: "\x1bP$qm\x1b\\", gap: g3gapRep},
	{name: "stream: XTGETTCAP responses", cols: 80, rows: 24,
		in: "\x1bP+q536D756C78\x1b\\", gap: g3gapRep},
	{name: "stream: XTGETTCAP without write effect is ignored", cols: 80, rows: 24,
		in: "\x1bP+q536D756C78\x1b\\", gap: g3gapRep},
	{name: "stream: XTGETTCAP TN responses", cols: 80, rows: 24,
		in: "\x1bP+q544E\x1b\\", gap: g3gapRep},
	{name: "stream: DECALN screen alignment", cols: 10, rows: 3,
		in: "\x1b#8", screen: scr("EEEEEEEEEE\nEEEEEEEEEE\nEEEEEEEEEE"), cur: at(0, 0)},
	{name: "stream: full reset", cols: 80, rows: 24,
		in:  "Hello\x1b[10;20H\x1b[5;20r\x1b[?7l\x1b_25a1;r;cp=e0a0;AAAAAAAAAAAAAA==\x1b\\\x1bc",
		cur: at(0, 0),
		check: g3all(
			g3is("scroll region not reset", func(t *Terminal) bool { return t.top == 0 && t.bot == 23 }),
			g3is("wraparound not set", func(t *Terminal) bool { return t.autowrap }))},
	{name: "stream: ignores query actions", cols: 80, rows: 24,
		in: "\x1b[c\x1b[5n\x1b[6n\x1b]4;0;?\x1b\\\x1b]21;foreground=?\x1b\\\x1b]52;c;%%%invalid-base64%%%\x1b\\" +
			"\x1b_Ga=p,i=999\x1b\\\x1b_25a1;r;cp=41;%%%invalid%%%\x1b\\Test",
		screen: scr("Test")},

	// Palette / dynamic colors (OSC 4/104/10/11/12/110/111/112, kitty OSC 21).
	{name: "stream: OSC 4 set and reset palette", cols: 10, rows: 10,
		in: "\x1b]4;0;rgb:ff/00/00\x1b\\\x1b]104;0\x1b\\", gap: g3gapPal},
	{name: "stream: OSC 104 reset all palette colors", cols: 10, rows: 10,
		in: "\x1b]4;0;rgb:ff/00/00\x1b\\\x1b]4;1;rgb:00/ff/00\x1b\\\x1b]4;2;rgb:00/00/ff\x1b\\\x1b]104\x1b\\", gap: g3gapPal},
	{name: "stream: OSC 10 set and reset foreground color", cols: 10, rows: 10,
		in: "\x1b]10;rgb:ff/00/00\x1b\\\x1b]110\x1b\\", gap: g3gapPal},
	{name: "stream: OSC 11 set and reset background color", cols: 10, rows: 10,
		in: "\x1b]11;rgb:00/ff/00\x1b\\\x1b]111\x1b\\", gap: g3gapPal},
	{name: "stream: OSC 12 set and reset cursor color", cols: 10, rows: 10,
		in: "\x1b]12;rgb:00/00/ff\x1b\\\x1b]112\x1b\\", gap: g3gapPal},
	{name: "stream: OSC color query responses", cols: 10, rows: 10,
		in: "\x1b]4;0;?\x1b\\\x1b]10;?\x1b\\\x1b]11;?\x1b\\", gap: g3gapPal},
	{name: "stream: kitty color protocol set palette", cols: 10, rows: 10,
		in: "\x1b]21;1=rgb:00/ff/00\x1b\\", gap: g3gapPal},
	{name: "stream: kitty color protocol reset palette", cols: 10, rows: 10,
		in: "\x1b]21;1=rgb:00/ff/00\x1b\\\x1b]21;1=\x1b\\", gap: g3gapPal},
	{name: "stream: kitty color protocol set foreground", cols: 10, rows: 10,
		in: "\x1b]21;foreground=rgb:ff/00/00\x1b\\", gap: g3gapPal},
	{name: "stream: kitty color protocol set background", cols: 10, rows: 10,
		in: "\x1b]21;background=rgb:00/ff/00\x1b\\", gap: g3gapPal},
	{name: "stream: kitty color protocol set cursor", cols: 10, rows: 10,
		in: "\x1b]21;cursor=rgb:00/00/ff\x1b\\", gap: g3gapPal},
	{name: "stream: kitty color protocol reset foreground", cols: 10, rows: 10,
		in: "\x1b]21;foreground=rgb:ff/00/00\x1b\\\x1b]21;foreground=\x1b\\", gap: g3gapPal},
	{name: "stream: kitty color protocol query responses", cols: 10, rows: 10,
		in: "\x1b]21;foreground=?;background=?\x1b\\", gap: g3gapPal},
	{name: "stream: palette dirty flag set on color change", cols: 10, rows: 10,
		in: "\x1b]4;0;rgb:ff/00/00\x1b\\\x1b]104;0\x1b\\\x1b]21;1=rgb:00/ff/00\x1b\\", gap: g3gapPal},

	// Current native stream_terminal.zig at 46e9dc9a verifies these same
	// cursor positions; public row annotations distinguish L from A/P/N.
	{name: "stream: semantic prompt fresh line", cols: 10, rows: 10,
		in: "Hello\x1b]133;L\x07", cur: at(0, 1), pending: no(), screen: scr("Hello"),
		scrollback: []string{}, check: g1Prompts()},
	{name: "stream: semantic prompt fresh line new prompt", cols: 10, rows: 10,
		in: "Hello\x1b]133;A\x07", cur: at(0, 1), pending: no(), screen: scr("Hello"),
		scrollback: []string{}, check: g1Prompts(PromptNone, PromptPrimary)},
	{name: "stream: semantic prompt prompt_start", cols: 10, rows: 10,
		in: "Hello\x1b]133;P\x07", cur: at(5, 0), pending: no(), screen: scr("Hello"),
		scrollback: []string{}, check: g1Prompts(PromptPrimary)},
	{name: "stream: semantic prompt new_command", cols: 10, rows: 10,
		in: "Hello\x1b]133;N\x07", cur: at(0, 1), pending: no(), screen: scr("Hello"),
		scrollback: []string{}, check: g1Prompts(PromptNone, PromptPrimary)},
	{name: "stream: semantic prompt new_command at column zero", cols: 10, rows: 10,
		in: "\x1b]133;N\x07", cur: at(0, 0), pending: no(), screen: scr(""),
		scrollback: []string{}, check: g1Prompts(PromptPrimary)},

	// Effects with no callback leave the terminal usable.
	{name: "stream: bell effect callback", cols: 80, rows: 24,
		in: "\x07AfterBell", screen: scr("AfterBell")},
	{name: "stream: desktop_notification effect callback", cols: 80, rows: 24,
		in: "\x1b]9;Ignored\x1b\\AfterNotification", screen: scr("AfterNotification")},
	{name: "stream: clipboard_write effect callback", cols: 80, rows: 24,
		in: "\x1b]52;c;aGVsbG8=\x1b\\AfterClipboard", screen: scr("AfterClipboard")},
	{name: "stream: kitty clipboard malformed packets are silently dropped", cols: 80, rows: 24,
		in: "\x1b]5522;loc=primary\x1b\\\x1b]5522;type=bobr\x1b\\\x1b]5522;type=read:bare\x1b\\" +
			"\x1b]5522;type=wdata:mime=!!!;R2hvc3Q=\x1b\\\x1b]5522;type=wdata:mime=dGV4dC9wbGFpbg==;R2hvc3Q=\x1b\\" +
			"\x1b]5522;type=walias:mime=dGV4dC9wbGFpbg==;VEVYVA==\x1b\\ok",
		screen: scr("ok")},

	// DECRQM ANSI: the mode state part is checkable; the replies are not.
	{name: "stream: request mode DECRQM ANSI responses", cols: 80, rows: 24,
		in: "\x1b[4h", check: g3is("IRM not set", func(t *Terminal) bool { return t.insert })},
	{name: "stream: request mode DECRQM ANSI responses#2", cols: 80, rows: 24,
		in: "\x1b[4h\x1b[4l", check: g3is("IRM still set", func(t *Terminal) bool { return !t.insert })},
	{name: "stream: request mode DECRQM ANSI responses#3", cols: 80, rows: 24,
		in: "\x1b[20h", check: g3is("LNM not set", func(t *Terminal) bool { return t.newline })},
	{name: "stream: request mode DECRQM ANSI responses#4", cols: 80, rows: 24,
		in: "\x1b[20h\x1b[20l", check: g3is("LNM still set", func(t *Terminal) bool { return !t.newline })},
	{name: "stream: request mode DECRQM ANSI responses#5", cols: 80, rows: 24,
		in:    "\x1b[4h\x1b[?4l",
		check: g3is("DEC ?4 reset cleared ANSI IRM", func(t *Terminal) bool { return t.insert })},
	{name: "stream: request mode DECRQM ANSI responses#6", cols: 80, rows: 24,
		in: "\x1b[4h\x1b[4$p\x1b[?4$p\x1b[9999$p\x1b[1$p\x1b[19$p", gap: g3gapRep},

	{name: "stream: CSI W with intermediate but no params", cols: 80, rows: 24,
		in: "\x1b[?W"},

	// Window title (OSC 2).
	{name: "stream: window_title effect is called", cols: 80, rows: 24,
		in:    "\x1b]2;Hello World\x1b\\",
		check: func(t *Terminal) string { return g3title(t, "Hello World") }},
	{name: "stream: window_title effect not called without callback", cols: 80, rows: 24,
		in: "\x1b]2;Hello World\x1b\\Test", screen: scr("Test"),
		check: func(t *Terminal) string { return g3title(t, "Hello World") }},
	{name: "stream: window_title effect with empty title", cols: 80, rows: 24,
		in:    "\x1b]2;\x1b\\",
		check: func(t *Terminal) string { return g3title(t, "") }},
	{name: "stream: window_title not changed by cancelled OSC", cols: 80, rows: 24,
		in:    "\x1b]2;before\x07\x1b]2;can\x18",
		check: func(t *Terminal) string { return g3title(t, "before") }},
	{name: "stream: window_title not changed by cancelled OSC#2", cols: 80, rows: 24,
		in:    "\x1b]2;before\x07\x1b]2;can\x18\x1b]2;sub\x1a",
		check: func(t *Terminal) string { return g3title(t, "before") }},
	{name: "stream: window_title not changed by cancelled OSC#3", cols: 80, rows: 24,
		in:    "\x1b]2;before\x07\x1b]2;can\x18\x1b]2;sub\x1a\x1b]2;after\x1b\\",
		check: func(t *Terminal) string { return g3title(t, "after") }},

	{name: "stream: device status: readonly ignores all", cols: 80, rows: 24,
		in: "\x1b[5n\x1b[6n\x1b[?996n\x1b[?998nTest", screen: scr("Test")},
	{name: "stream: device attributes: readonly ignores", cols: 80, rows: 24,
		in: "\x1b[c\x1b[>c\x1b[=cTest", screen: scr("Test")},
	{name: "stream: kitty dnd: state updates work without write_pty effect", cols: 80, rows: 24,
		in: "\x1b]72;t=q\x1b\\\x1b]72;t=a\x1b\\ok", screen: scr("ok")},
}, g3resizePendingWrap()...)

func g3title(t *Terminal, want string) string {
	if got := t.Title(); got != want {
		return fmt.Sprintf("title got %q want %q", got, want)
	}
	return ""
}
