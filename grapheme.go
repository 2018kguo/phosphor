package phosphor

//go:generate go run ./internal/unicodegen

import "unicode/utf8"

// A cluster stores its physical width independently of its first scalar:
// selectors can narrow a wide emoji or widen a text character. joinModes
// records which appended codepoints were clustered under mode 2027, so
// history can replay mixed-mode cells without joining neighboring cells.
type cluster struct {
	text      string
	joinModes clusterModes
	width     uint8
	tail      graphemeTail
}

// The second word holds the final codepoint of a full native-sized cluster.
// This metadata is allocated only for cells containing multiple codepoints.
type clusterModes [2]uint64

func (m clusterModes) has(i int) bool { return m[i/64]&(1<<uint(i%64)) != 0 }
func (m *clusterModes) set(i int)     { m[i/64] |= 1 << uint(i%64) }

// CodepointWidth returns the terminal width of a single scalar:
// zero for combining or ignored scalars, two for wide scalars, otherwise one.
// Invalid scalars have width one. A complete grapheme can have a different
// width under mode 2027, so summing scalar widths measures mode-off layout.
func CodepointWidth(r rune) int {
	return codepointInfo(r).width()
}

func firstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

// extendGrapheme attaches r to the previous cell only when Unicode's
// grapheme boundary rules say it continues that cell's last cluster.
// Re-segmenting the bounded cell text also handles enabling mode 2027
// after older cells collected zero-width characters without segmentation.
func (t *Terminal) extendGrapheme(r rune, info unicodeInfo) bool {
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
	// Two simple scalar properties always start separate cells.
	// Stored clusters still need replay: earlier mode-off text may contain
	// boundaries that affect the suffix's state.
	var tail graphemeTail
	if row[x].R < clusterBase {
		previous := codepointInfo(row[x].R)
		if previous.simple() && info.simple() {
			return false
		}
		tail = graphemeTail{count: 1, lastInfo: previous}
	} else {
		tail = t.clusterTail(row[x])
	}
	s := t.cellString(row[x])
	// Outside GB9b's Prepend rule, Ghostty follows UTS #51: a modifier
	// must immediately follow its base. Extend, Control and ZWJ cannot
	// bridge that boundary, including text attached while mode was off.
	if emojiModifier(r) {
		previous := tail.lastInfo.grapheme()
		if previous&gpModifierBase == 0 && previous&gpKindMask != gpPrepend {
			return false
		}
	}
	if !tail.continues(info) {
		return false
	}
	count := int(tail.count)
	width := 1
	if x+1 < len(row) && row[x+1].R == wideTail {
		width = 2
	}
	newWidth, keep := graphemeWidthEffectInfo(width, tail.lastInfo, info, r)
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

func graphemeSimpleScalar(r rune) bool {
	u := uint32(r)
	if u > utf8.MaxRune {
		return false
	}
	// Unicode 18 permits a GCB Other linker to join a later consonant.
	return codepointInfo(r).simple()
}

func graphemeWidthEffect(width int, prev, r rune) (int, bool) {
	return graphemeWidthEffectInfo(width, codepointInfo(prev), codepointInfo(r), r)
}

func graphemeWidthEffectInfo(width int, previous, current unicodeInfo, r rune) (int, bool) {
	if r == 0xfe0e || r == 0xfe0f {
		if previous&codepointVSBase == 0 {
			return width, false
		}
		if r == 0xfe0f {
			return 2, true
		}
		return 1, true
	}
	// A second spacing scalar widens a cluster. Emoji modifiers contribute
	// no width within a valid cluster, despite their standalone width of two.
	if current.width() > 0 && !emojiModifier(r) && current.grapheme()&gpKindMask != gpPrepend {
		return 2, true
	}
	return width, true
}

func emojiVariationBase(r rune) bool {
	return codepointInfo(r)&codepointVSBase != 0
}

func emojiModifier(r rune) bool { return r >= 0x1f3fb && r <= 0x1f3ff }

// appendCluster starts the base with clustering off, then replays the
// mode each following codepoint was joined under. This also separates a
// new cell from a prior cell ending in ZWJ.
func (t *Terminal) appendCluster(b []byte, c cluster, head, tail cell) []byte {
	if c.joinModes == (clusterModes{}) && int(c.width) == CodepointWidth(firstRune(c.text)) {
		return append(b, c.text...)
	}
	if c.joinModes[1] == 0 && c.joinModes[0] >= 2 {
		count := utf8.RuneCountInString(c.text)
		if count >= 2 && count < 64 && c.joinModes[0] == (uint64(1)<<uint(count))-2 {
			first, n := utf8.DecodeRuneInString(c.text)
			if CodepointWidth(first) <= t.cols && (tail.R != wideTail || styleOf(tail) == styleOf(head)) {
				// Every suffix scalar was joined with clustering on. With
				// no differing wide-tail style, width changes add no bytes.
				b = append(b, c.text[:n]...)
				b = append(b, "\x1b[?2027h"...)
				b = append(b, c.text[n:]...)
				return append(b, "\x1b[?2027l"...)
			}
		}
	}
	mode := false
	style := head
	width := 0
	var prev rune
	for i, index := 0, 0; i < len(c.text); index++ {
		cp, n := utf8.DecodeRuneInString(c.text[i:])
		next := c.joinModes.has(index) || i == 0 && CodepointWidth(cp) > t.cols
		if next != mode {
			if next {
				b = append(b, "\x1b[?2027h"...)
			} else {
				b = append(b, "\x1b[?2027l"...)
			}
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
			b = append(b, "\x1b[?2027h"...)
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
		b = append(b, "\x1b[?2027l"...)
	}
	return t.appendStyle(b, style, head)
}
