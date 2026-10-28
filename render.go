package phosphor

import (
	"io"
	"slices"
)

// AppendRowANSI appends the visible row y as ANSI, preserving Unicode cluster
// join modes, colors, attributes and links. Invalid rows append nothing.
// The caller positions the cursor at column 1 on a terminal of the same width.
// This changes SGR, the cursor save slot, autowrap and mode 2027; it does not
// position the row vertically or change the host palette. Program-changed
// colors become RGB; unchanged colors remain relative to the host's palette.
// A renderer must restore the desired cursor and input modes after painting.
func (t *Terminal) AppendRowANSI(dst []byte, y int) []byte {
	if uint(y) >= uint(t.rows) {
		return dst
	}
	return t.appendDisplayRow(dst, t.screen()[y])
}
func (t *Terminal) appendDisplayRow(dst []byte, row []cell) []byte {
	paint := *t
	current := t.Palette()
	base := defaultPalette
	if t.cfg != nil && t.cfg.palette != nil {
		base = *t.cfg.palette
	}
	color := func(c Color, bg bool) Color {
		if c.Kind() == ColorRGB {
			return c
		}
		old, next := base.Resolve(c, bg), current.Resolve(c, bg)
		if old != next {
			return RGBColor(next.R, next.G, next.B)
		}
		return c
	}
	if current != base || t.reverseVideo {
		size := len(row)
		if t.reverseVideo || base.Foreground != current.Foreground || base.Background != current.Background {
			size = t.cols
		}
		projected := make([]cell, size)
		copy(projected, row)
		row = projected
		for i := range row {
			row[i].Fg = color(row[i].Fg, false)
			row[i].Bg = color(row[i].Bg, true)
			if t.reverseVideo {
				row[i].A ^= AttrInverse
			}
		}
		if t.x != nil {
			extra := *t.x
			for i, v := range extra.rare {
				if c := color(v.underline, false); v.underline != 0 && c != v.underline {
					if len(extra.rare) > 0 && &extra.rare[0] == &t.x.rare[0] {
						extra.rare = slices.Clone(extra.rare)
					}
					extra.rare[i].underline = c
				}
			}
			paint.x = &extra
		}
	}
	dst = append(dst, "\x1b[0m\x1b[0\"q\x1b]8;;\x1b\\\x1b[?2027l\x1b[?7h"...)
	saved := false
	dst = paint.appendRepaintRow(dst, row, t.cols, false, &saved)
	return append(dst, "\x1b[?7l"...)
}

// WriteScrollback writes retained history to w, one ANSI-encoded physical row
// per Write call, using the same row format as AppendRowANSI. The writer must
// not retain the supplied bytes. It stops on the first error or short write.
func (t *Terminal) WriteScrollback(w io.Writer) error {
	var buf []byte
	for _, line := range t.sb[t.sbStart:] {
		tmp := replayLine(line, t.cols)
		tmp.reverseVideo = t.reverseVideo
		if t.cfg != nil {
			tmp.cfg = &config{palette: t.cfg.palette, colors: t.cfg.colors}
		}
		for y := 0; y <= tmp.cur.y; y++ {
			buf = tmp.appendDisplayRow(buf[:0], tmp.screen()[y])
			n, err := w.Write(buf)
			if err != nil {
				return err
			}
			if n != len(buf) {
				return io.ErrShortWrite
			}
		}
	}
	return nil
}

// Reflow moves existing rows between screen and history; these are not new
// program-output scroll events and must not duplicate a viewer's history.
func (t *Terminal) pauseScrollbackOutput() func() {
	if t.cfg == nil || t.cfg.scrollbackOutput == nil {
		return func() {}
	}
	writer := t.cfg.scrollbackOutput
	t.cfg.scrollbackOutput = nil
	return func() { t.cfg.scrollbackOutput = writer }
}
