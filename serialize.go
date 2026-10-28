package phosphor

import (
	"bytes"
	"encoding/binary"
	"math/bits"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
	"unsafe"
)

var sgrAttributeCodes = [...]struct {
	bit  Attr
	code string
}{
	{AttrBold, ";1"}, {AttrFaint, ";2"}, {AttrItalic, ";3"}, {AttrBlink, ";5"},
	{AttrInverse, ";7"}, {AttrHidden, ";8"}, {AttrStrikethrough, ";9"}, {AttrOverline, ";53"},
}

// appendSGR appends the escape sequence that sets exactly c's style,
// starting from a reset.
func (t *Terminal) appendSGR(b []byte, c cell) []byte {
	b = append(b, "\x1b[0"...)
	// Range by reference so each style does not copy the encoding table.
	for _, a := range &sgrAttributeCodes {
		if c.A&a.bit != 0 {
			b = append(b, a.code...)
		}
	}
	switch ul := (c.A & ulMask) >> ulShift; ul {
	case 0:
	case 1:
		b = append(b, ";4"...)
	default:
		b = append(b, ";4:"...)
		b = strconv.AppendInt(b, int64(ul), 10)
	}
	b = appendColor(b, c.Fg, 30)
	b = appendColor(b, c.Bg, 40)
	b = appendColor(b, t.extraAt(c.E).underline, 50)
	return append(b, 'm')
}

func appendColor(b []byte, c Color, base int) []byte {
	switch {
	case c == 0:
		return b
	case c&colorRGB != 0:
		b = append(b, ';')
		b = strconv.AppendInt(b, int64(base+8), 10)
		b = append(b, ";2;"...)
		b = strconv.AppendInt(b, int64(c>>16&0xff), 10)
		b = append(b, ';')
		b = strconv.AppendInt(b, int64(c>>8&0xff), 10)
		b = append(b, ';')
		return strconv.AppendInt(b, int64(c&0xff), 10)
	}
	n := int(c & 0xff)
	b = append(b, ';')
	switch {
	case n < 8 && base != 50:
		return strconv.AppendInt(b, int64(base+n), 10)
	case n < 16 && base != 50:
		return strconv.AppendInt(b, int64(base+60+n-8), 10)
	}
	b = strconv.AppendInt(b, int64(base+8), 10)
	b = append(b, ";5;"...)
	return strconv.AppendInt(b, int64(n), 10)
}

func styleOf(c cell) cell { return cell{Fg: c.Fg, Bg: c.Bg, A: c.A, E: c.E} }

// appendRow appends one row's characters with the escape sequences for
// their styles, starting and ending in the default style. With trim,
// trailing default blanks are left out.
func (t *Terminal) appendRow(b []byte, row []cell, width int, trim bool) []byte {
	return t.appendRowHistory(b, row, width, trim, false)
}

// appendRowHistory keeps sparse history cells distinct from literal spaces.
// The local tag precedes the spaces the VT formatter already needs, so rows
// still append in place and ordinary log lines keep their original bytes.
func (t *Terminal) appendRowHistory(b []byte, row []cell, width int, trim, history bool) []byte {
	if row == nil {
		if !trim {
			if history {
				b = append(b, 0)
				b = binary.AppendUvarint(b, uint64(width))
			}
			for range width {
				b = append(b, ' ')
			}
		}
		return b
	}
	end := width
	if trim {
		end = min(end, contentEnd(row))
	}
	var style cell
	// The style as two words (Fg; Bg, A and E), for two compares a cell.
	var fg Color
	var rest uint64
	cells := row[:min(end, len(row))]
	if history {
		// Stored default zeros and the implicit tail are one hole run.
		// Its encoding must not depend on the row's allocated prefix.
		for len(cells) > 0 && cells[len(cells)-1] == (cell{}) {
			cells = cells[:len(cells)-1]
		}
	}
	for i := 0; i < len(cells); {
		c := &cells[i]
		if c.Fg != fg || *(*uint64)(unsafe.Pointer(&c.Bg)) != rest {
			if c.E != style.E || (c.A^style.A)&AttrProtected != 0 {
				b = t.appendStyle(b, style, *c)
			} else {
				b = t.appendSGR(b, *c)
			}
			style = styleOf(*c)
			fg, rest = c.Fg, *(*uint64)(unsafe.Pointer(&c.Bg))
		}
		if history && c.R == 0 {
			end := i + 1
			for end < len(cells) && cells[end].R == 0 && styleOf(cells[end]) == style {
				end++
			}
			b = append(b, 0)
			b = binary.AppendUvarint(b, uint64(end-i))
			for ; i < end; i++ {
				b = append(b, ' ')
			}
			continue
		}
		if uint32(c.R-1) < utf8.RuneSelf-1 {
			b = slices.Grow(b, len(cells)-i)
			n := copyASCIICells(b[len(b):cap(b)], cells[i:])
			b = b[:len(b)+n]
			i += n
			continue
		}
		i++
		r := c.R
		switch {
		case r == wideTail: // printed with its left half (same style)
		case r == 0, r == spacerHead:
			if r == 0 && t.cols == 1 && !history {
				// A wide glyph cannot fit in one column: native writes an
				// empty cell with the current pen and sets pending wrap.
				b = utf8.AppendRune(b, '界')
			} else {
				b = append(b, ' ')
			}
		case r >= clusterBase:
			var tail cell
			if i < len(cells) && cells[i].R == wideTail {
				tail = cells[i]
			}
			b = t.appendCluster(b, t.x.clusters[r-clusterBase], *c, tail)
		default:
			b = utf8.AppendRune(b, r)
		}
	}
	if end > len(cells) {
		b = t.appendStyle(b, style, cell{})
		style = cell{}
	}
	if style.A&AttrProtected != 0 {
		b = appendProtect(b, false)
	}
	if t.link(style.E) != "" {
		b = t.appendLink(b, 0)
	}
	if style.A&^AttrProtected != 0 || style.Fg != 0 || style.Bg != 0 || t.extraAt(style.E).underline != 0 {
		b = append(b, "\x1b[0m"...)
	}
	if padding := end - len(cells); padding > 0 {
		if history {
			b = append(b, 0)
			b = binary.AppendUvarint(b, uint64(padding))
		}
		for range padding {
			b = append(b, ' ')
		}
	}
	return b
}

// copyASCIICells writes the initial ASCII run in one style. The destination
// has room for every cell, so the copy loop needs no per-byte growth check.
func copyASCIICells(dst []byte, src []cell) int {
	dst = dst[:len(src)]
	fg := src[0].Fg
	rest := *(*uint64)(unsafe.Pointer(&src[0].Bg))
	for i := range src {
		c := &src[i]
		if uint32(c.R-1) >= utf8.RuneSelf-1 || c.Fg != fg || *(*uint64)(unsafe.Pointer(&c.Bg)) != rest {
			return i
		}
		dst[i] = byte(c.R)
	}
	return len(src)
}

// appendStyle appends what changes the style from to that of c.
func (t *Terminal) appendStyle(b []byte, from, c cell) []byte {
	s := styleOf(c)
	oldExtra, newExtra := t.extraAt(from.E), t.extraAt(s.E)
	if (s.A^from.A)&AttrProtected != 0 {
		b = appendProtect(b, s.A&AttrProtected != 0)
	}
	if (s.A^from.A)&^AttrProtected != 0 || s.Fg != from.Fg || s.Bg != from.Bg || oldExtra.underline != newExtra.underline {
		b = t.appendSGR(b, s)
	}
	if oldExtra.link != newExtra.link {
		b = t.appendLink(b, s.E)
	}
	return b
}

// appendLink appends OSC 8 starting hyperlink l, or ending one (0).
func (t *Terminal) appendLink(b []byte, l uint16) []byte {
	b = append(b, "\x1b]8;"...)
	if target := t.link(l); target == "" {
		b = append(b, ';')
	} else {
		b = append(b, target...)
	}
	return append(b, "\x1b\\"...)
}

// appendProtect appends DECSCA, turning character protection on or off.
func appendProtect(b []byte, on bool) []byte {
	if on {
		return append(b, "\x1b[1\"q"...)
	}
	return append(b, "\x1b[0\"q"...)
}

// appendWrapped appends a row that continues on next: all its cells (blanks
// as spaces, so the row is full and the next character wraps), but for a
// blank the wide character starting next left behind.
func (t *Terminal) appendWrapped(b []byte, row, next []cell, saved *bool) []byte {
	end := t.wrapEnd(row, next)
	return t.appendRepaintRow(b, row, end, false, saved)
}

// appendRepaintRow repairs the empty cells that row printing temporarily
// makes literal spaces. DECSC/DECRC preserve the pending wrap while ECH
// repairs the row before the next print can move it into history.
func (t *Terminal) appendRepaintRow(b []byte, row []cell, width int, trim bool, saved *bool) []byte {
	end := width
	if trim {
		end = min(width, contentEnd(row))
	}
	if end == t.cols && t.canStageNarrowEdge(row) {
		// Build a completed narrow cluster where its wide base fits, then
		// shift it into the final column. Saving a temporary edge print
		// keeps this row's pending wrap available to the next painted row.
		b = t.appendRow(b, row[:end-1], end-1, false)
		b = append(b, ' ', 0x1b, '7')
		*saved = true
		b = appendHPA(b, end-2)
		b = t.appendRow(b, row[end-1:], 1, false)
		b = appendHPA(b, end-2)
		b = append(b, "\x1b[@"...)
		x := end - 2
		if row[x].R == wideTail {
			x--
		}
		b = appendHPA(b, x)
		b = t.appendRow(b, row[x:end-1], end-1-x, false)
		b = append(b, "\x1b8"...)
	} else {
		b = t.appendRow(b, row, width, trim)
	}
	return t.appendRepaintHoles(b, row, end, 0, saved)
}

func appendHPA(b []byte, x int) []byte {
	b = append(b, "\x1b["...)
	b = strconv.AppendInt(b, int64(x+1), 10)
	return append(b, 'G')
}

func (t *Terminal) canStageNarrowEdge(row []cell) bool {
	if t.cols < 2 || len(row) != t.cols || row[t.cols-1].R < clusterBase {
		return false
	}
	last := t.x.clusters[row[t.cols-1].R-clusterBase]
	if last.width != 1 || clusterReplayWidth(last) < 2 {
		return false
	}
	if previous := row[t.cols-2].R; previous >= clusterBase {
		c := t.x.clusters[previous-clusterBase]
		if c.width == 1 && clusterReplayWidth(c) > 1 {
			return false // restoring it would overwrite the staged last cell
		}
	}
	return true
}

func clusterReplayWidth(c cluster) int {
	width, peak, index := 0, 0, 0
	var previous rune
	for _, cp := range c.text {
		if index == 0 {
			width = CodepointWidth(cp)
		} else if c.joinModes.has(index) {
			width, _ = graphemeWidthEffect(width, previous, cp)
		}
		peak = max(peak, width)
		previous, index = cp, index+1
	}
	return peak
}

func (t *Terminal) appendRepaintHoles(b []byte, row []cell, end, offset int, saved *bool) []byte {
	started := false
	for x := 0; x < end; {
		if !repaintHole(rowCell(row, x)) {
			x++
			continue
		}
		bg := rowCell(row, x).Bg
		to := x + 1
		for to < end && repaintHole(rowCell(row, to)) && rowCell(row, to).Bg == bg {
			to++
		}
		if !started {
			b = append(b, "\x1b7"...)
			started, *saved = true, true
		}
		b = t.appendSGR(b, cell{Bg: bg})
		b = appendHPA(b, offset+x)
		b = append(b, "\x1b["...)
		b = strconv.AppendInt(b, int64(to-x), 10)
		if offset+to == t.cols {
			// Inserting through the right edge replaces only these holes.
			// ECH would also clear a wrap already restored by screen painting.
			b = append(b, '@')
		} else {
			b = append(b, 'X')
		}
		x = to
	}
	if started {
		b = append(b, "\x1b8"...)
	}
	return b
}

// appendPendingCursor recreates wrap at the physical edge, then uses CBT
// to return to a saved column without clearing it. Tabs are restored before
// saved cursors are built, so the temporary stop can be removed immediately.
func (t *Terminal) appendPendingCursor(b []byte, scr [][]cell, c cursor) []byte {
	edge := t.cols - 1
	temporaryTab := c.x > 0 && c.x < edge && !t.isTab(c.x)
	if temporaryTab {
		b = appendCUP(b, c.y, c.x)
		b = append(b, "\x1bH"...)
	}
	row := rowAt(scr, c.y)
	x, end := edge, edge+1
	if x > 0 && rowCell(row, x).R == wideTail {
		x--
	}
	b = appendCUP(b, c.y, x)
	b = append(b, "\x1b[0m\x1b[0\"q\x1b]8;;\x1b\\\x1b(B\x1b)B\x1b*B\x1b+B\x0f"...)
	if row == nil {
		if t.cols == 1 {
			b = utf8.AppendRune(b, '界')
		} else {
			b = append(b, ' ')
		}
	} else {
		row = row[min(x, len(row)):min(end, len(row))]
		b = t.appendRow(b, row, end-x, false)
	}
	if t.cols > 1 {
		usedSave := false
		b = t.appendRepaintHoles(b, row, end-x, x, &usedSave)
	}
	if c.x < edge {
		count := 1
		for stop := c.x + 1; stop < edge; stop++ {
			if t.isTab(stop) {
				count++
			}
		}
		b = append(b, "\x1b["...)
		b = strconv.AppendInt(b, int64(count), 10)
		b = append(b, 'Z')
		if temporaryTab {
			b = append(b, "\x1b[g"...)
		}
	}
	return b
}

// Erase commands create empty cells with only a background. Preserve the
// formatter's existing styled-space fallback for a clipped wide cell whose
// retained foreground or attributes cannot be encoded as an ANSI empty cell.
func repaintHole(c cell) bool {
	return c.R == 0 && c.Fg == 0 && c.A == 0 && c.E == 0
}

// appendSavedCursor reinstates a screen's DECSC slot after painting used it.
// With full margins, origin mode can save an arbitrary physical position.
func (t *Terminal) appendSavedCursor(b []byte, slot int, originOn bool) []byte {
	var s savedCursor
	if t.saved != nil {
		s = t.saved.screen[slot]
	}
	if !s.valid {
		// Before margins and modes are restored, DECSTR can forget the
		// temporary DECSC slot without changing the painted cells.
		return append(b, "\x1b[!p\x1b[?25l"...)
	}
	switch {
	case s.origin:
		b = append(b, "\x1b[?6h"...)
	case originOn:
		b = append(b, "\x1b[?6l"...)
	}
	if s.wrapPending {
		scr := t.main
		if slot == 1 {
			scr = t.alt
		}
		b = t.appendPendingCursor(b, scr, s.cursor)
	} else {
		b = appendCUP(b, s.y, s.x)
	}
	b = t.appendSGR(b, s.pen)
	b = appendProtect(b, s.pen.A&AttrProtected != 0)
	b = appendCharsets(b, s.cursor)
	if s.ss != 0 {
		b = append(b, 0x1b, 'L'+s.ss)
	}
	b = append(b, "\x1b7"...)
	b = appendProtect(b, false)
	if s.origin || s.ss != 0 {
		b = append(b, "\x1b[?6l"...)
	}
	if s.ss != 0 {
		// At column zero this combining mark has no base. It consumes the
		// temporary single shift without changing cells or the saved slot.
		b = append(b, "\x1b[H\u0300"...)
	}
	return append(b, "\x1b[0m\x1b(B\x1b)B\x1b*B\x1b+B\x0f"...)
}

func appendHistoryRepaint(b, line []byte, cols int, open bool, saved *bool) []byte {
	sparse := false
	if bytes.IndexByte(line, 0) >= 0 {
		walkHistory(line, nil, func(start, spaces, end int) { sparse = true }, func(start, end int, flags rowFlags) { sparse = true })
	}
	clustered := bytes.Contains(line, []byte("\xef\xb8\x8e")) && bytes.Contains(line, []byte("\x1b[?2027h"))
	if !sparse && !clustered {
		return append(b, line...)
	}
	tmp := replayLine(line, cols)
	if !sparse {
		stage := false
		for _, row := range tmp.main[:tmp.cur.y+1] {
			stage = stage || tmp.canStageNarrowEdge(row)
		}
		if !stage {
			return append(b, line...)
		}
	}
	for y, row := range tmp.main[:tmp.cur.y+1] {
		before := len(b)
		if y < tmp.cur.y {
			end := cols
			if rowCell(row, end-1).R == spacerHead && startsWide(tmp.main[y+1]) {
				end-- // history omits the spacer, regardless of its style
			}
			b = tmp.appendRepaintRow(b, row, end, false, saved)
		} else if open {
			end := tmp.cur.x
			if tmp.cur.wrapPending {
				end++
			}
			b = tmp.appendRepaintRow(b, row, end, false, saved)
		} else {
			b = tmp.appendRepaintRow(b, row, cols, true, saved)
		}
		if y > 0 && len(b) == before {
			b = tmp.appendRepaintRow(b, []cell{{}}, 1, false, saved)
		}
		if flags := rowFlagsAt(tmp.mainWrap, y); flags&rowSemanticMask != 0 {
			b = appendPromptMarker(b, flags)
			b = append(b, "\x1b]133;D\x1b\\"...)
		}
	}
	return b
}

func appendCUP(b []byte, y, x int) []byte {
	y, x = max(y, 0), max(x, 0) // a position left of or above the origin
	b = append(b, "\x1b["...)
	b = strconv.AppendInt(b, int64(y+1), 10)
	b = append(b, ';')
	b = strconv.AppendInt(b, int64(x+1), 10)
	return append(b, 'H')
}

func appendMode(b []byte, mode int, on bool) []byte {
	b = append(b, "\x1b[?"...)
	b = strconv.AppendInt(b, int64(mode), 10)
	if on {
		return append(b, 'h')
	}
	return append(b, 'l')
}

func appendCharsets(b []byte, c cursor) []byte {
	for i, g := range c.g {
		b = append(b, 0x1b, "()*+"[i], [...]byte{'B', '0', 'A'}[g])
	}
	return append(b, [...]string{"\x0f", "\x0e", "\x1bn", "\x1bo"}[c.gl]...)
}

// Repaint returns a byte sequence that, written to a terminal of the same
// size in its initial state, reproduces this one: the scrollback (when
// withScrollback is set) followed by the main screen with its soft wraps,
// then the alternate screen if active (or kept by mode 47), tab stops,
// modes saved with XTSAVE, the saved cursor, the scroll region and
// margins, the cursor with its pending wrap, pen, character sets and
// shape, the modes the program set (wraparound, insert, keypad and cursor
// keys, mouse tracking and encoding, focus events, bracketed paste, kitty
// keyboard flags...) and the title. Feeding it to a new Terminal of the
// same size gives the same screen, cursor, modes and scrollback; it is
// also meant for real terminals, so a client can attach to a running
// program, and a checkpoint can be restored by replaying it.
//
// Unfinished escape sequences and UTF-8 characters are restored after the
// repaint, so the next output continues them instead of printing their
// remaining bytes. A synchronized update in progress stays in progress.
// Colors set by the program (OSC 4, 10, 11, 12) are not included, since
// replaying those would change the receiving terminal's own colors.
//
// The output is wrapped in a synchronized update (mode 2026) and hides the
// cursor while painting.
func (t *Terminal) Repaint(withScrollback bool) []byte {
	n := 256 + t.rows*(t.cols+16)
	if withScrollback {
		n += t.sbBytes
	}
	return t.AppendRepaint(make([]byte, 0, n), withScrollback)
}

// AppendRepaint appends Repaint's output to b and returns the extended
// buffer, for callers that reuse one.
func (t *Terminal) AppendRepaint(b []byte, withScrollback bool) []byte {
	// Synchronized update, normalize the receiving terminal, hide the
	// cursor while painting.
	b = append(b, "\x1b[?2026h\x1b[?2027l\x1b[?1049l\x1b[?5l\x1b[?69l\x1b[0\"q\x1b[0m\x1b[r\x1b[?6l\x1b[?7h\x1b[4l\x1b(B\x1b)B\x1b*B\x1b+B\x0f\x1b[?25l\x1b[H\x1b[2J"...)
	var savedModes []savedMode
	if t.x != nil {
		savedModes = t.x.savedModes
	}
	b = t.appendColumnModes(b, savedModes)
	b, entryMode, exitMode := t.appendScreenModes(b, savedModes)
	b = append(b, "\x1b[!p\x1b[?25l"...)
	b = t.appendPromptRedraw(b)
	sbOpen := false
	usedSave := false
	if withScrollback {
		lines := t.sb[t.sbStart:]
		for i, line := range lines {
			b = appendHistoryRepaint(b, line, t.cols, i == len(lines)-1 && t.sbOpen, &usedSave)
			b = append(b, "\r\n"...)
		}
		if sbOpen = t.sbOpen && len(lines) > 0; sbOpen {
			b = b[:len(b)-2] // the screen's first row goes on with it
		}
	}
	// The main screen flows in as lines, so with scrollback above it ends
	// up exactly on screen. A full-width last row leaves a pending wrap,
	// never a scroll.
	if sbOpen && t.sbSpacer && !startsWide(t.main[0]) {
		// The line left out a spacer head, but row 0 no longer starts with
		// the wide character that made it: keep the column.
		b = append(b, ' ')
	}
	b = t.appendScreen(b, t.main, t.mainWrap, sbOpen, false, &usedSave)
	b = t.appendRowSemantics(b, t.mainWrap)
	if t.tabs() != nil {
		b = append(b, "\x1b[3g"...)
		for x := range t.cols {
			if t.isTab(x) {
				b = appendCUP(b, 0, x)
				b = append(b, "\x1bH"...)
			}
		}
	}
	if t.altActive {
		if usedSave || t.saved != nil && t.saved.screen[0].valid {
			b = t.appendSavedCursor(b, 0, false)
			usedSave = false
		}
		p := t.x.inactivePrimary
		b = t.appendSemanticCursor(b, 0, p.y)
		if p.pending {
			c := t.cur
			c.x, c.y, c.wrapPending, c.origin = p.x, p.y, p.pending, false
			b = t.appendPendingCursor(b, t.main, c)
		} else {
			b = appendCUP(b, p.y, p.x)
		}
		if entryMode == 1049 && t.saved != nil && t.saved.screen[0].valid {
			c := t.saved.screen[0].cursor
			b = t.appendSGR(b, c.pen)
			b = appendProtect(b, c.pen.A&AttrProtected != 0)
			b = appendCharsets(b, c)
			if c.ss != 0 {
				b = append(b, 0x1b, 'L'+c.ss)
			}
		}
		b = appendMode(b, entryMode, true)

		b = append(b, "\x1b[!p\x1b[?25l\x1b[?6l\x1b[0m\x1b[0\"q\x1b]8;;\x1b\\\x1b(B\x1b)B\x1b*B\x1b+B\x0f\x1b[H\x1b[2J"...)
		b = t.appendScreen(b, t.alt, t.altWrap, false, true, &usedSave)
		b = t.appendRowSemantics(b, t.altWrap)
	} else if t.alt != nil || t.saved != nil && t.saved.screen[1].valid {
		// A kept alternate screen must remain available to the next switch.
		usedSave = usedSave || entryMode == 1049
		b = appendMode(b, entryMode, true)
		b = append(b, "\x1b[!p\x1b[?25l\x1b[H\x1b[2J"...)
		altUsedSave := false
		b = t.appendScreen(b, t.alt, t.altWrap, false, true, &altUsedSave)
		b = t.appendRowSemantics(b, t.altWrap)
		if altUsedSave || t.saved != nil && t.saved.screen[1].valid {
			b = t.appendSavedCursor(b, 1, false)
		}
		b = t.appendSemanticCursor(b, 1, 0)
		if exitMode == 1047 {
			b = t.appendSGR(b, cell{Bg: t.alternateEraseBackground()})
			if t.protISO {
				b = append(b, "\x1bV"...)
			}
		}
		b = appendMode(b, exitMode, false)
	}

	for _, m := range savedModes {
		if m.mode == 3 || m.mode == 40 || screenModeBit(m.mode) != 0 {
			continue
		}
		b = appendMode(b, m.mode, m.on)
		b = append(b, "\x1b[?"...)
		b = strconv.AppendInt(b, int64(m.mode), 10)
		b = append(b, 's')
	}
	originOn := false // whether the restored modes left origin mode on
	for _, m := range savedModes {
		if m.mode == 3 || m.mode == 40 || screenModeBit(m.mode) != 0 {
			continue
		}
		on, _ := t.modeValue(m.mode)
		b = appendMode(b, m.mode, on && m.mode != 25)
		if m.mode == 6 {
			originOn = on
		}
	}
	slot := 0
	if t.altActive {
		slot = 1
	}
	if usedSave || t.saved != nil && t.saved.screen[slot].valid {
		b = t.appendSavedCursor(b, slot, originOn)
		originOn = false
	}
	if t.top != 0 || t.bot != t.rows-1 {
		b = append(b, "\x1b["...)
		b = strconv.AppendInt(b, int64(t.top+1), 10)
		b = append(b, ';')
		b = strconv.AppendInt(b, int64(t.bot+1), 10)
		b = append(b, 'r')
	}
	if t.lrmm {
		b = appendMode(b, 69, true)
		if !t.fullWidth() {
			b = append(b, "\x1b["...)
			b = strconv.AppendInt(b, int64(t.left+1), 10)
			b = append(b, ';')
			b = strconv.AppendInt(b, int64(t.right+1), 10)
			b = append(b, 's')
		}
	}
	// Cursor. Origin mode homes the cursor when switched on, so it is
	// enabled first and followed by a region-relative CUP.
	b = appendMode(b, 6, false)
	originOn = false
	b = t.appendSemanticCursor(b, t.semanticSlot(), t.cur.y)
	x, y := t.cur.x, t.cur.y
	if originOn && (!t.cur.origin || t.cur.wrapPending) {
		// The restored modes left origin mode on, and the CUP below is
		// absolute.
		b = append(b, "\x1b[?6l"...)
	}
	if !t.autowrap && t.cur.wrapPending {
		// Disable wrapping before the final edge print so restoring a
		// one-column empty cell does not advance to another row.
		b = appendMode(b, 7, false)
	}
	savedSlot := 0
	if t.altActive {
		savedSlot = 1
	}
	invalidSaved := t.saved == nil || !t.saved.screen[savedSlot].valid
	canRepairPending := t.cur.wrapPending && invalidSaved && !t.cur.origin && t.top == 0 && t.bot == t.rows-1 && t.left == 0 && t.right == t.cols-1
	if canRepairPending {
		b = t.appendPendingCursor(b, t.screen(), t.cur)
		// Native DECSTR preserves pending wrap while forgetting the
		// temporary DECSC slot used to turn printed spaces back into holes.
		b = append(b, "\x1b[!p\x1b[?25l"...)
		if t.lrmm {
			b = appendMode(b, 69, true)
		}
	} else if t.cur.wrapPending {
		// A tab can move the cursor while leaving wrap pending. Recreate
		// the wrap at the edge, then use CBT to return without clearing it.
		row := t.screen()[y]
		edge := t.cols - 1
		if x <= t.right {
			edge = t.right
		}
		temporaryTab := x > 0 && x < edge && !t.isTab(x)
		if temporaryTab {
			b = appendCUP(b, y, x)
			b = append(b, "\x1bH"...)
		}
		cx, end := edge, edge+1
		if cx > 0 && rowCell(row, cx).R == wideTail {
			cx--
		}
		b = appendCUP(b, y, cx)
		b = append(b, "\x1b[0m"...)
		if row == nil {
			if t.cols == 1 {
				b = utf8.AppendRune(b, '界')
			} else {
				b = append(b, ' ')
			}
		} else {
			b = t.appendRow(b, row[min(cx, len(row)):min(end, len(row))], end-cx, false)
		}
		if x < edge {
			count := 1
			for stop := x + 1; stop < edge; stop++ {
				if t.isTab(stop) {
					count++
				}
			}
			b = append(b, "\x1b["...)
			b = strconv.AppendInt(b, int64(count), 10)
			b = append(b, 'Z')
			if temporaryTab {
				b = append(b, "\x1b[g"...)
			}
		}
		if t.cur.origin {
			b = append(b, "\x1b[?6h"...)
			// Origin mode homes the cursor; the pending wrap cannot be
			// recreated in that combination, so land on the last column.
			b = appendCUP(b, y-t.top, x-t.left)
		}
	} else if t.canStageOriginCursor() {
		// ED22 on a scratch alternate screen can reach physical positions
		// above/left of origin margins. Mode 47 copies that cursor back
		// without touching the painted main screen or either DECSC slot.
		b = append(b, "\x1b[?47h\x1b[?6h\x1b[0m.\x1b[22J"...)
		if y > 0 {
			b = append(b, "\x1b["...)
			b = strconv.AppendInt(b, int64(y), 10)
			b = append(b, 'B')
		}
		if x > 0 {
			b = append(b, "\x1b["...)
			b = strconv.AppendInt(b, int64(x), 10)
			b = append(b, 'C')
		}
		mode := 47
		if t.privateModes()&privateMode47 != 0 {
			mode = 1047
		}
		b = appendMode(b, mode, false)
	} else if t.cur.origin {
		b = append(b, "\x1b[?6h"...)
		b = appendCUP(b, y-t.top, x-t.left)
	} else {
		b = appendCUP(b, y, x)
	}
	b = t.appendSGR(b, t.cur.pen)
	if t.link(t.cur.pen.E) != "" {
		b = t.appendLink(b, t.cur.pen.E)
	}
	switch prot := t.cur.pen.A&AttrProtected != 0; {
	case t.protISO && prot:
		b = append(b, "\x1bV"...)
	case t.protISO:
		b = append(b, "\x1bV\x1bW"...) // ISO protection mode, nothing protected
	case prot:
		b = appendProtect(b, true)
	}
	b = appendCharsets(b, t.cur)
	if t.cur.ss != 0 {
		b = append(b, 0x1b, 'L'+t.cur.ss) // SS2 is ESC N, SS3 ESC O
	}
	if !t.autowrap {
		b = appendMode(b, 7, false)
	}
	if t.reverseVideo {
		b = appendMode(b, 5, true)
	}
	if t.privateModes()&privateModeSlowScroll != 0 {
		b = appendMode(b, 4, true)
	}
	if t.revWrap {
		b = appendMode(b, 45, true)
	}
	if t.revWrapExt {
		b = appendMode(b, 1045, true)
	}
	if t.insert {
		b = append(b, "\x1b[4h"...)
	}
	if t.newline {
		b = append(b, "\x1b[20h"...)
	}
	if t.appCursor {
		b = appendMode(b, 1, true)
	}
	if t.appKeypad {
		b = append(b, "\x1b="...)
	}
	b = appendMouseModes(b, t.mouse, mouseTrackingNumbers)
	b = appendMouseModes(b, t.mouseEnc, mouseEncodingNumbers)
	if t.focus {
		b = appendMode(b, 1004, true)
	}
	if t.bracketed {
		b = appendMode(b, 2004, true)
	}
	for _, f := range t.kitty {
		b = append(b, "\x1b[>"...)
		b = strconv.AppendInt(b, int64(f), 10)
		b = append(b, 'u')
	}
	if t.cursorStyle.shape != CursorDefault {
		b = append(b, "\x1b["...)
		b = strconv.AppendInt(b, int64(t.cursorStyle.parameter()), 10)
		b = append(b, " q"...)
	}
	b = appendMode(b, 12, t.cursorStyle.blink)
	if t.title != "" {
		b = append(b, "\x1b]2;"...)
		b = append(b, t.title...)
		b = append(b, '\a')
	}
	if !t.cursorHidden {
		b = append(b, "\x1b[?25h"...)
	}
	b = append(b, "\x1b[?2026l"...)
	if t.sync {
		b = appendMode(b, 2026, true)
	}
	if t.grapheme {
		b = appendMode(b, 2027, true)
	}
	return t.appendContinuation(b)
}

func (t *Terminal) canStageOriginCursor() bool {
	return t.cur.origin && !t.cur.wrapPending && !t.altActive && t.altBlank() &&
		t.privateModes()&(privateMode47|privateMode1047) != privateMode47|privateMode1047 &&
		(t.cur.x < t.left || t.cur.y < t.top) && t.cur.x <= t.right && t.cur.y <= t.bot
}

// appendContinuation reconstructs the unfinished parser prefix from its
// bounded state. Replaying the original input would repeat controls which
// already moved the cursor or rang the bell. Keeping no separate raw log
// also leaves Write's fast path and an idle terminal's size unchanged.
func (t *Terminal) appendContinuation(b []byte) []byte {
	switch t.state {
	case stGround:
		return append(b, t.utf8buf[:t.utf8n]...)
	case stEsc:
		return append(b, 0x1b)
	case stEscInter:
		return append(b, 0x1b, t.inter)
	case stCSI:
		b = append(b, '\x1b', '[')
		if t.marker != 0 {
			b = append(b, t.marker)
		}
		if t.paramSet || t.nparams != 0 {
			for i := 0; i <= min(int(t.nparams), maxParams-1); i++ {
				if i > 0 {
					separator := byte(';')
					if t.subParam(i) {
						separator = ':'
					}
					b = append(b, separator)
				}
				if i != int(t.nparams) || t.paramSet {
					b = strconv.AppendInt(b, int64(t.params[i]), 10)
				}
			}
			if t.nparams == maxParams {
				// Further parameter bytes are ignored once the array fills.
				b = append(b, ';')
			}
		}
		if t.inter != 0 {
			b = append(b, t.inter)
		}
		return b
	case stCSIIgnore:
		// A second private marker enters the ignore state without effects.
		return append(b, "\x1b[??"...)
	case stOSC:
		b = append(b, '\x1b', ']')
		b = append(b, t.osc...)
		if t.oscOver {
			// The retained payload is capped; one more byte restores the
			// overflow flag, so its eventual terminator cannot dispatch it.
			b = append(b, ' ')
		}
		return b
	case stDCS:
		b = append(b, '\x1b', 'P')
		b = append(b, t.osc...)
		return b
	case stStr:
		// Ignored strings have no retained payload. SOS recreates the
		// same wait for ST without enabling a graphics protocol.
		b = append(b, '\x1b', 'X')
		return b
	default:
		panic("invalid terminal parser state")
	}
}

// appendScreen paints rows from the top-left corner, recreating soft
// wraps: a row that continues on the next is painted whole and the next
// row follows without a line break, so the receiving terminal wraps it.
// With open, the first row continues a line already painted. With
// addressed, rows are reached with cursor moves (blank rows skipped);
// otherwise they flow in separated by CR LF.
func (t *Terminal) appendScreen(b []byte, scr [][]cell, wrap []rowFlags, open, addressed bool, saved *bool) []byte {
	for y, row := range scr {
		cont := open && y == 0 || y > 0 && wrap != nil && wrap[y-1]&rowWrap != 0
		next := y < len(scr)-1 && wrap != nil && wrap[y]&rowWrap != 0
		if addressed && !cont {
			if !next && zeroRow(row) {
				continue
			}
			b = appendCUP(b, y, 0)
		}
		n := len(b)
		if next {
			b = t.appendWrapped(b, row, scr[y+1], saved)
		} else {
			b = t.appendRepaintRow(b, row, t.cols, true, saved)
		}
		if cont && len(b) == n {
			// Something must be printed for the wrap to happen.
			c := cell{R: ' '}
			if len(row) > 0 {
				c = styleOf(row[0])
				c.R = ' '
			}
			if rowCell(row, 0).R == 0 {
				c.R = 0
			}
			b = t.appendRepaintRow(b, []cell{c}, 1, false, saved)
		}
		if cont && !next && contentEnd(row) < t.cols {
			// The wrap may have scrolled in a row painted in the current
			// background: clear the rest of it.
			b = append(b, "\x1b[K"...)
		}
		if !addressed && !next && y < len(scr)-1 {
			b = append(b, "\r\n"...)
		}
	}
	return b
}

// rowText returns a row's characters with trailing blanks removed.
func (t *Terminal) rowText(row []cell) string {
	end := len(row)
	for end > 0 {
		r := row[end-1].R
		if r != 0 && r != ' ' && r != spacerHead {
			break
		}
		end--
	}
	row = row[:end]
	var sb strings.Builder
	sb.Grow(len(row))
	for i := range row {
		switch r := row[i].R; {
		case uint32(r-1) < utf8.RuneSelf-1:
			sb.WriteByte(byte(r))
		case r == wideTail:
		case r == 0, r == spacerHead:
			sb.WriteByte(' ')
		case r >= clusterBase:
			sb.WriteString(t.x.clusters[r-clusterBase].text)
		default:
			sb.WriteRune(r)
		}
	}
	return strings.TrimRight(sb.String(), " ")
}

// stripSGR removes the escape sequences (SGR, EL, DECSCA, OSC 8) a
// scrollback line holds.
func stripSGR(line []byte) string {
	var sb strings.Builder
	walkHistory(line, func(text []byte) { sb.Write(text) }, ignoreHistoryHoles, ignoreHistoryRows)
	return sb.String()
}

func ignoreHistoryHoles(start, spaces, end int)        {}
func ignoreHistoryRows(start, end int, flags rowFlags) {}

func validateHistoryLine(line []byte) bool {
	return walkHistory(line, nil, ignoreHistoryHoles, ignoreHistoryRows)
}

// walkHistoryText scans the bounded grammar produced by appendRow: text,
// CSI styles/modes/EL and OSC 8 rare terminated by ST. Sharing this lexer
// keeps text extraction and history allocation sizing independent of URLs.
func walkHistoryText(line []byte, text func([]byte)) bool {
	return walkHistory(line, text, nil, nil)
}

// walkHistory shares the formatter grammar across text, replay and snapshot
// validation. A nonnil holes callback enables the sparse history extension:
// NUL, a canonical positive uvarint, then exactly that many ASCII spaces.
// The callback receives the tag start, spaces start and spaces end offsets.
// NUL inside an OSC payload is ordinary payload, never a sparse tag.
func walkHistory(line []byte, text func([]byte), holes func(start, spaces, end int), rows func(start, end int, flags rowFlags)) bool {
	for i := 0; i < len(line); {
		start := i
		for i < len(line) && line[i] != 0x1b && line[i] != 0 {
			if i+8 <= len(line) && printable8(binary.LittleEndian.Uint64(line[i:])) {
				i += 8
				continue
			}
			if c := line[i]; c >= 0x20 && c < 0x7f {
				i++
				continue
			}
			r, n := utf8.DecodeRune(line[i:])
			if n == 1 || r < 0xa0 {
				return false
			}
			i += n
		}
		if text != nil && start < i {
			text(line[start:i])
		}
		if i == len(line) {
			return true
		}
		if line[i] == 0 {
			n, size := binary.Uvarint(line[i+1:])
			if n == 0 {
				// A zero count is a typed row annotation, never an empty hole.
				// The kind is 0 none,1 primary prompt,2 continuation.
				if rows == nil || size != 1 || i+4 > len(line) || line[i+2] != 'P' || line[i+3] > 2 {
					return false
				}
				rows(i, i+4, rowFlags(line[i+3])<<1)
				i += 4
				continue
			}
			if holes == nil || size <= 0 || n > uint64(len(line)-i-1-size) || size != (bits.Len64(n)+6)/7 {
				return false
			}
			spaces, end := i+1+size, i+1+size+int(n)
			for _, c := range line[spaces:end] {
				if c != ' ' {
					return false
				}
			}
			holes(i, spaces, end)
			if text != nil {
				text(line[spaces:end])
			}
			i = end
			continue
		}
		if i+2 > len(line) {
			return false
		}
		switch line[i+1] {
		case '[':
			body := line[i+2:]
			switch {
			case len(body) >= 6 && bytes.Equal(body[:5], []byte("?2027")) && (body[5] == 'h' || body[5] == 'l'):
				i += 8
			case len(body) >= 1 && body[0] == 'K':
				i += 3
			case len(body) >= 3 && (body[0] == '0' || body[0] == '1') && body[1] == '"' && body[2] == 'q':
				i += 5
			default:
				j := i + 2
				for j < len(line) && (line[j] >= '0' && line[j] <= '9' || line[j] == ';' || line[j] == ':') {
					j++
				}
				if j == len(line) || line[j] != 'm' {
					return false
				}
				i = j + 1
			}
		case ']':
			if !bytes.HasPrefix(line[i+2:], []byte("8;")) {
				return false
			}
			payload := line[i+4:]
			end := bytes.IndexByte(payload, 0x1b)
			if end < 0 || end+1 >= len(payload) || payload[end+1] != '\\' {
				return false
			}
			payload = payload[:end]
			if bytes.IndexByte(payload, ';') < 0 || bytes.IndexByte(payload, 0x07) >= 0 ||
				bytes.IndexByte(payload, 0x18) >= 0 || bytes.IndexByte(payload, 0x1a) >= 0 {
				return false
			}
			i += end + 6
		default:
			return false
		}
	}
	return true
}
