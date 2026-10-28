# Performance and compatibility

Phosphor targets headless terminals that spend much of their time idle.
The measurements below describe pinned builds on one Apple M5 Pro host.
They measure emulation and retained state. Rendering, PTYs, client delivery
and whole-daemon performance need separate measurements.

## Read the evidence

The [evidence index](benchmarks.json) records sources, toolchains, library hashes,
sample ordering and archive checksums. The [balanced throughput audit](https://github.com/2018kguo/phosphor/releases/download/v0.4.2/phosphor-benchmark-audit.tar.gz)
contains all 360 samples, matched-build records and native verification logs.
[Download the historical evidence bundle](https://github.com/2018kguo/phosphor/releases/download/v0.4.2/phosphor-benchmark-evidence.tar.gz)
for the original timing and memory samples, source manifests, trace failures and
rejected experiments. The two experiments are recorded separately.
Its `manifest.json` verifies the original paths and SHA-256 of all files.
Historical reports describe their recorded revisions; this page is the maintained guide.

Run output belongs in the ignored `.bench/` directory. Keep source fixtures,
comparison tools and current findings in git. Publish evidence bundles with
checksums when updating a measured claim; do not add a directory per experiment.

## Throughput

Median MiB per process CPU second; higher is better. The recorded comparison
uses 120×40 terminals, one Go execution thread and 1 KiB writes. Six samples per
build, stream and history policy run for one second each. All six build-order
permutations are used, with repeat and position recorded in every sample.
Input generation, one warm-up feed, forced pre-run GC, construction and disposal
are outside timing. Process CPU includes GC within the timed window, native
work and the Go-to-C boundary. Both engines parse synchronously and format
replies into discard sinks.
Native continuation retention is disabled.

| Input | Phosphor, history | Native, history | Phosphor, no history | Native, no history |
|---|---:|---:|---:|---:|
| Synthetic agent TUI | 491.5 | 483.9 | 520.5 | 510.9 |
| Scrolling logs | 766.8 | 914.4 | 1,317.9 | 595.0 |
| Progress redraws | 2,009.7 | 1,124.7 | 2,088.6 | 1,129.5 |
| vim recording | 424.5 | 336.7 | 433.5 | 346.2 |
| less recording | 367.7 | 137.8 | 371.5 | 136.7 |
| nvim recording | 830.6 | 628.7 | 833.9 | 633.7 |
| nano recording | 705.0 | 578.4 | 706.1 | 579.3 |
| man recording | 752.0 | 435.5 | 733.9 | 429.2 |
| Mixed graphemes | 145.6 | 165.2 | 159.4 | 164.5 |
| OSC 8 links | 261.2 | 24.5 | 297.7 | 16.2 |

Nominal history budgets are 2,000 lines / 256 KiB. They retain different work:
Phosphor keeps 2,000 logical log lines; native keeps 387–678 physical rows.
For mixed graphemes those counts are 1,691 and 539–739. The history columns
compare each engine's configured policy, not equal retained contents. The
no-history control explicitly verifies zero retained rows on both engines.

The measured native grapheme lead is about 13% with history and 3% without it.
Small differences remain uncertain on a shared host; six short samples do not
identify an overall fastest terminal core. The native library was built in
ReleaseFast from the verified source, with the compiler binary and library
checksums recorded. Both binaries' loaded library paths were observed before timing.

Relative to the recorded Phosphor baseline, mixed-grapheme throughput increases
about 6.4×, progress 42% and budgeted logs 17%. The earlier archive includes
8 KiB/64 KiB writes and a distinct-cluster control. Those runs use different
sample ordering and durations and are not pooled into this table. The separate
Go benchmarks use wall time and decimal MB/s. Distinct clusters still allocate.

## Retained memory

An 80×24 prompt after erase retains about 1.51 KB of Go heap after `Compact` with a reply
sink installed. The measured prompt RSS comparison specifically uses prompts
followed by erase-to-end-of-line, then compaction. Across fresh processes with
128 or 1,024 emulators, Phosphor uses about 10–26× less incremental RSS than native
in those fixtures. This is a workload result, not a cost for every idle terminal.

Completed editor/pager recordings retain inactive alternate screens. At 1,024
emulators, Phosphor retains 22–47 KB of Go heap and uses about 19–51% less
incremental RSS than native. A separate fixture checks identical contents of
exactly 100 styled history rows before and after compaction; its RSS is about
3.1× lower. All 126 fresh-process memory samples are in the bundle.

Both engines compact. The same collection policy runs on both, without forced
scavenging. RSS depends on allocator pages and emulator count. Native allocations
are absent from Go heap counters, so cross-engine claims use RSS. These deltas
exclude process/shared-library startup, child programs, PTYs and output rings.
History byte budgets do not cap the entire emulator.

## Compatibility

The native trace corpus contains 218 named cases and 853 parity actions.
It compares cells, styles, links, cursor, modes, history and raw replies. Whole,
one-byte and seven-byte writes agree. Continuation checks compare each engine
with its own untouched control; they are separate from direct cross-engine parity.

| Observation | Passing actions |
|---|---:|
| Direct input | 852 / 853 |
| Phosphor checkpoint/repaint/Compact continuation | 851 / 853 |
| Native checkpoint/repaint continuation | 746 / 853 |

The direct difference is DEC special-graphics underscore: Phosphor uses xterm's
blank mapping; native keeps an underscore. Two Phosphor ANSI continuation cases
lose mixed alternate-screen backgrounds after resize and forced 1047 exit.
Exact snapshots retain those cells. Failures stay in the evidence, and these
counts are not a percentage of universal terminal compatibility.

Ordinary tests also run translated Ghostty/WezTerm tables, libvterm scripts,
Alacritty references, the pinned Unicode corpus and independent xterm-style cases.
The [comparison contract](../oracle/ghostty/coverage.md) defines bounds,
normalization and exclusions. [Source notices](../testdata/conformance/NOTICE.md)
preserve upstream attribution and policy differences.

## Reproduce

[Development](development.md) covers Go benchmarks and test suites. The
[native comparison guide](../oracle/ghostty/README.md) builds the native library,
then builds the same driver against two explicit core revisions. The burst
witnesses include cumulative cursor movement so missing middle input stays
observable even with no history or collected replies. Run them before interpreting timings. Use fresh processes for each memory sample,
retain failures and report actual history contents alongside throughput.
