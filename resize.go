package phosphor

import (
	"bytes"
	"unicode/utf8"
)

// point is a cursor position carried through a resize.
type point struct {
	x, y    int
	pending bool
}

// Resize changes the screen size to cols x rows (each at least 1). On the
// main screen, lines that soft wrapped are rewrapped to the new width
// (reflow); when the content needs more rows than the screen has, blank
// rows below the cursor go first, then rows from the top go to the
// scrollback, and when it needs fewer, lines come back from the
// scrollback. The alternate screen is not reflowed: its rows are cut or
// padded (full-screen programs redraw when told of the new size). The
// cursor and saved cursors follow the characters they were on. The scroll
// region and margins are reset, and every row is marked dirty.
//
// Resize does not tell the program: the caller resizes the pseudo-terminal
// too (which sends it SIGWINCH).
func (t *Terminal) Resize(cols, rows int) {
	t.sync = false
	cols, rows = max(cols, 1), max(rows, 1)
	if cols == t.cols && rows == t.rows {
		return
	}
	defer t.resizeDirty()
	oldCols := t.cols
	oldRows := t.rows
	primary := point{t.cur.x, t.cur.y, t.cur.wrapPending}
	if t.altActive {
		primary = t.ext().inactivePrimary
	}
	// Native height-only growth pulls history only when the original
	// primary cursor was on the bottom row, including while it is inactive.
	pullHistory := cols != oldCols || rows <= oldRows || primary.y >= oldRows-1
	if cols != oldCols {
		t.dropLineErases()
	}

	// The positions on each screen.
	var mainPts, altPts []*cursor
	if t.saved != nil {
		if t.saved.screen[0].valid {
			mainPts = append(mainPts, &t.saved.screen[0].cursor)
		}
		if t.saved.screen[1].valid {
			altPts = append(altPts, &t.saved.screen[1].cursor)
		}
	}
	if t.altActive {
		if t.saved != nil && t.saved.alternate.valid {
			mainPts = append(mainPts, &t.saved.alternate.cursor)
		}
		altPts = append(altPts, &t.cur)
	} else {
		mainPts = append(mainPts, &t.cur)
	}

	// Main screen: reflow, then fit to the height.
	pts := make([]point, len(mainPts), len(mainPts)+1)
	for i, c := range mainPts {
		pts[i] = point{c.x, c.y, c.wrapPending}
	}
	if t.altActive {
		pts = append(pts, primary) // actual screen cursor is last, independently of saves
	}
	scr, wrap := t.main, t.mainWrap
	if cols != oldCols && !t.autowrap {
		// Nothing rewraps: the scrollback's last line ends where it is.
		t.sbOpen = false
	}
	if cols != oldCols && t.autowrap && t.sbOpen {
		// The line the screen starts with began in the scrollback: take it
		// back so it is rewrapped whole.
		back, backWrap := t.popLine()
		if wrap == nil {
			wrap = make([]rowFlags, len(scr))
		}
		scr = append(back, scr...)
		wrap = append(backWrap, wrap...)
		for i := range pts {
			pts[i].y += len(back)
		}
	}
	t.cols, t.rows = cols, rows
	switch {
	case cols != oldCols && t.autowrap:
		scr, wrap = t.rewrap(scr, wrap, oldCols, cols, pts, 0, nil)
	case cols != oldCols:
		// Without autowrap nothing wraps, so rows are cut instead.
		for y, row := range scr {
			scr[y] = t.resizeRow(row)
		}
		for i := range wrap {
			wrap[i] &^= rowWrap
		}
		for i := range pts {
			pts[i] = t.resizeUnflowedPoint(pts[i])
		}
	}
	if wrap == nil {
		wrap = make([]rowFlags, len(scr))
	}
	cy := 0
	if len(pts) > 0 {
		cy = pts[len(pts)-1].y // the cursor, or the main screen's saved one
	}
	// Blank rows below the cursor go first.
	for len(scr) > rows && len(scr)-1 > cy && zeroRow(scr[len(scr)-1]) {
		scr, wrap = scr[:len(scr)-1], wrap[:len(wrap)-1]
	}
	if extra := len(scr) - rows; extra > 0 {
		for i, row := range scr[:extra] {
			t.pushScrollback(row, wrap[i], rowAt(scr, i+1))
		}
		scr, wrap = scr[extra:], wrap[extra:]
		for i := range pts {
			pts[i].y -= extra
		}
	} else if extra < 0 && pullHistory {
		back, backWrap := t.popScrollback(-extra)
		scr = append(back, scr...)
		wrap = append(backWrap, wrap...)
		for i := range pts {
			pts[i].y += len(back)
		}
	}
	t.main, t.mainWrap = fitRows(scr, wrap, rows)
	for i, c := range mainPts {
		t.place(c, pts[i])
	}
	if t.altActive {
		p := pts[len(pts)-1]
		var c cursor
		t.place(&c, p)
		t.x.inactivePrimary = point{c.x, c.y, c.wrapPending}
	}

	// The alternate screen (also when kept by mode 47): truncated or
	// padded.
	if t.alt != nil {
		for y, row := range t.alt {
			t.alt[y] = t.resizeRow(row)
		}
		altY := t.cur.y
		if !t.altActive {
			altY = 0
		}
		scr, wrap := t.alt, t.altWrap
		if wrap == nil {
			wrap = make([]rowFlags, len(scr))
		}
		for len(scr) > rows && len(scr)-1 > altY && zeroRow(scr[len(scr)-1]) {
			scr, wrap = scr[:len(scr)-1], wrap[:len(wrap)-1]
		}
		extra := max(len(scr)-rows, 0)
		scr, wrap = scr[extra:], wrap[extra:]
		t.alt, t.altWrap = fitRows(scr, wrap, rows)
		for _, c := range altPts {
			p := t.resizeUnflowedPoint(point{c.x, c.y - extra, c.wrapPending})
			t.place(c, p)
		}
	}
	if !t.altActive && t.alt != nil && t.altBlank() {
		t.alt, t.altWrap = nil, nil // a kept alternate screen cut to nothing
	}
	if t.altWrap != nil && !t.anyRowFlags(t.altWrap) {
		t.altWrap = nil
	}
	if t.mainWrap != nil && !t.anyRowFlags(t.mainWrap) {
		t.mainWrap = nil
	}
	t.top, t.bot, t.left, t.right = 0, rows-1, 0, cols-1
	// Saved cursors of a screen not in use (the alternate one's DECSC
	// slot) are just kept in bounds.
	if t.saved != nil {
		for _, c := range []*cursor{&t.saved.screen[0].cursor, &t.saved.screen[1].cursor, &t.saved.alternate.cursor} {
			t.place(c, point{c.x, c.y, c.wrapPending})
		}
	}
	t.resizeTabs(oldCols)
	t.clearPromptForRedraw()
	// Imported history and rewrapped rows are now visible to the pool
	// collectors. Reclaim detached entries here too: a window can resize
	// repeatedly while its program prints nothing.
	t.compactPools()
}

func (t *Terminal) anyRowFlags(w []rowFlags) bool {
	for _, b := range w {
		if b != 0 {
			return true
		}
	}
	return false
}

// place moves c to p, clamped to the new size.
func (t *Terminal) place(c *cursor, p point) {
	c.x = min(max(p.x, 0), t.cols-1)
	c.y = min(max(p.y, 0), t.rows-1)
	c.wrapPending = p.pending && c.x == t.cols-1
}

func (t *Terminal) resizeUnflowedPoint(p point) point {
	p.x = min(max(p.x, 0), t.cols-1)
	if p.pending && p.x != t.cols-1 {
		p.x++
		p.pending = false
	}
	return p
}

// fitRows pads or cuts scr and wrap to exactly rows entries. The wrap
// flags are dropped when none is set.
func fitRows(scr [][]cell, wrap []rowFlags, rows int) ([][]cell, []rowFlags) {
	out := make([][]cell, rows)
	copy(out, scr)
	var w []rowFlags
	for i, b := range wrap[:min(len(wrap), rows)] {
		if i == rows-1 {
			b &^= rowWrap
		}
		if b != 0 {
			if w == nil {
				w = make([]rowFlags, rows)
			}
			w[i] = b
		}
	}
	return out, w
}

// resizeRow drops cells beyond the new width. An absent tail stays zero.
func (t *Terminal) resizeRow(row []cell) []cell {
	switch {
	case row == nil:
		return nil
	case t.cols < len(row):
		row = row[:t.cols]
		t.fixWide(row)
	}
	return row
}

// rewrap reflows scr (oldCols wide, wrap flags as in t.mainWrap) to cols
// columns, returning the new rows and their wrap flags (one per row).
// Points on the screen are moved with the characters they were on.
func (t *Terminal) rewrap(scr [][]cell, wrap []rowFlags, oldCols, cols int, pts []point, extent int, markers []rowMark) ([][]cell, []rowFlags) {
	var out [][]cell
	var outWrap []rowFlags
	var cells []cell
	var starts []int // offset in cells of each row of the line
	orig := append([]point(nil), pts...)
	for y := 0; y < len(scr); {
		// The logical line is rows y..e.
		e := y
		for e < len(scr)-1 && wrap != nil && wrap[e]&rowWrap != 0 {
			e++
		}
		cells, starts = cells[:0], starts[:0]
		for r := y; r <= e; r++ {
			starts = append(starts, len(cells))
			row := scr[r]
			n := oldCols
			// A blank left at the end of a row because the wide character
			// after it did not fit is not part of the line.
			if r < e && rowCell(row, n-1).R == spacerHead {
				n--
			}
			start := len(cells)
			cells = append(cells, make([]cell, n)...)
			copy(cells[start:], row[:min(n, len(row))])
		}
		length := len(cells)
		// History's explicitly serialized trailing holes remain part of the
		// line. Screen reflow passes zero and trims its implicit padding.
		for length > extent && isZero(cells[length-1]) {
			length--
		}
		// Native keeps one cell for an empty annotated prompt row.
		for r := y; r <= e; r++ {
			if rowFlagsAt(wrap, r)&rowSemanticMask != 0 {
				length = max(length, min(starts[r-y]+1, len(cells)))
			}
		}
		// On a nonempty logical line the live cursor's blank also belongs
		// to reflow. A wholly blank line keeps the native clamp behavior.
		if extent == 0 && length > 0 && len(orig) > 0 {
			p := orig[len(orig)-1]
			if !p.pending && p.x > 0 && p.y >= y && p.y <= e {
				off := starts[p.y-y] + p.x
				length = max(length, min(off+1, len(cells)))
			}
		}
		extent = 0
		// Offsets of the points on this line.
		type pref struct {
			i   int
			off int
		}
		var mine []pref
		for i, p := range orig {
			if p.y >= y && p.y <= e {
				off := starts[p.y-y] + p.x
				if p.pending {
					off++
				}
				mine = append(mine, pref{i, off})
			}
		}
		// Lay the cells out in rows of cols.
		row := make([]cell, cols)
		col := 0
		placed := make(map[int][2]int, len(mine))
		sourceRow, marker := 0, 0
		flags := rowFlagsAt(wrap, y) & rowSemanticMask
		for i := 0; i < length; i++ {
			nextFlags := flags
			for sourceRow+1 < len(starts) && i >= starts[sourceRow+1] {
				sourceRow++
				nextFlags = rowFlagsAt(wrap, y+sourceRow) & rowSemanticMask
			}
			for marker < len(markers) && markers[marker].offset <= i {
				nextFlags = markers[marker].flags
				marker++
			}
			c := cells[i]
			if c.R == wideTail {
				continue
			}
			w := 1
			if i+1 < len(cells) && cells[i+1].R == wideTail {
				w = 2
			}
			if w > cols {
				// Unlike a glyph printed into an already narrow terminal,
				// native reflow destroys a clipped wide glyph's whole cell.
				c, w = cell{}, 1
			}
			if col+w > cols {
				if col == cols-1 && w == 2 {
					row[col] = cell{R: spacerHead, Bg: c.Bg}
				}
				out = append(out, row)
				outWrap = append(outWrap, flags|rowWrap)
				row = make([]cell, cols)
				col = 0
			}
			flags = nextFlags
			for _, m := range mine {
				if m.off == i || (w == 2 && m.off == i+1) {
					placed[m.i] = [2]int{len(out), col + m.off - i}
				}
			}
			row[col] = c
			if w == 2 {
				row[col+1] = cells[i+1]
			}
			col += w
		}
		if length == 0 && scr[y] == nil && y == e {
			row = nil
		}
		out = append(out, row)
		outWrap = append(outWrap, flags)
		last := len(out) - 1
		for _, m := range mine {
			p := &pts[m.i]
			if at, ok := placed[m.i]; ok && m.off < length {
				p.x, p.y, p.pending = at[1], at[0], false
				continue
			}
			// After the end of the line.
			x := col + m.off - length
			p.y, p.pending = last, false
			switch {
			case x < cols:
				p.x = x
			case x == cols && m.off == length:
				p.x, p.pending = cols-1, true
			default:
				p.x = cols - 1
			}
		}
		y = e + 1
	}
	return out, outWrap
}

// dropLineErases removes the erases (EL) after wrapped rows in the
// scrollback. They clear what a wrap may scroll in when a line is replayed
// at the width it was written at; at any other width one could land on a
// pending wrap and erase the last character.
func (t *Terminal) dropLineErases() {
	for i, line := range t.sb[t.sbStart:] {
		if bytes.Contains(line, eraseLine) {
			n := len(line)
			line = bytes.ReplaceAll(line, eraseLine, nil)
			t.sb[t.sbStart+i] = line
			t.sbBytes -= n - len(line)
		}
	}
}

var eraseLine = []byte("\x1b[K")

// popScrollback takes up to n rows from the end of the scrollback,
// rewrapped to the current width, for the top of the screen. A line that
// does not fit whole leaves its beginning in the scrollback.
func (t *Terminal) popScrollback(n int) ([][]cell, []rowFlags) {
	var rows [][]cell
	var wraps []rowFlags
	for len(rows) < n && t.ScrollbackLen() > 0 {
		lr, lw := t.popLine()
		if need := n - len(rows); len(lr) > need {
			head := len(lr) - need
			for i, r := range lr[:head] {
				t.pushScrollback(r, rowFlagsAt(lw, i)|rowWrap, lr[i+1])
			}
			lr, lw = lr[head:], lw[head:]
		}
		rows = append(lr, rows...)
		wraps = append(lw, wraps...)
	}
	return rows, wraps
}

// popLine removes the last scrollback line and returns it as rows of the
// current width, with their wrap flags (the last row's says whether the
// line went on onto the screen).
func (t *Terminal) popLine() ([][]cell, []rowFlags) {
	last := len(t.sb) - 1
	line := t.sb[last]
	open, spacer := t.sbOpen, t.sbSpacer
	t.sb[last] = nil
	t.sb = t.sb[:last]
	t.sbBytes -= len(line) + 24
	t.sbOpen = false
	if t.sbStart > len(t.sb) {
		t.sbStart = len(t.sb)
	}
	rows, wraps := t.parseLine(line)
	if open {
		wraps[len(wraps)-1] |= rowWrap
	} else {
		wraps[len(wraps)-1] &^= rowWrap
	}
	if r := rows[len(rows)-1]; open && spacer && r != nil && r[t.cols-1] == (cell{}) {
		r[t.cols-1].R = spacerHead // left out of the line, see pushScrollback
	}
	return rows, wraps
}

// parseLine turns a serialized scrollback line back into rows of the
// current width, with their wrap flags.
func (t *Terminal) parseLine(line []byte) ([][]cell, []rowFlags) {
	tmp := replayLine(line, t.cols)
	k := tmp.cur.y + 1
	if tmp.x != nil {
		clusterRefs, extraRefs := 0, 0
		for _, row := range tmp.main[:k] {
			for _, c := range row {
				if c.R >= clusterBase {
					clusterRefs++
				}
				if c.E != 0 {
					extraRefs++
				}
			}
		}
		if clusterRefs > 0 || extraRefs > 0 {
			x := t.ext()
			clusterCap, rareCap := x.clusterCap, x.rareCap
			// Imported cells, earlier history batches and rewrapped rows
			// are detached until Resize installs the finished screen. No
			// receiver collection may run while any of them are detached.
			// Bound by references, since recent-only interning may append
			// a repeated entry more than once during this batch.
			x.clusterCap = max(clusterCap, len(x.clusters)+clusterRefs)
			x.rareCap = max(rareCap, len(x.rare)+extraRefs)
			defer func() {
				// Resize collects only after all working rows are attached.
				x.clusterCap, x.rareCap = clusterCap, rareCap
			}()
		}
	}
	rows := make([][]cell, k)
	wraps := make([]rowFlags, k)
	for y := range k {
		wraps[y] = rowFlagsAt(tmp.mainWrap, y)
		row := tmp.main[y]
		if row == nil {
			continue
		}
		for x := range row {
			if r := row[x].R; r >= clusterBase {
				row[x].R = t.intern(tmp.x.clusters[r-clusterBase])
			}
			if l := row[x].E; l != 0 {
				row[x].E = t.internExtra(tmp.extraAt(l))
			}
		}
		rows[y] = row
	}
	return rows, wraps
}

// replayLine first decodes clustered history away from a physical edge.
// Otherwise an emoji base can wrap before its VS15 narrows it, even though
// the completed cell fits. Reflow then uses the stored cell widths. Ordinary
// history keeps its destination-width path. Text sizing shares the history
// lexer and excludes CSI/OSC payloads: a large URL cannot allocate a wide row.
func replayLine(line []byte, cols int) *Terminal {
	cells := 0
	sparse := false
	walkHistory(line, func(text []byte) {
		for len(text) > 0 {
			r, n := utf8.DecodeRune(text)
			text = text[n:]
			// A zero-width selector may widen the preceding cell by one;
			// counting one for each mark is a safe bound for mixed modes.
			cells += max(1, CodepointWidth(r))
		}
	}, func(start, spaces, end int) { sparse = true }, func(start, end int, flags rowFlags) {
		sparse = true
		if flags != 0 {
			cells++
		} // space for a meaningful empty annotated row
	})
	if !sparse && !bytes.Contains(line, []byte("\x1b[?2027h")) {
		tmp := newTerminal(cols, cells/cols+2, 0, 0)
		tmp.Write(line)
		return tmp
	}
	wide := cells + 1
	tmp := newTerminal(wide, 1, 0, 0)
	var markers []rowMark
	finishEmptyMarker := func() {
		if len(markers) > 0 && markers[len(markers)-1].flags != 0 && markers[len(markers)-1].offset == tmp.cur.x {
			x := tmp.cur.x
			tmp.Write([]byte{' '})
			tmp.main[0][x] = cell{}
		}
	}
	if !sparse {
		tmp.Write(line)
	} else {
		start := 0
		walkHistory(line, nil, func(tag, spaces, end int) {
			tmp.Write(line[start:tag])
			x := tmp.cur.x
			tmp.Write(line[spaces:end])
			for i := x; i < x+end-spaces; i++ {
				tmp.main[0][i].R = 0
			}
			start = end
		}, func(tag, end int, flags rowFlags) {
			tmp.Write(line[start:tag])
			finishEmptyMarker()
			markers = append(markers, rowMark{offset: tmp.cur.x, flags: flags})
			start = end
		})
		tmp.Write(line[start:])
		finishEmptyMarker()
	}
	if tmp.main[0] != nil {
		// A narrowing selector may leave a styled old tail beyond the
		// formatter's final column; it is not serialized content.
		tmp.main[0] = tmp.main[0][:tmp.cur.x]
	}
	position := []point{{x: tmp.cur.x}}
	tmp.main, tmp.mainWrap = tmp.rewrap(tmp.main, nil, wide, cols, position, tmp.cur.x, markers)
	tmp.cols, tmp.rows, tmp.right = cols, len(tmp.main), cols-1
	tmp.cur.x, tmp.cur.y, tmp.cur.wrapPending = position[0].x, position[0].y, position[0].pending
	return tmp
}

// Restart prepares the terminal for a new program after the previous one
// ended, as a terminal that relaunches its command keeps the old output in
// view: it resets everything as RIS (ESC c) does, keeps the scrollback,
// and leaves what the screen showed above the cursor, as if it had been
// printed. That is the main screen down to its last non-blank row, then,
// if a full-screen program was showing the alternate screen, that too.
// Whatever does not fit above the last row scrolls into the scrollback,
// and the cursor starts the row below the kept content.
//
// Options given to New (reply writer, bell, palette) are kept; colors set
// by the program (OSC 4, 10, 11, 12) are reset.
func (t *Terminal) Restart() {
	var old [][]cell
	var oldWrap []rowFlags
	keep := func(scr [][]cell, wrap []rowFlags) {
		last := -1
		for y, row := range scr {
			if !rowBlank(row) {
				last = y
			}
		}
		old = append(old, scr[:last+1]...)
		for y := range last + 1 {
			flags := rowFlagsAt(wrap, y)
			if y == last {
				flags &^= rowWrap
			}
			oldWrap = append(oldWrap, flags)
		}
	}
	keep(t.main, t.mainWrap)
	if t.altActive {
		keep(t.alt, t.altWrap)
	}
	sbOpen, x := t.sbOpen, t.x
	t.reset()
	t.sbOpen = sbOpen
	if x != nil && (x.clusters != nil || x.rare != nil) {
		// The kept rows may refer to characters with combining marks and
		// to rare cell attributes.
		t.x = &extra{clusters: x.clusters, clusterCap: x.clusterCap, rare: x.rare, rareCap: x.rareCap}
	}
	if over := len(old) - (t.rows - 1); over > 0 {
		for i, row := range old[:over] {
			t.pushScrollback(row, oldWrap[i], rowAt(old, i+1))
		}
		old, oldWrap = old[over:], oldWrap[over:]
	}
	copy(t.main, old)
	for y, w := range oldWrap {
		if w != 0 {
			t.setRowFlags(y, w)
		}
	}
	t.cur.y = len(old)
}
