# Performance and compatibility

Choose Phosphor for small idle sessions, a Go cell model and exact checkpoints.
Parsing throughput varies with the program; it does not beat every other
engine on every stream. These results describe fixed workloads on an Apple
M5 Pro, macOS arm64, Go 1.26.3. Native comparisons use libghostty-vt at
`46e9dc9a927b2e7e7cba79e6902c176127dbdf2c`, ReleaseFast, Zig 0.17.0.

The [measurement report](../oracle/ghostty/results/2026-10-27-round6/README.md)
has the methods, source versions and every raw sample.

## Idle memory

At a shell prompt, an 80×24 terminal retained about **1.45 KB of Go heap after
Compact**, roughly **42% below v0.3.0**. Prefix rows avoid paying for a full
screen while only a prompt is visible.

Median bytes per terminal, three fresh processes with 1,024 live terminals:

| Workload after idle compaction | v0.3 heap | v0.4 heap | v0.4 RSS delta | Native RSS delta |
|---|---:|---:|---:|---:|
| Shell prompt, including EL | 2,521 | 1,449 | 1,264 | 31,280 |
| Colored prompt, including EL | 2,515 | 1,443 | 1,760 | 47,840 |
| Filled screen erased to prompt | 2,537 | 1,460 | 5,152 | 64,208 |
| Rich attributes erased to prompt | 2,546 | 1,472 | 5,072 | 80,720 |

Phosphor uses `Compact()` and native uses FULL compression. Go heap measures
retained Go objects. RSS measures resident process pages, including native
memory; the native Go-wrapper heap is not a meaningful substitute for it.
Allocator pages can remain resident after rows are freed, especially after a
large screen is erased. The comparison uses the same GC procedure for both.

A closed editor is a larger idle state. Correct 1049 exit retains its
alternate screen for later re-entry. Across completed vim/nvim/nano/less/man
recordings, Phosphor kept **22–47 KB heap and 34–57 KB RSS after Compact**,
versus about **82 KB native RSS after FULL compression** at 1,024 terminals.
The [120 completed-program samples](../oracle/ghostty/results/2026-10-27-round6/closed-memory.md)
also measure 256 terminals and show the smaller RSS savings for vim/nvim.

History and rich attributes add storage beyond a prompt. The serialized
history byte budget covers history, not the whole emulator; a newest line
can exceed it. Measure the screen contents and history retention you need.

## Throughput

Median MiB/s, five alternating rounds, 500 ms samples, 120×40, 1 KiB writes:

| Stream | v0.3.0 | v0.4.0 | libghostty-vt |
|---|---:|---:|---:|
| Synthetic coding-agent TUI | 441.8 | 427.4 | 467.7 |
| Logs | 521.8 | 537.9 | 884.7 |
| Progress bars | 1,286.4 | 1,237.2 | 1,098.6 |
| vim recording | 385.4 | 362.9 | 317.3 |
| less recording | 260.0 | 308.9 | 131.7 |
| nvim recording | 587.8 | 707.5 | 601.0 |
| nano recording | 499.2 | 508.9 | 563.5 |
| man recording | 409.7 | 519.9 | 407.8 |

Phosphor leads native on five of these streams; native leads logs, the
synthetic TUI and nano. Against v0.3.0, several recordings improve while the
TUI, progress and vim medians are about 3–6% lower. Short samples and shared
host scheduling do not establish small regressions or universal speedups.
Raw wall and process CPU samples retain the variation and every outlier.

These numbers measure emulator work, not process startup, PTY latency,
networking, rendering or an entire daemon. Chunk size and terminal geometry
matter. Use your own streams before budgeting a service around a headline
throughput number. See [development](development.md#measure-your-workload)
for commands.

The README's Go-only chart is historical: it measures the original comparison
with charmbracelet/x/vt and hinshun/vt10x using different versions, defaults
and 8 KiB writes. It is not the source for the current native table.

## Compatibility

The native comparison captures complete physical text cells, colors,
underline styles, hyperlinks, cursor, row prompt/wrap metadata, modes,
history and raw replies. Untouched controls distinguish direct-input parity
from state lost by either engine's checkpoint or ANSI exporter.

The expanded corpus has **208 named cases and 773 parity actions**. Whole,
one-byte and seven-byte writes produce the same results:

| Comparison | Passing actions |
|---|---:|
| Direct input, untouched engines | 772 / 773 |
| Phosphor checkpoint/repaint/Compact continuation | 771 / 773 |
| Native checkpoint/repaint continuation | 666 / 773 |

The direct difference is DEC special-graphics underscore: Phosphor keeps
xterm's blank mapping, while the pinned native engine keeps an underscore.
Two Phosphor ANSI variants lose mixed old/new alternate-screen backgrounds
on forced 1047 exit after resize. Exact snapshots preserve those cells.
The [raw differences](../oracle/ghostty/results/2026-10-27-round6/README.md)
remain failures in the denominator. Passing the rest is not a percentage
of universal terminal compatibility.

Ordinary Go tests also run five external suites: libvterm scripts, translated
Ghostty and WezTerm tests, Alacritty references, and independently written
xterm/esctest-style cases. Upstream references stay unchanged when terminal
policies conflict. [The coverage contract](../oracle/ghostty/coverage.md)
defines observations and exclusions; [the notices](../testdata/conformance/NOTICE.md)
record sources and licenses. Graphics and GUI behavior are outside this
headless text-model comparison.

Snapshot tests cover future output and resize, including checkpoints written
by the released v0.3.0 library. Recordings and fuzz tests exercise chunk
boundaries, Unicode, dirty rows and exact versus ANSI continuation. Read
[the API limits](reference.md#attach-a-client) when attach fidelity matters.

The [earlier Go-only chart](bench-light.svg) ([dark version](bench-dark.svg))
compares older library versions. Use the tables above for the current native
comparison.
