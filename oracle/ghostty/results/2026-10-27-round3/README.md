# Unicode and exact checkpoints, 27 October 2026

Phosphor retains its headless Go design and idle-memory advantage while
closing measured gaps against libghostty-vt. This is progress on a finite
text-model contract, not a universal terminal compatibility percentage.

Runtime measured: `15c8763fd294c892b37e00f93e56cdd47608cff1` (later test and
evidence commits do not change runtime code). Baseline: phosphor v0.2.0,
`7bc614a8503d40c48e0c09a62c170ef0a892cb0f`. Native Ghostty:
`46e9dc9a927b2e7e7cba79e6902c176127dbdf2c`, Zig 0.17.0,
ReleaseFast. macOS 26.6.1, Apple M5 Pro, Go 1.26.3, arm64. Linux
correctness used Go 1.26.10, arm64, in a four-CPU Docker VM.

## Compatibility

The expanded paired corpus has 72 named cases and 174 parity actions.
The baseline passes 100; this candidate passes 150. Three additional actions
retain product-identity, host-color and cluster-capacity differences.
`traces-baseline.jsonl`, `traces-final.jsonl`, `traces-byte.jsonl`,
`traces-seven.jsonl` and `traces-linux.jsonl` retain every difference.
Whole, one-byte, seven-byte and Linux results agree exactly after removing
the chunk-size label. The trace command still exits 1 because parity gaps
remain; its successful execution is not a passing compatibility result.

All 39 original text/cursor probes pass for each engine. Eight program
streams agree at their original correctness geometry; each engine also
passes its 32 stream checkpoint continuations and eight exact-snapshot
edge cases. ANSI attach results remain separate, including failures.
See `checks.jsonl` and the [coverage contract](../../coverage.md).

The actual public `CodepointWidth` matches native on all 1,112,064 Unicode
scalars. The generated lookup uses 109 deduplicated packed pages: 11,328
bytes shared by the process, no heap allocation and no per-terminal field.
A range-search prototype caused a measured TUI slowdown; the packed lookup
and shared control-byte handling removed that regression. `native-widths.jsonl`
regenerates the lookup with `go run ./internal/unicodegen -widths PATH`.

The segmentation helper checks all 766 Unicode 17 corpus cases and 1,224
boundaries: 1,199 match the official boundaries and 25 retain the explicit
native control tailoring. This is helper coverage, not a claim about all
terminal dispatch or geometry. Clusters retain at most 32 UTF-8 bytes;
mode 2027 starts disabled. The translated Ghostty suite passes 502 cases,
keeps five documented differences and excludes 42 cases.

Remaining paired failures cover underline color, independent mouse bits,
tab policy after width changes, empty versus literal-space history cells,
and portable ANSI repaint outside origin margins. Existing documented
native differences and unsupported graphics/input surfaces remain visible.

## Throughput

Median MiB/s, five alternating rounds per stream, 120x40, 1 KiB chunks,
250 ms per sample. Process CPU includes Go GC and native work; raw wall and
CPU samples are in `throughput.jsonl`. Engines use the same generated
streams and recorded sessions, as in `oracle/ghostty/ab.sh`.

| Stream | v0.2.0 | Candidate | Ghostty |
|---|---:|---:|---:|
| claude-synthetic | 464.0 | 479.6 | 508.6 |
| logs | 553.7 | 547.7 | 931.7 |
| progress | 1322.4 | 1327.3 | 1146.7 |
| vim-recording | 398.8 | 401.8 | 348.0 |
| less-recording | 281.1 | 277.4 | 139.2 |
| rec-nvim-recording | 688.7 | 694.2 | 637.7 |
| rec-nano-recording | 516.6 | 516.7 | 574.5 |
| rec-man-recording | 459.2 | 441.3 | 440.3 |

These short samples do not establish small gains or regressions. A longer
seven-round, 750 ms confirmation for `man` gives medians 467.6 / 472.0 /
440.2 MiB/s (baseline / candidate / Ghostty), with substantial outliers on
both Go variants; see `man-confirmation.jsonl`. The candidate does not win
all throughput workloads: native still leads scrolling logs, the synthetic
TUI and nano, while phosphor leads several other recordings and progress.

## Idle memory

Fresh processes, 1,024 terminals, three samples per engine/workload.
Medians are bytes per terminal. Go heap describes phosphor's retained Go
objects; RSS includes native allocation. Ghostty's tiny Go wrapper heap is
not its emulator memory. The terminal remains 560 bytes and a cell 16 bytes
on these 64-bit hosts. Prompt writes remain zero-allocation with mode 2027
both off and on. Raw measurements are in `memory.jsonl`.

| Workload | Phosphor Go heap | Phosphor RSS delta | Ghostty RSS delta |
|---|---:|---:|---:|
| prompt | 2515 | 2192 | 31200 |
| colored-prompt | 2520 | 2256 | 47648 |
| cleared-compact | 2526 | 5552 | 64128 |
| rich-cleared-compact | 2535 | 6800 | 80608 |

Idle heap stays near v0.2.0's 2.5 KB. RSS is page-granular and depends on
touched pages and allocator reuse; these are process measurements, not a
fleet forecast. Rich-history replay sizing separately fell from 1,928,102
to 82,496 allocated bytes per operation by excluding URLs and escape
payloads from temporary cell geometry (`BenchmarkGraphemeHistoryReplay`).

## Checkpoints

`Snapshot` / `RestoreSnapshot` store explicit terminal state, independent
of ANSI drawing. They preserve both screens, saved cursors, palette base
and overrides, REP, tabs, modes, damage and unfinished parser input.
Restore is bounded, validates before replacing the receiver, copies its
input and retains receiver callbacks without invoking them. Malformed
cluster, title, hyperlink and history control sequences are rejected.
The format uses a corruption checksum, not an authentication mechanism.

Median microseconds, three 200 ms Go benchmark samples (`snapshots.txt`):

| Workload | Snapshot creation | ANSI creation | Exact restore | ANSI replay |
|---|---:|---:|---:|---:|
| Prompt | 0.35 | 0.44 | 0.56 | 0.71 |
| Sparse 200x50 | 1.00 | 1.28 | 1.17 | 1.46 |
| Full screen/history | 32.17 | 15.10 | 107.45 | 401.35 |
| Rich history | 42.55 | 18.72 | 164.96 | 1241.61 |

Full and rich restore are about 3.7x and 7.5x faster, respectively. Rich
restore allocation falls from about 731 KB / 6,495 allocations to 320 KB /
2,070 allocations. Dense snapshot creation uses about twice the CPU of
ANSI creation. Reusable append operations allocate zero times in these
workloads. Full/rich snapshots are slightly smaller than their ANSI
exports; sparse/prompt snapshots are substantially smaller.

Ptyd writes version 2 checkpoints using this API and reads existing
version 1 ANSI checkpoints. Both regular checkpoints and restore-decline
notices use the new writer. Tests cross checkpoint, tail data, resize,
future output and restart. The disk reader preserves ptyd's existing
accepted geometry/storage contract; the library's smaller default restore
limits are not silently imposed on owner-only daemon files. Old ptyd
binaries cannot read new version 2 checkpoints.

## Verification

- Go 1.24.0 full suite, Go 1.26.3 vet and full race suite on macOS.
- Linux arm64 race run: all tests except the initially omitted libvterm
  fixture scripts passed. A corrected tracked-file archive then passed
  the libvterm race test. The omission was in the test-copy command,
  not source behavior.
- Native driver unit tests and identical paired outputs on macOS/Linux.
- Snapshot malformed-input fuzz: 4,360,186 executions / 30 seconds;
  valid continuation: 812,677 / 45 seconds; grapheme transitions:
  865,377 / 30 seconds. Earlier token fuzz also passed. CI runs every
  target for 30 seconds, including the two new snapshot fuzzers.
- Examples build/vet; oracle module race/vet; Unicode generator rerun
  leaves both generated tables unchanged.
- Independent reviews reproduced and fixed malformed control breakout,
  accepted-geometry restore rejection, version/body mismatch and modifier
  joining across an intervening scalar. The repeated-pool-collection test
  compares resolved targets rather than incidental pool IDs.

No running ptyd daemon or installed executable was changed for these tests.
