package phosphor

//go:generate go run ./internal/unicodegen

import (
	"unicode/utf8"

	"github.com/clipperhouse/uax29/v2/graphemes"
)

// A cluster stores its physical width independently of its first scalar:
// selectors can narrow a wide emoji or widen a text character. joinModes
// records which appended scalars were clustered under mode 2027, by UTF-8
// byte offset, so history can replay mixed-mode cells without joining neighbors.
type cluster struct {
	text      string
	joinModes uint32
	width     uint8
}

func scalarWidth(r rune) int {
	lo, hi := 0, len(scalarWidthRanges)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		v := scalarWidthRanges[mid]
		if r < v.first {
			hi = mid
		} else if r > v.last {
			lo = mid + 1
		} else {
			return int(v.width)
		}
	}
	return 1
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
	if !t.autowrap && x == edge && row[x].R == 0 && x > 0 {
		x--
	}
	if row[x].R == wideTail && x > 0 {
		x--
	}
	if row[x].R <= 0 {
		return false
	}
	s := t.cellString(row[x])
	// Ghostty tailors UAX #29 so a modifier only joins a valid emoji
	// modifier base. Ignore intervening Extend scalars, as rule GB10 does.
	if emojiModifier(r) {
		base := s
		for len(base) > 0 {
			cp, n := utf8.DecodeLastRuneInString(base)
			if runeInRanges(cp, emojiModifierBases[:]) {
				break
			}
			if scalarWidth(cp) != 0 {
				return false
			}
			base = base[:len(base)-n]
		}
		if len(base) == 0 {
			return false
		}
	}
	if !graphemeContinues(s, r) {
		return false
	}
	if len(s)+utf8.RuneLen(r) > maxCluster {
		return true
	}
	width := 1
	if x+1 < t.cols && row[x+1].R == wideTail {
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
	var modes uint32
	if base.R >= clusterBase {
		modes = t.x.clusters[base.R-clusterBase].joinModes
	}
	modes |= 1 << uint(len(s))
	base.R = t.intern(cluster{text: s + string(r), width: uint8(newWidth), joinModes: modes})
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
	if scalarWidth(r) > 0 && !emojiModifier(r) && !runeInRanges(r, graphemePrepends[:]) {
		return 2, true
	}
	return width, true
}

func graphemeContinues(s string, r rune) bool {
	// Terminal controls have already been executed. Ghostty treats any
	// zero-width grapheme Control left in a mode-off cell as Other when
	// clustering is later enabled. Replace only those properties for the
	// boundary test; the original UTF-8 is retained in the stored cell.
	var storage [2*maxCluster + utf8.UTFMax]byte
	b := storage[:0]
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
			return iter.Start() < boundary
		}
	}
	return false
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
	if c.joinModes == 0 {
		return append(b, c.text...)
	}
	mode := false
	style := head
	width := 0
	var prev rune
	for i := 0; i < len(c.text); {
		cp, n := utf8.DecodeRuneInString(c.text[i:])
		next := c.joinModes&(1<<uint(i)) != 0 || i == 0 && scalarWidth(cp) > t.cols
		if next != mode {
			b = appendMode(b, 2027, next)
			mode = next
		}
		if i == 0 {
			width = scalarWidth(cp)
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
	if mode {
		b = appendMode(b, 2027, false)
	}
	return t.appendStyle(b, style, head)
}
