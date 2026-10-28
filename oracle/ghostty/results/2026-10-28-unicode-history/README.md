# Unicode, scrolling logs and idle storage

This round makes Phosphor about **2.5× faster on mixed graphemes**, **7% faster
on budgeted logs** and **20% faster on progress output** in the primary CPU
comparison. libghostty-vt still leads mixed graphemes by about **3×** and
budgeted logs by **27%**. Agent TUI throughput is within 1%; small differences
need longer measurements.

The [metadata](metadata.json) and source manifests for the [baseline](baseline-source.json)
and [candidate](candidate-source.json) identify the measured code, binaries and
native library. All 600 throughput samples and 126 fresh-process memory
samples are retained. The host was macOS arm64 on an Apple M5 Pro; native used
ReleaseFast. These measurements cover headless terminal emulation.

## What reading Ghostty changed

Ghostty's [Unicode path](https://github.com/ghostty-org/ghostty/blob/6ba144158577d033ccb6ed4b574e4013957c9161/src/terminal/Terminal.zig#L679)
checks simple scalar pairs before replaying a stored cluster. Its
[boundary helper](https://github.com/ghostty-org/ghostty/blob/6ba144158577d033ccb6ed4b574e4013957c9161/src/unicode/grapheme.zig#L37)
uses precomputed property/state transitions, and its
[cluster storage](https://github.com/ghostty-org/ghostty/blob/6ba144158577d033ccb6ed4b574e4013957c9161/src/terminal/Page.zig#L1562)
usually appends without allocating a new string.

Those choices suggested changes that fit Phosphor's storage:

- Reuse its existing temporary buffer for boundary scans.
- Skip segmentation between two single scalars classified as Other. A generated
  7,008-byte lookup is shared by the process; stored clusters still use the full rules.
- Check the existing recent-cluster pool before allocating appended text.
- Dispatch complete short CSI commands directly and reduce work when formatting ASCII history.

No per-terminal fields, mutable cluster storage or native dependency were added.
The scalar lookup comes from the existing checksum-verified Unicode source;
Ghostty's state-machine implementation was not copied. Complex clusters still
use the pinned Go segmenter, leaving room to improve that path further.

## CPU throughput

Median MiB per process CPU second, including GC and native work. Five alternating
samples per build/case, one Go execution thread, 120×40 geometry and 1 KiB writes.
Budgeted samples last 750 ms; zero-history samples last 500 ms. Input generation
precedes timing; both engines format replies into discard sinks. Native
continuation retention is disabled outside checkpoint checks.

| Stream | Baseline, budgeted | Phosphor, budgeted | Native, budgeted | Phosphor, no history | Native, no history |
|---|---:|---:|---:|---:|---:|
| Agent TUI | 464.1 | 500.4 | 502.2 | 512.8 | 504.9 |
| Logs | 676.5 | 726.9 | 922.5 | 1,223.9 | 590.9 |
| Progress | 1,457.7 | 1,744.9 | 1,139.1 | 1,763.5 | 1,077.1 |
| vim | 415.0 | 443.8 | 345.6 | 420.1 | 332.3 |
| less | 355.3 | 379.0 | 138.7 | 355.7 | 129.7 |
| nvim | 771.4 | 784.3 | 633.8 | 755.7 | 556.6 |
| nano | 626.9 | 661.6 | 579.6 | 627.7 | 541.9 |
| man | 622.4 | 708.6 | 438.3 | 621.6 | 394.9 |
| Grapheme-heavy Unicode | 23.0 | 57.0 | 168.0 | 65.1 | 161.5 |
| OSC 8 links | 261.0 | 277.3 | 24.5 | 359.2 | 19.2 |

With nominal 2,000-line /256 KiB budgets, Phosphor retained 2,000 logical log
lines and native retained 504–654 physical rows in the primary samples.
These are different amounts of retained work. With history disabled and zero
rows verified, Phosphor is about **2.1× faster on logs**. Neither policy gives
a universal parsing-speed ratio.

All ten primary medians improve over the Phosphor baseline; the smallest gain
is 1.7%, which short shared-host samples do not resolve confidently. The
[1 KiB](throughput-1024.jsonl), [8 KiB](throughput-8192.jsonl),
[64 KiB](throughput-65536.jsonl) and [zero-history](throughput-none-1024.jsonl)
records retain input hashes, byte totals and actual history counts. The
[summary](throughput-summary.json) includes every phase. These costs exclude
PTY I/O, HTTP, rendering and client delivery.

## Unicode allocation and cache misses

Separate Go benchmarks use wall time and decimal MB/s, five alternating 200 ms
samples, 1 KiB writes and about 1 MiB per operation. Their ratios should not be
mixed with the native CPU table.

In the repeated mixed fixture, allocated bytes fell from about **153 MB to
40 KB per operation**. Warm repeated combining text and CJK have zero-allocation
regression checks. This does not mean every Unicode stream allocates nothing.

The [distinct-cluster control](variety-summary.json) improves throughput **3.3×**
with 16 clusters and **2.2×** with 1,024 clusters. The latter allocates 45 MB per operation, compared with 240 MB in the baseline.
New clusters and pool compaction still cost storage.
[Unicode samples](unicode-paired.txt), [medians](unicode-summary.json),
[variety samples](variety-paired.txt) and [all exploratory screens](screening/method.json)
retain the evidence, including rejected ideas and an initial timing overlap.
Only the integrated runs support the native comparison above.

## Idle storage

Retained Go heap is unchanged from the baseline: a compacted prompt uses about
**1,512 bytes**, and completed editor buffers remain available for re-entry.
At 1,024 emulators, editor/pager RSS is 19–50% lower than native; exactly 100
styled history rows use about 3.1× less RSS. Prompt RSS is about 9–26× lower
across the tested 128- and 1,024-emulator processes.

Both engines compact. Each sample starts in a fresh process with the same GC
policy and no forced scavenging. Native wrapper heap excludes native memory;
use RSS for cross-engine comparisons. Allocator pages change RSS even when
retained Go objects match. The process-wide lookup adds 7,008 bytes of static
data, rather than storage per session. Child programs, PTYs, output rings and
shared-library startup are excluded.

[All memory medians](memory-summary.json), [128-terminal samples](memory-128.jsonl)
and [1,024-terminal samples](memory-1024.jsonl) remain available. Equal-history
contents are checked before and after both compactions.

## Behavior and reproduction

The 208-case corpus produces byte-identical baseline/candidate observations for
whole, one-byte and seven-byte writes. Direct parity remains 772/773 actions;
Phosphor continuation remains 771/773 and native continuation 666/773.
The DEC underscore policy difference and the two Phosphor ANSI background
variants remain visible. This is a bounded corpus, not a universal compliance score.

[Raw traces](trace-phosphor-0.jsonl), [chunked traces](trace-phosphor-1.jsonl)
and [native checks](native-checks.txt) retain failures as well as successes.
The burst witness verifies all input, replies, history and future state; its
input-discarding negative control fails on both engines. Unicode boundaries,
cache misses, scratch ownership, history encoding and future checkpoint behavior
also have focused regression checks. Independent source reviews found no blockers.

Use the [build instructions](../../README.md#build) and common driver with both
production revisions from the metadata. From `oracle/`, reproduce a phase with:

```sh
GOMAXPROCS=1 PHOSPHOR_DIR="$(cd .. && pwd)" \
  COMPARE_CHUNK=1024 COMPARE_DURATION=750ms COMPARE_HISTORY=budgeted \
  bash ghostty/ab.sh /tmp/baseline /tmp/candidate /tmp/throughput.jsonl
```

Repeat with the other chunk sizes and the `none` history policy. Build each Go
revision with the same current benchmark files for the auxiliary Unicode and
cache-miss controls. Keep every sample and compare retention alongside speed.
