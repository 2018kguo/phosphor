# Memory after completed editor and pager recordings

Keep the native-compatible inactive alternate screen and use the existing
`Compact()` implementation. This release adds no cold-screen representation,
compression codec, thaw importer or capacity reservation. Correct retention
has a real idle cost: after compaction these completed recordings retain
roughly 22–47KB of Go heap per terminal, compared with roughly 54–56KB before
compaction. These are larger states than a fresh shell prompt.

## Exact source and binary

- Phosphor and permanent driver source: `8d1511ce4f0d5b5951f4506818c4da14b4f4a3bc`.
- Native source: `46e9dc9a927b2e7e7cba79e6902c176127dbdf2c`, built with
  Zig 0.17.0, `zig build -Demit-lib-vt -Doptimize=ReleaseFast -j4`.
- Sampled driver SHA256: `bb7cd704d558ed4d0c7b374284af079a582623f5a1a6e8f0de75fbd1d94745bb`.
- Linked `libghostty-vt.dylib` SHA256: `7700d7817d5c3c7107e5efb9cd165a1d9f16e877fe9f2cf886c528bfb707acbe`.
- Production source manifest SHA256: `5880bcd622156f80410cce12b6fd3bb75918e6526b86261a088326d4ce4973b4`.
  Every file was checked against the committed source before building and
  checked again after all samples. The manifest hashes root Go files,
  internal Go files, driver Go/C/header files and both modules' go.mod/go.sum,
  excluding tests. It is the SHA256 of the path-to-file-SHA256 JSON object,
  with sorted keys and compact separators. No uncommitted production bytes
  were used. The unrelated README edit does not enter this build.
- Platform: macOS arm64; Go 1.26.3; cgo enabled with the `ghostty` build tag.

Build from `oracle/` using the native checkout's include and library paths:

```sh
CGO_CFLAGS="-I$GHOSTTY_SOURCE/include" \
CGO_LDFLAGS="-L$GHOSTTY_SOURCE/zig-out/lib -lghostty-vt -Wl,-rpath,$GHOSTTY_SOURCE/zig-out/lib" \
go build -tags ghostty -o /tmp/phosphor-ghostty ./ghostty
PHOSPHOR_DIR=/path/to/phosphor /tmp/phosphor-ghostty \
  -mode memory -engine phosphor -workload closed-vim-compact -n 1024
```

The binary SHA is for the recorded local build, whose absolute include,
library and rpath paths were `/private/tmp/phosphor-gaps.ro32ng/ghostty`.
Building elsewhere need not produce the same binary hash.

## Method and raw samples

[closed-memory-1024.jsonl](closed-memory-1024.jsonl) and
[closed-memory-256.jsonl](closed-memory-256.jsonl) retain all 120 samples:
five recordings, two engines, plain/compacted, three fresh processes each,
at two terminal counts. Each JSON line is the driver's raw output with
only a `repeat` label added. Engine order alternates between repeats.
There are no discarded outliers or reply/cell normalization.

The permanent `closed-{vim,nvim,nano,less,man}[-compact]` workloads load the
actual completed `rec-X.raw` inputs at 100×30. They reject unknown recording
names and reject a final active alternate screen. Phosphor compaction is
`Compact()`; native compaction is `GHOSTTY_TERMINAL_COMPRESSION_MODE_FULL`.
Both engines are alive when counters are collected. The driver warms shared
initialization, runs GC before the baseline and twice after initialization,
then records retained Go heap and process RSS. It does not force the Go
scavenger to return all free pages to the OS.

All table values are median 3 **bytes per terminal**. The native Go-heap field
in the raw files measures only its small Go wrapper, excluding C allocations;
use RSS for the cross-engine memory comparison.

### 1024 terminals

| Recording | Phosphor heap | Phosphor heap after Compact | Phosphor RSS | Phosphor RSS after Compact | Native RSS | Native RSS after FULL |
|---|---:|---:|---:|---:|---:|---:|
| vim | 56,198.5 | 46,958.9 | 57,520 | 56,768 | 82,288 | 82,352 |
| nvim | 56,211.2 | 47,316.2 | 57,424 | 55,392 | 82,592 | 82,608 |
| nano | 54,387.4 | 37,812.0 | 55,584 | 45,680 | 82,416 | 82,336 |
| less | 56,185.0 | 30,148.0 | 57,440 | 42,272 | 82,480 | 82,544 |
| man | 56,158.5 | 22,431.0 | 57,456 | 33,824 | 82,352 | 82,320 |

### 256 terminals

| Recording | Phosphor heap | Phosphor heap after Compact | Phosphor RSS | Phosphor RSS after Compact | Native RSS | Native RSS after FULL |
|---|---:|---:|---:|---:|---:|---:|
| vim | 56,226.0 | 46,985.4 | 55,552 | 53,120 | 83,904 | 83,968 |
| nvim | 56,250.3 | 47,337.8 | 55,488 | 54,016 | 84,416 | 84,416 |
| nano | 54,447.2 | 37,878.7 | 53,952 | 44,288 | 83,648 | 83,584 |
| less | 56,240.5 | 30,169.8 | 55,424 | 42,368 | 84,160 | 84,032 |
| man | 56,208.5 | 22,457.9 | 55,616 | 36,288 | 83,648 | 83,712 |

Compaction's heap savings are larger than its immediate RSS savings because
allocator pages can remain resident after rows are freed. This is especially
visible for vim and nvim. Native FULL compression makes little difference
for these no-history workloads: the pinned native Terminal.compress targets
PRIMARY scrollback pages, while the completed program's retained ALT remains
resident. Native page estimates in the raw files are allocation estimates,
not resident-process measurements.

## What occupies the retained alternate screen

A separate read-only cell inspection counted physical cell categories and
slice capacities, distinguishing literal spaces from empty cells. The
100×30 grid has 3000 positions. Prefix compaction removes only true zero
suffixes and preserves every literal or styled space.

| Recording | ALT cell capacity before | ALT cell capacity after | Retained cells after | Zero cells in full grid | Default spaces | Styled spaces | Other cells |
|---|---:|---:|---:|---:|---:|---:|---:|
| vim |48,000B |41,024B |2,564 |443 |577 |335 |1,645 |
| nvim |48,000B |41,664B |2,604 |818 |337 |89 |1,756 |
| nano |46,400B |32,544B |2,034 |1,012 |251 |113 |1,624 |
| less |48,000B |26,032B |1,627 |1,373 |275 |0 |1,352 |
| man |48,000B |18,944B |1,184 |1,816 |281 |0 |903 |

Each also retains 720B of row headers. These are capacity payloads, excluding
Go size-class rounding, Terminal, saved cursors and lookup tables; they are
not interchangeable with the measured heap or RSS. Some zero cells are
inside retained prefixes, particularly in nvim. There are no styled empty
cells in these five final ALT grids. Ordinary cell size remains 16B and
Terminal remains 560B on 64-bit platforms.

Default spaces remain observable through Cell, and styled spaces also retain
appearance. They cannot be treated as empty padding just to lower memory.
The final ALT grids contain 1,941/2,038/1,866/1,567/1,069 complete-cell runs
respectively. A naive 24B-per-run representation would exceed the existing
compacted cell prefixes for every recording. Sparse-history VT encoding is
not a drop-in exact screen representation: it reconstructs logical lines
through terminal dispatch/reflow, and a cold ALT would still require one
canonical live/cold owner and safe pool import/mutation lifetimes. That
complexity is deferred, with no claimed compression benefit in this release.

The corrected 1049l behavior retains ALT contents for later 47h re-entry;
`TestCursorLifetimeCheckpoint` and the paired native
`TestNativeCursorLifetimeAndRepaint` exercise that future, including snapshot
and ANSI replay. The RSS workload also requires the completed program to
have returned to MAIN. It does not clear ALT or alter recording bytes to
manufacture the lower result. Prompt correctness is covered separately from this memory denominator.

No new production storage code was needed for this choice. The tradeoff is
explicit: compatibility retains roughly 19.7–42.4KB of ALT row payload after
prefix compaction for these recordings. A future exact image implementation
could be measured against this baseline, but it is not implemented here.
