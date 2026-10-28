# Using Phosphor

Use one goroutine per `Terminal`, or hold a lock around every access, including
reads, resize, compaction and snapshots. Reply writers and bell callbacks run
synchronously during `Write` and must not re-enter the terminal.

## Connect a PTY

Create the process and PTY using your host application's preferred library.
Given its master as `ptmx`, wire output and query replies like this:

```go
t := phosphor.New(80, 24,
	phosphor.WithReply(ptmx),
	phosphor.WithScrollback(5000),
	phosphor.WithScrollbackBytes(1<<20),
)
if _, err := io.Copy(t, ptmx); err != nil {
	return err
}
```

`Write` accepts arbitrary byte chunks, including split escape sequences and
UTF-8. `WithReply` writes answers back to the program; without it, queries
are ignored. Omit it when an attached real terminal is responsible for those
answers. Reply-writer errors are ignored, and the writer must not retain the
provided byte slice. `WithBell(fn)` handles BEL. `WithPalette(p)` sets the
host's initial colors and the colors reported to programs.

Phosphor answers common device/status/cursor reports, ANSI and DEC mode
queries, DECRQSS, kitty keyboard queries, text-area size, and palette queries
through OSC 4/10/11/12/21. Product identity replies identify Phosphor. Colors
can also be set and reset by supported OSC commands.

Send user input directly to the PTY. Your encoder uses `Modes()` for application
cursor/keypad modes, bracketed paste, focus events, mouse modes and kitty
keyboard flags. The independent mouse mode masks describe enabled bits;
`Mouse` and `MouseEncoding` report the effective selection. They need not
select a remaining enabled bit after a reset.

Call `Resize(cols, rows)` when the host size changes, and resize the PTY too so
the program receives SIGWINCH. Dimensions are at least one. The main screen
reflows soft wraps; the alternate screen is cut or padded. Width changes
reset tab stops to eight-column defaults; height-only changes preserve them.
Height-only growth pulls history when the original main cursor was on its
bottom row, including while that screen is inactive.

## Read and render the screen

| Read | Returns |
|---|---|
| `Size()`, `Title()` | Current geometry and title |
| `Line(y)`, `Lines()` | Visible text with trailing blanks removed |
| `Cell(x, y)` | Rune, combining text, width, style and hyperlink |
| `Cursor()` | Position, visibility, shape, blink and pending wrap |
| `Wrapped(y)` | Whether row y continues the preceding row |
| `Prompt(y)` | No prompt, primary prompt or continuation prompt |
| `Modes()`, `Palette()` | Program modes and current host/program colors |

An empty cell has `Rune == 0`; a wide character's second cell has `Width == 0`.
Skip that second cell when drawing. `Cell.Text()` includes combining text;
`Cell.AppendText(dst)` reuses a byte buffer. Default, indexed and RGB colors
remain distinct. Get `palette := t.Palette()`, then use
`palette.Resolve(color, background)` to obtain RGB. Apply inverse/reverse-video
modes in your renderer. A default underline color follows the foreground.

For each frame, draw rows where `RowDirty(y)` is true, then call `ClearDirty()`.
Before the first clear, every row is dirty and no tracking storage is needed.
Damage covers cells and palette/reverse-video changes; compare the cursor
separately. `Modes().SyncUpdate` indicates a program's synchronized frame.
The [mirror example](../examples/mirror/main.go) wires these pieces together.

OSC 133 shell markers appear through `Prompt(y)` as `PromptNone`,
`PromptPrimary` or `PromptContinuation`. They follow rows through scrolling
and reflow. A/N accept `redraw=0`, `redraw=1` and `redraw=last`; redraw starts
disabled on construction and is enabled by RIS.

## Keep history and release idle storage

History is stored as logical lines, so a soft-wrapped line counts once.
`ScrollbackLines()` returns text; `ScrollbackCells(i)` returns styled cells
laid out at the current width. The latter decodes a line and allocates.
`ClearScrollback()` drops retained history.

The defaults are 2,000 lines and about 256 KiB of serialized history.
`WithScrollback(n <= 0)` disables it. `WithScrollbackBytes(n <= 0)` removes
the byte limit. Oldest lines are evicted first; the newest line is always
kept, so the byte setting is approximate and does not bound total emulator
memory.

Call `Compact()` once output settles. It trims unused row tails, releases
dead Unicode/attribute entries and temporary buffers, and keeps observable
state and unfinished input. Later output may allocate again. Main-screen
rows grow as needed; alternate-screen output favors full rows for redraws.
Compaction is explicit and does not run on ordinary writes. A closed editor's
inactive alternate contents are still live and remain stored.

`Restart()` prepares for a new program while keeping the old screen above
the new cursor, including an active alternate screen. Constructor options
remain; program-set colors reset. Use it when relaunching a process, rather
than to resume a checkpoint.

## Save and restore

```go
checkpoint := t.Snapshot()
var restored phosphor.Terminal
if err := restored.RestoreSnapshot(checkpoint, phosphor.SnapshotLimits{}); err != nil {
	return err
}
```

Snapshots preserve exact cells on both screens, both DECSC slots, modes,
colors, REP, history, prompt state, damage and unfinished parser input.
`AppendSnapshot(dst)` reuses a buffer. Restore is atomic, copies its input and
replaces constructor state, including geometry and history policy. Only the
receiver's reply writer and bell callback remain; neither runs during restore.

Zero restore limits allow 64 MiB of decoded storage and four million cells per
screen. Set positive `MaxBytes` and `MaxCells` for your application; these
bound restoration, not future output. Use `errors.Is` with
`ErrSnapshotMalformed`, `ErrSnapshotVersion` and `ErrSnapshotLimit`.

Phosphor v0.4 writes snapshot format 2 and reads released format 1 checkpoints
from v0.3. Older readers cannot read format 2: upgrade readers before writing
new checkpoints. Legacy snapshots lack newer facts such as prompt metadata;
the reader supplies defaults and migrates the old cursor representation.
For ptyd checkpoints written with Phosphor v0.4, use ptyd v0.3 or newer.

## Attach a client

`Repaint(withScrollback)` produces ANSI for a fresh receiving terminal of the
same size. `true` includes history. It paints both screens and restores many
modes, saved cursors and pending input sequences. `AppendRepaint(dst, history)`
reuses storage. The bytes hide the cursor and use synchronized output while
painting; program-set palette colors are omitted to respect the client host.

Use Snapshot for exact persistence. ANSI cannot express every retained state:

- The preceding character for REP is not exported.
- Some cursors outside enabled origin margins need an unused alternate
  screen as scratch space; other combinations can move the resumed cursor.
- Very narrow screens with VS15-narrowed graphemes have replay limits.
- A forced 1047 exit can erase mixed old/new alternate-screen backgrounds
  after a resize; exact snapshots retain those cells.
- An unmarked prompt cursor may be reproduced with explicit input state.
  Headless text effects match, while per-cell GUI semantic classification
  can differ. Binary snapshots preserve the original enum.

These limits apply to future output as well as the first attached picture.
[Compatibility evidence](benchmarks.md#compatibility) keeps them visible.

## Supported behavior

Phosphor models UTF-8, Unicode 17 scalar widths, combining marks, wide cells,
DEC special graphics and the British charset; cursor movement, tabs, pending
wrap, editing, protection, BCE, scrolling and margins; screens 47/1047/1049;
SGR attributes and underline styles/colors; title, hyperlinks, shell markers,
palette commands, common replies and input modes. It supports gated
80/132-column switching, synchronized output and kitty keyboard flags.

Mode 2027 enables streaming grapheme clustering and starts disabled.
`CodepointWidth` reports a scalar's width, not the sum for a whole grapheme.
Each cell retains a primary codepoint plus 64 suffixes, at most 260 UTF-8
bytes. Further suffix text is dropped while its width effects still apply.
East Asian ambiguous characters have width one. Invalid UTF-8 becomes U+FFFD.

Hyperlinks and underline colors share up to 65,535 live attribute pairs.
Unused pairs are collected. If every pair is live, setting a new pair keeps
the current hyperlink/underline pair until space becomes available.

Phosphor supplies no keyboard/mouse/paste encoder, renderer, fonts, image
protocols, selection UI, clipboard provider or desktop integration. Unsupported
VT features include double-size lines, rectangular area operations, other
national charsets, SGR fonts and super/subscript, OSC 52/7, icon names/title
stack, and raw 8-bit C1/S8C1T behavior. C1 controls encoded as UTF-8 are handled.
If a specific command matters, check the [API](https://pkg.go.dev/github.com/2018kguo/phosphor)
and the [coverage contract](../oracle/ghostty/coverage.md).
