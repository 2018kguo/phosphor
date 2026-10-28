# phosphor

**A small, headless terminal emulator for Go programs that keep lots of PTYs alive.**

Feed it a program's output and read what a terminal would show: cells, colors,
hyperlinks, cursor, modes and scrollback. Save an exact checkpoint, or send ANSI
to a client attaching later. You supply the PTY, input encoding and renderer.

Phosphor is the screen model inside [ptyd](https://github.com/2018kguo/ptyd).
It is built for sessions that spend much of their lives waiting at a prompt,
with no client attached, but still need to resume an editor correctly.

![bash and vim running inside phosphor; the gutter lights up on changed rows](docs/demo.gif)

## Use it

Add it to a Go 1.24+ project:

```sh
go get github.com/2018kguo/phosphor
```

```go
package main

import (
	"fmt"
	"io"

	"github.com/2018kguo/phosphor"
)

func main() {
	t := phosphor.New(80, 24)
	io.WriteString(t, "\x1b[32mok\x1b[0m 12 tests\r\n")

	fmt.Println(t.Line(0)) // ok 12 tests
	fmt.Println(t.Cell(0, 0).Style.Fg == phosphor.IndexedColor(2)) // true
}
```

`Terminal` is an `io.Writer`; escape sequences and UTF-8 may be split across
writes. For a live program, use `phosphor.WithReply(ptmx)` to send terminal
query replies back through the PTY master. Some shells and editors wait for
those answers. The [reference](docs/reference.md#connect-a-pty) shows the wiring.

To try a renderer, clone the repo and run:

```sh
cd examples
go run ./mirror vim
```

The [mirror example](examples/mirror/main.go) produces the demo above. Its
characters and colors come from `Cell`; it redraws only changed rows.

## When it is useful

- **Hosting many sessions.** Keep a screen model for remote shells, coding
  agents or background jobs, including sessions nobody is currently viewing.
- **Inspecting terminal programs.** Read the current screen, attributes,
  hyperlinks and shell prompt markers instead of trying to interpret logs as
  an editor's display.
- **Attaching and recovering.** Give a new client the current picture with
  `Repaint`, and preserve exact emulator state on disk with `Snapshot`.
- **Building your own UI.** Draw cells in a browser, native app or terminal
  renderer, using row damage to avoid redrawing everything.

The library is pure Go and embeds in your process. It answers terminal
queries and models output; your application owns process management, keyboard
and mouse encoding, clipboard integration and the way cells appear on screen.

## Why choose it

**Idle memory stays small.** An 80×24 shell prompt used about **1.45 KB of Go
heap after Compact**, around **42% less than v0.3.0**, in the current measured
workload. Rows grow as they are used, history has configurable line and byte
budgets, and `Compact()` releases unused storage after output settles.

That prompt number is not the cost of every session. A closed editor's
alternate screen must stay available for later re-entry: the completed
recordings retained **22–47 KB of heap after Compact**. History and rich
attributes add their own storage. [Benchmarks](docs/benchmarks.md) explains
what is retained and how heap differs from process RSS.

**Continuation is part of correctness.** Checkpoints keep both screens,
saved cursors, modes, colors, scrollback and unfinished input. Tests continue
writing and resizing after restore, so a correct first picture is only the
start. The library reads released v1 snapshots as well as its current format;
[reader requirements](docs/reference.md#save-and-restore) matter when upgrading.

**Compatibility is checked against other terminals.** Five external suites,
real editor recordings, fuzzing and a pinned libghostty-vt comparison cover
cells, Unicode, replies and future behavior. Remaining policy differences and
ANSI replay limits are [published](docs/benchmarks.md#compatibility), including
the cases where terminals disagree. This is not a claim of universal VT
compatibility.

**Output parsing is fast, with tradeoffs.** The current comparison leads
libghostty-vt on several editor/pager recordings and progress output; native
leads on logs, a synthetic TUI and nano. Changes from v0.3.0 are mixed too.
Choose using [your workload](docs/benchmarks.md#throughput), rather than a claim
that one engine wins every stream.

## Tradeoffs

Phosphor models terminal text: UTF-8 and Unicode widths, colors and attributes,
wide characters and graphemes, hyperlinks, alternate screens, margins,
scrollback, common queries and input modes. `Modes()` tells your input encoder
what the running program requested.

It provides no window, fonts, GPU renderer, input encoder, sixel/kitty images
or clipboard provider. Some VT features, including rectangular operations
and double-size lines, are outside its scope. See the
[supported behavior](docs/reference.md#supported-behavior) before choosing it
for an application that depends on a particular escape sequence.

`Snapshot` is the persistence format. `Repaint` is portable terminal output
and has a weaker contract: REP, a few cursor/margin combinations, very narrow
graphemes and mixed-background alternate screens can differ on continued
output. Program-set palette colors are omitted from replay. See the precise
[snapshot reader requirements](docs/reference.md#save-and-restore) and
[ANSI attach limits](docs/reference.md#attach-a-client).

A `Terminal` needs exclusive access: use one goroutine or hold a lock around
all reads and writes. Reply writers and bell callbacks run during `Write`
and must not call back into that terminal.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/bench-dark.svg">
  <img alt="Historical Go-only comparison: throughput and idle memory for phosphor, charmbracelet/x/vt and hinshun/vt10x" src="docs/bench-light.svg">
</picture>

*The retained chart is the earlier Go-only comparison, not the current native
benchmark. Current measurements and their constraints are in the guide below.*

## Docs

- [Reference](docs/reference.md): PTY wiring, rendering, replies, history,
  checkpoints, replay and supported behavior.
- [Benchmarks](docs/benchmarks.md): current memory and throughput results,
  compatibility coverage, caveats and raw evidence.
- [Development](docs/development.md): run the demo, tests, benchmarks and
  optional native comparison.
- [Go API and examples](https://pkg.go.dev/github.com/2018kguo/phosphor): all
  exported types and methods.

## License

MIT. See [LICENSE](LICENSE). External test sources and their licenses are
listed in [NOTICE.md](testdata/conformance/NOTICE.md). X11 color names retain
the notices in [licenses/X11.txt](licenses/X11.txt).
