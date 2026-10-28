# Performance and compatibility

Measure the input, retained state and host operations you plan to use.
Phosphor favors small idle storage and Go integration. The current comparison
shows faster editor/pager streams and a large native lead on grapheme-heavy
Unicode. History policy changes the log result.

The [measurement report](../oracle/ghostty/results/2026-10-28-unicode-history/README.md)
contains all tables, raw samples and reproduction commands. Its
[metadata](../oracle/ghostty/results/2026-10-28-unicode-history/metadata.json)
records source pins, toolchains, binary hashes and policies. These are headless
macOS arm64 measurements on an Apple M5 Pro, not GUI or whole-daemon results.

## Idle storage

An 80×24 prompt retains about **1.51 KB of Go heap after Compact**, with a
reply sink installed. Rows grow as needed. Baseline and candidate retained
heaps are unchanged by this performance round.

Prompt RSS is about **9–26× lower than native** across the tested 128- and
1,024-emulator processes. The range matters: allocator pages and amortization
change RSS even when retained Go objects stay the same. Both engines compact;
native continuation retention is disabled. The earlier larger ratios included
optional native checkpoint bookkeeping and should not be carried forward.

Completed editors and pagers retain their inactive alternate buffers. At
1,024 emulators, Phosphor retains 22–47 KB of Go heap and 37–61 KB of incremental
RSS, versus about 75 KB native RSS: **19–50% less**. A separate fixture retaining
exactly the same 100 styled history rows uses about **3.1× less RSS**. Tests
compare the complete retained model before and after compaction.

These are incremental emulator costs. They exclude child programs, PTYs,
output rings and shared-library startup. Native Go-wrapper heap excludes
native allocations; use RSS for comparisons between engines. The history
byte budget covers serialized history rather than the entire emulator.

## Throughput

Mixed graphemes improve **2.5×**, budgeted logs **7%** and progress output
**20%** over the Phosphor baseline in the primary CPU comparison. All builds
receive identical inputs, format replies into discard sinks and alternate
order. Process CPU includes GC and native work. Five samples per build/case
are retained; small gains remain uncertain on a shared host.

Equal byte budgets retain different amounts of history. With nominal
2,000-line /256 KiB limits, Phosphor keeps 2,000 logical log lines while native
keeps 504–654 physical rows in the primary samples. Native is about **27% faster**
there. With history disabled and zero rows verified on both sides, Phosphor is
about **2.1× faster**. Compare retention alongside throughput.

Both engines enable grapheme mode for the Unicode fixture. Native still leads
by about **3×**, reduced from roughly 7.3× in the previous audit. A separate
1,024-distinct-cluster control improves **2.2×**, including cache misses. The
repeated mixed fixture allocates about **40 KB per MiB operation**, down from
153 MB. Varied clusters still allocate; warm repeat cases have zero-allocation
checks. These Go allocation benchmarks use wall time and decimal MB/s, rather
than the native table's MiB per CPU second.

Editor/pager recordings and progress remain ahead of native under the recorded
budgets; the OSC 8 link fixture also favors Phosphor. Additional 8 KiB and
64 KiB phases show how chunk size changes ratios. A shared 7,008-byte Unicode
lookup adds no fields to each terminal, preserving the small idle model.

Every write is parsed synchronously. A large-burst test checks replies,
ordered history, cells, styles, links, graphemes, modes and future input.
Its negative control discards the leading input and must fail. This protects
against measuring reads while silently skipping terminal emulation.

The separate [Go-core measurements](../oracle/ghostty/results/2026-10-27-agent-round/README.md#other-go-cores)
use library defaults and decimal MB/s. Feature coverage, reply handling and
history settings differ; those results do not isolate equivalent work.
The historical [Go-only chart](bench-light.svg) describes an earlier experiment.

## Compatibility

The native comparison captures text cells, colors, underline styles,
hyperlinks, cursor, prompt/wrap metadata, modes, history and raw replies.
Untouched controls distinguish direct-input differences from state lost by
either engine's checkpoint or ANSI exporter.

The corpus has **208 named cases and 773 parity actions**. Whole, one-byte
and seven-byte writes produce the same results:

| Comparison | Passing actions |
|---|---:|
| Direct input, untouched engines | 772 / 773 |
| Phosphor checkpoint/repaint/Compact continuation | 771 / 773 |
| Native checkpoint/repaint continuation | 666 / 773 |

The direct difference is DEC special-graphics underscore: Phosphor keeps
xterm's blank mapping, while native keeps an underscore. Two Phosphor ANSI
variants lose mixed alternate-screen backgrounds on forced 1047 exit after
resize. Exact snapshots preserve those cells. These failures remain in the
[raw corpus results](../oracle/ghostty/results/2026-10-28-unicode-history/README.md#behavior-and-reproduction);
passing the rest is not a percentage of universal terminal compatibility.

Ordinary Go tests also run libvterm scripts, translated Ghostty and WezTerm
tests, Alacritty references and independently written xterm/esctest-style cases.
Upstream references remain unchanged when terminal policies conflict.
[Coverage](../oracle/ghostty/coverage.md) defines observations, bounds and
excluded graphics/GUI behavior; [notices](../testdata/conformance/NOTICE.md)
record sources and licenses.

Snapshot tests cover future output and resize, including earlier saved
formats. Recordings and fuzzing exercise chunk boundaries, Unicode, dirty
rows and exact versus ANSI continuation. Read the
[export limits](reference.md#attach-a-client) when attach fidelity matters.
