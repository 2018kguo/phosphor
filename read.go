package phosphor

import (
	"strings"
	"unicode/utf8"
)

// Size returns the screen size.
func (t *Terminal) Size() (cols, rows int) { return t.cols, t.rows }

// Title returns the window title last set with OSC 0 or 2.
func (t *Terminal) Title() string { return t.title }

// Cursor returns the cursor's position, visibility and shape.
func (t *Terminal) Cursor() Cursor {
	return Cursor{X: t.cur.x, Y: t.cur.y, Visible: !t.cursorHidden, PendingWrap: t.cur.wrapPending,
		Shape: t.cursorStyle.shape, Blink: t.cursorStyle.blink}
}

// Cursor shape and blinking can be changed independently by DECSCUSR and
// DEC private mode 12. Keep their current values rather than the last command.
type cursorAppearance struct {
	shape CursorShape
	blink bool
}

func (c cursorAppearance) parameter() int {
	p := int(max(c.shape, CursorBlock)) * 2
	if c.blink {
		p--
	}
	return p
}

// Modes returns the modes the program has set.
func (t *Terminal) Modes() Modes {
	m := Modes{
		AltScreen:             t.altActive,
		ApplicationCursorKeys: t.appCursor,
		ApplicationKeypad:     t.appKeypad,
		BracketedPaste:        t.bracketed,
		FocusEvents:           t.focus,
		Mouse:                 MouseMode(mouseModeNumber(t.mouse, mouseTrackingNumbers)),
		MouseEncoding:         MouseEncoding(mouseModeNumber(t.mouseEnc, mouseEncodingNumbers)),
		MouseTrackingModes:    MouseTrackingModes(t.mouse & mouseModeBits),
		MouseEncodingModes:    MouseEncodingModes(t.mouseEnc & mouseModeBits),
		Autowrap:              t.autowrap,
		Insert:                t.insert,
		Newline:               t.newline,
		Origin:                t.cur.origin,
		ReverseVideo:          t.reverseVideo,
		SyncUpdate:            t.sync,
		GraphemeClustering:    t.grapheme,
		Columns132:            t.privateModes()&privateMode132 != 0,
		AllowColumnSwitching:  t.privateModes()&privateModeAllowColumns != 0,
		SmoothScrolling:       t.privateModes()&privateModeSlowScroll != 0,
	}
	if n := len(t.kitty); n > 0 {
		m.KittyKeyboard = t.kitty[n-1]
	}
	return m
}

// Cell returns the cell at column x, row y of the visible screen (the
// alternate screen while it is shown). It returns a zero Cell for a
// position outside the screen.
func (t *Terminal) Cell(x, y int) Cell {
	if uint(x) >= uint(t.cols) || uint(y) >= uint(t.rows) {
		return Cell{}
	}
	row := t.screen()[y]
	if row == nil {
		return Cell{Width: 1}
	}
	return t.publicCell(row, x)
}

// publicCell converts row[x] (of a row of t) into a Cell.
func (t *Terminal) publicCell(row []cell, x int) Cell {
	c := &row[x]
	out := Cell{Rune: c.R, Width: 1, Style: t.publicStyle(c)}
	switch r := c.R; {
	case r < utf8.RuneSelf && r >= 0:
	case r == wideTail:
		out.Rune, out.Width = 0, 0
	case r == spacerHead:
		out.Rune = 0
	default:
		if r >= clusterBase {
			s := t.x.clusters[r-clusterBase].text
			var n int
			out.Rune, n = utf8.DecodeRuneInString(s)
			out.Combining = s[n:]
		}
		if x+1 < len(row) && row[x+1].R == wideTail {
			out.Width = 2
		}
	}
	if c.E != 0 {
		out.Link = parseLink(t.link(c.E))
	}
	return out
}

// Wrapped reports whether row y of the visible screen continues the row
// above it, because the text there reached the last column and wrapped
// (row 0 of the main screen can continue the last scrollback line). Text
// selection and copying join such rows without a line break.
func (t *Terminal) Wrapped(y int) bool {
	if uint(y) >= uint(t.rows) {
		return false
	}
	return t.wrapped(y)
}

// Line returns row y of the visible screen as text, without trailing
// blanks ("" for a row out of range).
func (t *Terminal) Line(y int) string {
	if uint(y) >= uint(t.rows) {
		return ""
	}
	return t.rowText(t.screen()[y])
}

// Lines returns the visible screen as text, one string per row, without
// trailing blanks.
func (t *Terminal) Lines() []string {
	scr := t.screen()
	out := make([]string, len(scr))
	for y, row := range scr {
		out[y] = t.rowText(row)
	}
	return out
}

// ScrollbackLen returns the number of lines in the scrollback. A line that
// soft wrapped on screen is one line.
func (t *Terminal) ScrollbackLen() int { return len(t.sb) - t.sbStart }

// ScrollbackLine returns scrollback line i as text, without trailing
// blanks: 0 is the oldest line, ScrollbackLen()-1 the newest ("" for a
// line out of range).
func (t *Terminal) ScrollbackLine(i int) string {
	if uint(i) >= uint(t.ScrollbackLen()) {
		return ""
	}
	return strings.TrimRight(stripSGR(t.sb[t.sbStart+i]), " ")
}

// ScrollbackLines returns the scrollback as text, oldest line first.
func (t *Terminal) ScrollbackLines() []string {
	out := make([]string, 0, t.ScrollbackLen())
	for _, line := range t.sb[t.sbStart:] {
		out = append(out, strings.TrimRight(stripSGR(line), " "))
	}
	return out
}

// ScrollbackCells returns scrollback line i (see ScrollbackLine) with its
// styles and rare, laid out as the rows it takes at the current width:
// one row, or more for a line that soft wraps. Unlike the other accessors
// it allocates, about as much as replaying the line costs.
func (t *Terminal) ScrollbackCells(i int) [][]Cell {
	if uint(i) >= uint(t.ScrollbackLen()) {
		return nil
	}
	line := t.sb[t.sbStart+i]
	tmp := replayLine(line, t.cols)
	rows := make([][]Cell, tmp.cur.y+1)
	for y := range rows {
		rows[y] = make([]Cell, t.cols)
		for x := range rows[y] {
			rows[y][x] = tmp.Cell(x, y)
		}
	}
	return rows
}

// ClearScrollback drops the scrollback, as ED 3 (CSI 3 J) does.
func (t *Terminal) ClearScrollback() {
	t.sb, t.sbStart, t.sbBytes, t.sbOpen = nil, 0, 0, false
}

// Palette returns the colors in effect: the palette the terminal started
// with (DefaultPalette, or WithPalette's), with the changes the program
// made since (OSC 4, 10, 11 and 12).
func (t *Terminal) Palette() Palette {
	p := defaultPalette
	if t.cfg == nil {
		return p
	}
	if t.cfg.palette != nil {
		p = *t.cfg.palette
	}
	for _, s := range t.cfg.colors {
		*p.slot(s.i) = s.c
	}
	return p
}

// slot returns the palette entry for a colorSet index.
func (p *Palette) slot(i uint16) *RGB {
	switch i {
	case slotFg:
		return &p.Foreground
	case slotBg:
		return &p.Background
	case slotCursor:
		return &p.Cursor
	}
	return &p.Colors[i&0xff]
}
