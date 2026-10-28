<h1 align="center">Phosphor</h1>
<p align="center">A small terminal emulator for Go programs that keep many sessions alive.</p>
<p align="center">
  <a href="#use-it">Get started</a> ·
  <a href="docs/reference.md">Reference</a> ·
  <a href="docs/benchmarks.md">Benchmarks</a> ·
  <a href="docs/development.md">Development</a>
</p>

Feed Phosphor a program's output and read its screen: text, colors, hyperlinks,
cursor and scrollback. Save a checkpoint to resume later, or send the current
screen to a client attaching mid-session. You supply the PTY and renderer.

It powers [ptyd](https://github.com/2018kguo/ptyd), where shells and editors
often sit idle with nobody watching. Small retained state and reliable
recovery matter alongside parsing speed.

![bash and vim running inside Phosphor; the gutter lights up on changed rows](docs/demo.gif)

## Use it

```sh
go get github.com/2018kguo/phosphor
```

```go
package main

import (
	"fmt"

	"github.com/2018kguo/phosphor"
)

func main() {
	t := phosphor.New(80, 24)
	t.Write([]byte("\x1b[32mhello\x1b[0m\r\n"))
	fmt.Println(t.Line(0)) // hello
}
```

`Terminal` is an `io.Writer`; UTF-8 and escape sequences can span writes.
Use `Cell` for attributes and `RowDirty` to redraw changed rows. For a live
program, connect query replies with `phosphor.WithReply(ptmx)`: shells and
editors may wait for those answers. See [PTY wiring](docs/reference.md#connect-a-pty).

To run the renderer shown above, clone the repo, then:

```sh
cd examples
go run ./mirror vim
```

## Why choose Phosphor

- **Go integration.** Pure Go, no cgo, external Go dependencies or separate
  compiler toolchain. Read cells directly from your application.
- **Small idle sessions.** Rows grow as needed, history has line and byte
  budgets, and `Compact()` releases unused storage when output settles.
- **Recoverable state.** `Snapshot` saves both screens, history, modes,
  saved cursors and unfinished input. Tests continue writing and resizing
  after restore.
- **Your own interface.** Build a browser view, terminal renderer or agent
  tool around the same screen model.

### Compared with libghostty-vt

[libghostty-vt](https://github.com/ghostty-org/ghostty/blob/main/include/ghostty/vt.h)
is also a headless, embeddable core with snapshots and idle compression.
Phosphor's reasons to exist are its Go API and storage model. The recorded
comparison shows where those choices help and where libghostty-vt leads:

| Recorded workload | Result |
|---|---|
| Prompts after erase and compaction | Phosphor uses **10–26× less incremental RSS**, across 128 and 1,024 emulators |
| Completed editors and pagers, inactive screens retained | Phosphor uses **19–51% less incremental RSS** |
| Equal contents: 100 styled history rows | Phosphor uses **3.1× less incremental RSS** |
| Editor/pager recordings and progress | Phosphor processes **1.22–2.67× more bytes per CPU second** with the recorded history budgets |
| Scrolling logs | libghostty-vt is **19% faster** with budgeted history; Phosphor is **2.2× faster** without history |
| Mixed graphemes | libghostty-vt is **13% faster** with history and **3% faster** without history |

These are headless workloads on one host. RSS is process memory; the deltas
exclude startup, PTYs and child programs. Equal history budgets retain
different contents because Phosphor stores logical lines and libghostty-vt
prunes physical pages. [Methods, retained contents and raw samples](docs/benchmarks.md)
are published together.

Choose libghostty-vt when you want Ghostty's behavior and its broader APIs,
including input encoders, and can accept a native dependency. Choose Phosphor
when its Go integration and retained-memory costs fit your host application.

## Tradeoffs

Phosphor models terminal text, including Unicode graphemes, attributes,
hyperlinks, alternate screens, margins, queries and input modes. It supplies
no window, fonts, input encoder, clipboard provider or terminal graphics.
Some VT features, including rectangular operations and double-size lines,
are outside its scope. Check [supported behavior](docs/reference.md#supported-behavior)
and [compatibility coverage](docs/benchmarks.md#compatibility) for your programs.

Use `Snapshot` for exact persistence. `Repaint` emits ANSI for attachment
and has [documented limits](docs/reference.md#attach-a-client), including
palette changes and some alternate-screen, cursor and grapheme states.
See [checkpoint reader requirements](docs/reference.md#save-and-restore)
when upgrading.

A `Terminal` needs exclusive access: use one goroutine or a lock around reads
and writes. Reply writers and bell callbacks run during `Write` and must not
call back into that terminal.

## Documentation

- [Reference](docs/reference.md): PTYs, rendering, replies, history and persistence.
- [Benchmarks](docs/benchmarks.md): performance, compatibility and reproducible evidence.
- [Development](docs/development.md): tests, examples and libghostty-vt comparisons.
- [Go API](https://pkg.go.dev/github.com/2018kguo/phosphor): exported types and methods.

## License

MIT. See [LICENSE](LICENSE), [external test notices](testdata/conformance/NOTICE.md)
and [X11 color notices](licenses/X11.txt).
