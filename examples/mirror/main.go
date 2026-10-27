// Command mirror runs a program in a PTY, feeds its output to phosphor,
// and draws the screen back from phosphor's cells alone: every character,
// color and attribute you see comes from Terminal.Cell. The gutter on the
// left lights up for the rows phosphor reports dirty, and fades.
//
//	go run ./mirror vim README.md
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"

	"github.com/2018kguo/phosphor"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: mirror command [args...]")
		os.Exit(2)
	}
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w, h = 100, 30
	}
	cols, rows := w-2, h-2 // a gutter column on the left, a status line above and below

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	old, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err == nil {
		defer term.Restore(int(os.Stdin.Fd()), old)
	}
	go io.Copy(ptmx, os.Stdin)

	var mu sync.Mutex
	t := phosphor.New(cols, rows, phosphor.WithReply(ptmx))
	var bytesIn int
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 32<<10)
		for {
			n, err := ptmx.Read(buf)
			mu.Lock()
			t.Write(buf[:n])
			bytesIn += n
			mu.Unlock()
			if err != nil {
				close(done)
				return
			}
		}
	}()

	out := []byte("\x1b[?1049h\x1b[?25l\x1b[2J")
	heat := make([]int, rows) // frames left in each row's gutter glow
	tick := time.NewTicker(time.Second / 30)
	defer tick.Stop()
	for {
		select {
		case <-done:
			os.Stdout.WriteString("\x1b[0m\x1b[?25h\x1b[?1049l")
			return
		case <-tick.C:
		}
		mu.Lock()
		redrawn := 0
		for y := range rows {
			if t.RowDirty(y) {
				heat[y] = 8
				redrawn++
				out = appendRow(append(out, cup(y+2, 3)...), t, y, cols)
			}
			if heat[y] >= 0 {
				out = append(append(out, cup(y+2, 1)...), gutter(heat[y])...)
				heat[y]--
			}
		}
		t.ClearDirty()
		cur, title := t.Cursor(), t.Title()
		mode := "main screen"
		if t.Modes().AltScreen {
			mode = "alt screen"
		}
		mu.Unlock()

		out = append(out, cup(1, 1)...)
		out = append(out, fmt.Sprintf("\x1b[0;1;38;2;57;135;229m phosphor\x1b[0;2m  %dx%d  %s\x1b[K", cols, rows, title)...)
		out = append(out, cup(h, 1)...)
		out = append(out, fmt.Sprintf("\x1b[0;2m cursor %d,%d · %s · %2d rows redrawn · %d KiB parsed\x1b[K",
			cur.X, cur.Y, mode, redrawn, bytesIn>>10)...)
		if cur.Visible {
			out = append(append(out, cup(cur.Y+2, cur.X+3)...), "\x1b[?25h"...)
		} else {
			out = append(out, "\x1b[?25l"...)
		}
		os.Stdout.Write(out)
		out = out[:0]
	}
}

func cup(row, col int) string { return "\x1b[" + strconv.Itoa(row) + ";" + strconv.Itoa(col) + "H" }

// gutter draws a row's glow: bright when it just changed, fading out.
func gutter(heat int) string {
	if heat <= 0 {
		return "\x1b[0m  "
	}
	return fmt.Sprintf("\x1b[0;38;2;%d;%d;%dm█ ", 20+heat*5, 60+heat*10, 90+heat*20)
}

// appendRow draws row y from phosphor's cells, one SGR per style change.
func appendRow(b []byte, t *phosphor.Terminal, y, cols int) []byte {
	var last phosphor.Style
	b = append(b, "\x1b[0m"...)
	for x := 0; x < cols; x++ {
		c := t.Cell(x, y)
		if c.Width == 0 && c.Rune == 0 && x > 0 && t.Cell(x-1, y).Width == 2 {
			continue // right half of a wide character
		}
		if c.Style != last {
			b = appendSGR(b, c.Style)
			last = c.Style
		}
		if c.Rune == 0 {
			b = append(b, ' ')
		} else {
			b = c.AppendText(b)
		}
	}
	return b
}

func appendSGR(b []byte, s phosphor.Style) []byte {
	b = append(b, "\x1b[0"...)
	for _, a := range []struct {
		attr phosphor.Attr
		code string
	}{
		{phosphor.AttrBold, ";1"}, {phosphor.AttrFaint, ";2"}, {phosphor.AttrItalic, ";3"},
		{phosphor.AttrBlink, ";5"}, {phosphor.AttrInverse, ";7"}, {phosphor.AttrHidden, ";8"},
		{phosphor.AttrStrikethrough, ";9"},
	} {
		if s.Attrs&a.attr != 0 {
			b = append(b, a.code...)
		}
	}
	if s.Underline != phosphor.UnderlineNone {
		b = append(b, ";4"...)
	}
	b = appendColor(b, s.Fg, "38")
	b = appendColor(b, s.Bg, "48")
	b = appendColor(b, s.UnderlineColor, "58")
	return append(b, 'm')
}

func appendColor(b []byte, c phosphor.Color, base string) []byte {
	switch c.Kind() {
	case phosphor.ColorIndexed:
		return fmt.Appendf(b, ";%s;5;%d", base, c.Index())
	case phosphor.ColorRGB:
		rgb := c.RGB()
		return fmt.Appendf(b, ";%s;2;%d;%d;%d", base, rgb.R, rgb.G, rgb.B)
	}
	return b
}
