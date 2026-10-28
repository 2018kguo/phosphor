package phosphor

//go:generate go run ./internal/unicodegen

import (
	"unicode/utf8"

	"github.com/clipperhouse/uax29/v2/graphemes"
)

// A cluster stores its physical width independently of its first scalar:
// selectors can narrow a wide emoji or widen a text character. joinModes
// records which appended codepoints were clustered under mode 2027, so
// history can replay mixed-mode cells without joining neighboring cells.
type cluster struct {
	text      string
	joinModes clusterModes
	width     uint8
}

// The second word holds the final codepoint of a full native-sized cluster.
// This metadata is allocated only for cells containing multiple codepoints.
type clusterModes [2]uint64

func (m clusterModes) has(i int) bool { return m[i/64]&(1<<uint(i%64)) != 0 }
func (m *clusterModes) set(i int)     { m[i/64] |= 1 << uint(i%64) }

// CodepointWidth returns the Unicode 17 terminal width of a single scalar:
// zero for combining or ignored scalars, two for wide scalars, otherwise one.
// Invalid scalars have width one. A complete grapheme can have a different
// width under mode 2027, so summing scalar widths measures mode-off layout.
func CodepointWidth(r rune) int {
	u := uint32(r)
	if u > utf8.MaxRune {
		return 1
	}
	word := scalarWidthPages[scalarWidthPageIDs[u>>8]][u>>5&7]
	return int(word >> ((u & 31) * 2) & 3)
}

func firstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

// extendGrapheme attaches r to the previous cell only when Unicode's
// grapheme boundary rules say it continues that cell's last cluster.
// Re-segmenting the bounded cell text also handles enabling mode 2027
// after older cells collected zero-width characters without segmentation.
func (t *Terminal) extendGrapheme(r rune) bool {
	x, y := t.cur.x, t.cur.y
	if x == 0 {
		return false
	}
	edge := t.cols - 1
	if x <= t.right {
		edge = t.right
	}
	if !t.cur.wrapPending && !(x == edge && !t.autowrap) {
		x--
	}
	row := t.screen()[y]
	if x < 0 || row == nil {
		return false
	}
	if !t.autowrap && x == edge && rowCell(row, x).R == 0 && x > 0 {
		x--
	}
	if x >= len(row) {
		return false
	}
	if row[x].R == wideTail && x > 0 {
		x--
	}
	if row[x].R <= 0 {
		return false
	}
	s := t.cellString(row[x])
	// Outside GB9b's Prepend rule, Ghostty follows UTS #51: a modifier
	// must immediately follow its base. Extend, Control and ZWJ cannot
	// bridge that boundary, including text attached while mode was off.
	if emojiModifier(r) {
		previous, _ := utf8.DecodeLastRuneInString(s)
		if !runeInRanges(previous, emojiModifierBases[:]) && !runeInRanges(previous, graphemePrepends[:]) {
			return false
		}
	}
	// Finish segmentation before a width change can scroll and reuse scratch.
	var continues bool
	t.scratch, continues = graphemeContinues(t.scratch, s, r)
	if !continues {
		return false
	}
	count := utf8.RuneCountInString(s)
	width := 1
	if x+1 < len(row) && row[x+1].R == wideTail {
		width = 2
	}
	prev, _ := utf8.DecodeLastRuneInString(s)
	newWidth, keep := graphemeWidthEffect(width, prev, r)
	if !keep {
		return true
	}
	base := row[x]
	if newWidth > t.cols {
		return true
	}
	if newWidth == 2 && width == 1 && x == edge && !t.autowrap {
		return true
	}
	var modes clusterModes
	var text string
	if base.R >= clusterBase {
		old := t.x.clusters[base.R-clusterBase]
		modes, text = old.joinModes, old.text
	}
	if count < maxClusterRunes {
		modes.set(count)
		base.R = t.internAppend(s, r, uint8(newWidth), modes)
	} else {
		if newWidth == width {
			return true
		}
		// Native applies geometry before its bounded suffix storage drops r.
		// Keep that width change even when the cell text is already full.
		base.R = t.intern(cluster{text: text, width: uint8(newWidth), joinModes: modes})
	}
	if newWidth == width {
		row[x] = base
		t.touch(y)
		return true
	}
	if newWidth == 1 {
		row[x] = base
		// A selector changes geometry, not the old tail's erase colors.
		row[x+1].R = 0
		t.cur.x = min(x+1, edge)
		t.cur.wrapPending = false
		t.touch(y)
		return true
	}
	if x < edge {
		row = t.row(y)
		// Widening keeps the base's original pen and writes the new tail
		// with the current pen. Clear any wide cell displaced by that tail.
		if x+2 < t.cols && row[x+2].R == wideTail {
			row[x+2] = cell{}
		}
		row[x] = base
		tail := t.cur.pen
		tail.R = wideTail
		row[x+1] = tail
		t.cur.x, t.cur.wrapPending = min(x+2, edge), x+2 > edge
		t.touch(y)
		return true
	}
	// At the edge, reuse the normal path for wrapping and scrolling. The
	// transferred base and its new tail both receive the current pen.
	row[x] = cell{}
	t.touch(y)
	t.cur.x, t.cur.wrapPending = x, false
	insert, lastChar := t.insert, t.lastChar
	t.insert = false
	t.print(base.R, 2)
	t.insert, t.lastChar = insert, lastChar
	return true
}

func graphemeWidthEffect(width int, prev, r rune) (int, bool) {
	if r == 0xfe0e || r == 0xfe0f {
		if !emojiVariationBase(prev) {
			return width, false
		}
		if r == 0xfe0f {
			return 2, true
		}
		return 1, true
	}
	// A second spacing scalar widens a cluster. Emoji modifiers contribute
	// no width within a valid cluster, despite their standalone width of two.
	if CodepointWidth(r) > 0 && !emojiModifier(r) && !runeInRanges(r, graphemePrepends[:]) {
		return 2, true
	}
	return width, true
}

func graphemeContinues(b []byte, s string, r rune) ([]byte, bool) {
	// Terminal controls have already been executed. Ghostty treats any
	// zero-width grapheme Control left in a mode-off cell as Other when
	// clustering is later enabled. Replace only those properties for the
	// boundary test; the original UTF-8 is retained in the stored cell.
	b = b[:0]
	for _, cp := range s {
		if runeInRanges(cp, graphemeControls[:]) {
			cp = 0xe000 // a private-use scalar with grapheme property Other
		}
		b = utf8.AppendRune(b, cp)
	}
	boundary := len(b)
	if runeInRanges(r, graphemeControls[:]) {
		r = 0xe000
	}
	b = utf8.AppendRune(b, r)
	iter := graphemes.FromBytes(b)
	for iter.Next() {
		if iter.End() > boundary {
			return b, iter.Start() < boundary
		}
	}
	return b, false
}

func emojiVariationBase(r rune) bool {
	return runeInRanges(r, emojiVariationBases[:])
}

func emojiModifier(r rune) bool { return r >= 0x1f3fb && r <= 0x1f3ff }

func runeInRanges(r rune, ranges [][2]rune) bool {
	lo, hi := 0, len(ranges)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		v := ranges[mid]
		if r < v[0] {
			hi = mid
		} else if r > v[1] {
			lo = mid + 1
		} else {
			return true
		}
	}
	return false
}

// appendCluster starts the base with clustering off, then replays the
// mode each following codepoint was joined under. This also separates a
// new cell from a prior cell ending in ZWJ.
func (t *Terminal) appendCluster(b []byte, c cluster, head, tail cell) []byte {
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
		// A full cluster can change width without retaining its last input.
		// Replay an over-capacity suffix: it changes geometry, not cell text.
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
