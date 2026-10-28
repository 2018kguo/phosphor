# Performance and compatibility

Phosphor targets headless terminals that spend much of their time idle.
The measurements below describe pinned builds on one Apple M5 Pro host.
They measure emulation and retained state. Rendering, PTYs, client delivery
and whole-daemon performance need separate measurements.

## Read the evidence

The [evidence index](benchmarks.json) records the measured source revisions,
native optimization, toolchain, library hash, sample ordering and archive checksum.
[Download the complete evidence bundle](https://github.com/2018kguo/phosphor/releases/download/v0.4.2/phosphor-benchmark-evidence.tar.gz)
for every raw sample, source manifest, trace failure and rejected experiment.
Its `manifest.json` verifies the original paths and SHA-256 of all files.
Historical reports describe their recorded revisions; this page is the maintained guide.

Run output belongs in the ignored `.bench/` directory. Keep source fixtures,
comparison tools and current findings in git. Publish evidence bundles with
checksums when updating a measured claim; do not add a directory per experiment.

## Throughput

Median MiB per process CPU second; higher is better. The recorded comparison
uses 120×40 terminals, one Go execution thread and 1 KiB writes. Five samples
per build and stream run for 750 ms with history or 500 ms without it.
Inputs are generated before timing. CPU includes GC, native work and the Go-to-C
boundary. Both engines parse synchronously and format replies into discard sinks.
Native continuation retention is disabled.

| Input | Phosphor, history | Native, history | Phosphor, no history | Native, no history |
|---|---:|---:|---:|---:|
| Synthetic agent TUI | 511.3 | 504.3 | 515.9 | 500.4 |
| Scrolling logs | 769.5 | 932.9 | 1,315.1 | 594.1 |
| Progress redraws | 2,036.1 | 1,144.9 | 2,101.6 | 1,139.3 |
| vim recording | 440.1 | 346.5 | 433.3 | 346.3 |
| less recording | 377.0 | 139.0 | 380.5 | 137.3 |
| nvim recording | 834.4 | 630.6 | 835.0 | 634.7 |
| nano recording | 705.7 | 578.6 | 700.1 | 581.7 |
| man recording | 760.4 | 434.4 | 751.9 | 440.9 |
| Mixed graphemes | 146.1 | 156.4 | 180.1 | 187.5 |
| OSC 8 links | 275.0 | 24.3 | 359.1 | 22.4 |

Nominal history budgets are 2,000 lines / 256 KiB. They retain different work:
Phosphor keeps 2,000 logical log lines; native keeps 474–699 physical rows.
For mixed graphemes those counts are 1,691 and 374–570. The history columns
compare each engine's configured policy, not equal retained contents. The
no-history control explicitly verifies zero retained rows on both engines.

The archive also includes 8 KiB and 64 KiB writes, for 600 timing samples in all.
Native's measured grapheme lead ranges from 7–20% with history across those sizes,
and about 4% without history at 1 KiB. These short samples do not establish
small differences reliably or identify an overall fastest terminal core.
The historical runner reversed the endpoint order but always put the candidate
second; its inner `round` field is zero. File order reconstructs the five outer
repeats. The maintained runner rotates all three positions and records the
outer repeat and position explicitly. Historical samples remain unchanged.

Relative to the recorded Phosphor baseline, mixed-grapheme throughput increases
about 6.3×, progress 41% and budgeted logs 13%. A separate Go benchmark with
1,024 distinct clusters improves about 2.4×. It uses wall time and decimal MB/s;
do not combine its units with the native table. Distinct clusters still allocate.

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
then builds the same driver against two explicit core revisions. Run its burst
witnesses before interpreting timings. Use fresh processes for each memory sample,
retain failures and report actual history contents alongside throughput.
