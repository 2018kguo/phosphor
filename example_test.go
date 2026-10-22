package phosphor_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/2018kguo/phosphor"
)

func Example() {
	t := phosphor.New(20, 3)
	fmt.Fprint(t, "hello, \x1b[1;31mworld\x1b[m\r\n")
	fmt.Fprint(t, "a line long enough to wrap")

	for _, line := range t.Lines() {
		fmt.Printf("%q\n", line)
	}
	c := t.Cell(7, 0)
	fmt.Println(c.Text(), c.Style.Fg == phosphor.IndexedColor(1), c.Style.Attrs&phosphor.AttrBold != 0)
	cur := t.Cursor()
	fmt.Println("cursor", cur.X, cur.Y, "row 2 wrapped:", t.Wrapped(2))
	// Output:
	// "hello, world"
	// "a line long enough t"
	// "o wrap"
	// w true true
	// cursor 6 2 row 2 wrapped: true
}

// A headless runner answers the queries a program sends, or the program
// may wait forever for the reply. Pass the pseudo-terminal's input side
// to WithReply; here the program's output is simulated and the replies
// collected.
func Example_headless() {
	var toProgram bytes.Buffer // in a real runner: the pty master, e.g. from github.com/creack/pty
	t := phosphor.New(80, 24, phosphor.WithReply(&toProgram))

	// A program checking what it runs in: primary device attributes, the
	// cursor position, the background color, and kitty keyboard support.
	io.WriteString(t, "\x1b[5;10H\x1b[c\x1b[6n\x1b]11;?\x07\x1b[?u")

	fmt.Printf("%q\n", toProgram.String())
	// Output:
	// "\x1b[?62;22c\x1b[5;10R\x1b]11;rgb:0000/0000/0000\a\x1b[?0u"
}

// A renderer redraws only the rows that changed since its last frame.
func Example_renderer() {
	t := phosphor.New(12, 4)
	pal := t.Palette()
	draw := func() {
		var drawn []string
		_, rows := t.Size()
		for y := range rows {
			if !t.RowDirty(y) {
				continue
			}
			var line strings.Builder
			cols, _ := t.Size()
			for x := range cols {
				c := t.Cell(x, y)
				if c.Width == 0 {
					continue // the right half of a wide character
				}
				_ = pal.Resolve(c.Style.Fg, false) // the color to draw it in
				if c.Rune == 0 {                   // never written, or erased
					line.WriteByte('.')
				} else {
					line.WriteString(c.Text())
				}
			}
			drawn = append(drawn, fmt.Sprintf("%d:%s", y, line.String()))
		}
		t.ClearDirty()
		fmt.Println(strings.Join(drawn, " "))
	}

	io.WriteString(t, "$ ls\r\n")
	draw() // the first frame draws everything
	io.WriteString(t, "a.txt b.go\r\n$ ")
	draw()
	io.WriteString(t, "\x1b[1;3H\x1b[K")
	draw()
	// Output:
	// 0:$ ls........ 1:............ 2:............ 3:............
	// 1:a.txt b.go.. 2:$ ..........
	// 0:$ ..........
}

func ExampleTerminal_Repaint() {
	src := phosphor.New(30, 3)
	io.WriteString(src, "\x1b]2;build\x07\x1b[32mok\x1b[m 12 tests\r\n\x1b[?2004h")

	// A client attaching later (or a restore from a checkpoint) replays
	// Repaint on a fresh terminal of the same size.
	dst := phosphor.New(30, 3)
	dst.Write(src.Repaint(true))
	fmt.Printf("%q %q %v %v\n", dst.Line(0), dst.Title(), dst.Modes().BracketedPaste, dst.Cell(0, 0).Style == src.Cell(0, 0).Style)
	// Output:
	// "ok 12 tests" "build" true true
}

func ExampleTerminal_Restart() {
	t := phosphor.New(20, 4)
	io.WriteString(t, "$ make\r\nerror: oops\r\n\x1b[?1049h\x1b[2J\x1b[Hfull-screen app")

	// The program exited and is started again: keep its last screen above.
	t.Restart()
	io.WriteString(t, "$ ")
	for _, line := range t.Lines() {
		fmt.Printf("%q\n", line)
	}
	// Output:
	// "$ make"
	// "error: oops"
	// "full-screen app"
	// "$"
}
