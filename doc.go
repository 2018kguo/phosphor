// Package phosphor is a headless VT/xterm terminal emulator: a screen
// model you feed a program's output and read back what a terminal would
// show.
//
// It keeps the main and alternate screens, a bounded scrollback, the
// cursor, colors and attributes, hyperlinks, the scroll region and
// margins, tab stops, character sets, the title, and the modes a program
// switched on, following xterm closely (see the README for the conformance
// suites it passes). Resize reflows the main screen. Repaint exports ANSI
// output for attaching clients; Snapshot preserves exact emulator state
// for checkpoints, including unfinished input and saved cursors.
//
// It is built to be cheap when there are many terminals: a cell is 16
// bytes, rows are allocated only once drawn on, the alternate screen only
// while in use, scrollback is stored serialized under a byte budget, an
// idle 80x24 terminal holds about 3 KB, and output that redraws a screen
// does not allocate.
//
// # Use
//
//	t := phosphor.New(80, 24)
//	t.Write(output)          // io.Writer; sequences may be split across writes
//	t.Lines()                // the screen as text
//	t.Cell(x, y)             // one cell: Rune, Width, Style, Link
//	t.Cursor(); t.Modes(); t.Title()
//	t.Resize(100, 30)
//	t.Repaint(true)          // ANSI output for an attached terminal
//	t.Snapshot()             // exact checkpoint, scrollback included
//
// # Queries
//
// Programs ask their terminal things (device attributes, the cursor
// position, mode and color settings) and some wait for the answer. Pass
// WithReply with the pseudo-terminal's input side and the Terminal
// answers; without it queries are ignored, which is right when a real
// terminal attached downstream answers them instead.
//
// # Rendering
//
// RowDirty reports which rows changed since the last ClearDirty, so a
// renderer redraws only those. Tracking starts with the first ClearDirty
// and costs nothing before.
//
// # Concurrency
//
// A Terminal is not safe for concurrent use. Callers that write output in
// one goroutine and read the screen in another must hold a lock around
// both.
//
// # Not modeled
//
// Double-width and double-height lines, rectangular area operations
// (DECCRA, DECFRA...), 132-column mode, sixel and kitty graphics, the
// underline color (SGR 58), OSC 52 clipboard, OSC 7 and 133 (shell
// integration), and 8-bit C1 controls outside UTF-8. The alternate screen
// is cut or padded on resize rather than reflowed.
package phosphor
