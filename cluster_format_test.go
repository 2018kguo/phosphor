package phosphor

import (
	"bytes"
	"math/rand/v2"
	"strings"
	"testing"
	"unicode/utf8"
)

// Keep the original codepoint loop independent of the bulk eligibility checks.
func appendClusterReference(t *Terminal, b []byte, c cluster, head, tail cell) []byte {
	if c.joinModes == (clusterModes{}) && int(c.width) == CodepointWidth(firstRune(c.text)) {
		return append(b, c.text...)
	}
	mode := false
	style := head
	width := 0
	var prev rune
	for i, index := 0, 0; i < len(c.text); index++ {
		cp, n := utf8.DecodeRuneInString(c.text[i:])
		next := c.joinModes.has(index) || i == 0 && CodepointWidth(cp) > t.cols
		if next != mode {
			b = appendMode(b, 2027, next)
			mode = next
		}
		if i == 0 {
			width = CodepointWidth(cp)
		} else if next {
			newWidth, _ := graphemeWidthEffect(width, prev, cp)
			if newWidth == 2 && width == 1 && tail.R == wideTail {
				b = t.appendStyle(b, style, tail)
				style = tail
			}
			width = newWidth
		}
		b = append(b, c.text[i:i+n]...)
		prev = cp
		i += n
	}
	if int(c.width) != width && utf8.RuneCountInString(c.text) == maxClusterRunes {
		if !mode {
			b = appendMode(b, 2027, true)
			mode = true
		}
		if c.width == 1 && emojiVariationBase(prev) {
			b = utf8.AppendRune(b, 0xfe0e)
		} else if c.width == 2 {
			if tail.R == wideTail {
				b = t.appendStyle(b, style, tail)
				style = tail
			}
			if emojiVariationBase(prev) {
				b = utf8.AppendRune(b, 0xfe0f)
			} else if prev == 0x200d {
				b = utf8.AppendRune(b, 0x2764)
			}
		}
	}
	if mode {
		b = appendMode(b, 2027, false)
	}
	return t.appendStyle(b, style, head)
}

func TestClusterFormattingMatchesCodepointLoop(t *testing.T) {
	texts := []string{
		"e\u0301", "❤\ufe0e", "❤\ufe0f", "❤\ufe0f\ufe0e", "❤\ufe0e\ufe0f",
		"👩\u200d💻", "🇺🇸", "👨\u200d👩\u200d👧\u200d👦", "\u0600A", "क्ष", "界\u0301",
		"A" + strings.Repeat("\u0301", 62),
		"A" + strings.Repeat("\u0301", 63),
		"A" + strings.Repeat("\u0301", 64),
		"☔" + strings.Repeat("\u0301", 62) + "\u200d☔",
		"❤" + strings.Repeat("\u0301", 63) + "\u200d",
	}
	for _, cols := range []int{1, 2, 8} {
		term := New(cols, 2)
		link := term.internExtra(cellExtra{link: ";https://example.test/cluster"})
		other := term.internExtra(cellExtra{link: ";https://example.test/tail", underline: IndexedColor(4)})
		styles := []cell{{}, {Fg: IndexedColor(1)}, {Bg: RGBColor(3, 4, 5), A: AttrProtected}, {A: AttrBold, E: link}, {E: other}}
		for _, text := range texts {
			count := utf8.RuneCountInString(text)
			var joined, mixed clusterModes
			for i := 1; i < count; i++ {
				joined.set(i)
				if i%2 == 0 {
					mixed.set(i)
				}
			}
			for _, modes := range []clusterModes{{}, joined, mixed} {
				for _, width := range []uint8{1, 2} {
					c := cluster{text: text, width: width, joinModes: modes}
					for _, head := range styles {
						for tailAt := -1; tailAt < len(styles); tailAt++ {
							var tail cell
							if tailAt >= 0 {
								tail = styles[tailAt]
								tail.R = wideTail
							}
							got := term.appendCluster([]byte("prefix"), c, head, tail)
							want := appendClusterReference(term, []byte("prefix"), c, head, tail)
							if !bytes.Equal(got, want) {
								t.Fatalf("cols=%d text=%q width=%d modes=%v head=%+v tail=%+v:\n%q\n%q", cols, text, width, modes, head, tail, got, want)
							}
						}
					}
				}
			}
		}
	}
}

func TestClusterFormattingRandomModeWords(t *testing.T) {
	rng := rand.New(rand.NewPCG(84, 39))
	alphabet := []rune{'A', '界', '❤', '☔', '👩', '💻', 0x301, 0x200d, 0xfe0e, 0xfe0f, 0x1f1fa, 0x1f3fb}
	for trial := range 4000 {
		term := New(1+rng.IntN(4), 2)
		count := 1 + rng.IntN(maxClusterRunes)
		text := make([]rune, count)
		var modes clusterModes
		for i := range text {
			text[i] = alphabet[rng.IntN(len(alphabet))]
			if i > 0 && (trial%2 == 0 || rng.IntN(2) != 0) {
				modes.set(i)
			}
		}
		c := cluster{text: string(text), width: uint8(1 + rng.IntN(2)), joinModes: modes}
		head := cell{Fg: IndexedColor(uint8(rng.IntN(16))), A: Attr(rng.IntN(256))}
		tail := head
		tail.R = wideTail
		if trial%3 == 0 {
			tail.Bg = IndexedColor(4)
		}
		if trial%5 == 0 {
			tail = cell{}
		}
		got, want := term.appendCluster(nil, c, head, tail), appendClusterReference(term, nil, c, head, tail)
		if !bytes.Equal(got, want) {
			t.Fatalf("trial %d text=%q modes=%v width=%d: %q / %q", trial, c.text, modes, c.width, got, want)
		}
	}
}
