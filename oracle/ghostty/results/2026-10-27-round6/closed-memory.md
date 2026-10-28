# What completed editors keep in memory

Correct 1049 exit keeps inactive ALT for later 47 re-entry. After Compact these completed recordings retained 22–47 KB of Go heap per terminal, versus 54–56 KB before. This is a larger state than a fresh prompt; no cold-screen codec or extra compression was added.

## Build and method

| Provenance | Value |
|---|---|
| Phosphor/driver source |`8d1511ce4f0d5b5951f4506818c4da14b4f4a3bc` (before the later parsing repair) |
| Native source |`46e9dc9a927b2e7e7cba79e6902c176127dbdf2c`, Zig 0.17.0, `zig build -Demit-lib-vt -Doptimize=ReleaseFast -j4` |
| Driver SHA256 |`bb7cd704d558ed4d0c7b374284af079a582623f5a1a6e8f0de75fbd1d94745bb` |
| Linked dylib SHA256 |`7700d7817d5c3c7107e5efb9cd165a1d9f16e877fe9f2cf886c528bfb707acbe` |
| Source manifest SHA256 |`5880bcd622156f80410cce12b6fd3bb75918e6526b86261a088326d4ce4973b4` |

The manifest hashes production root/internal Go, driver Go/C/header and both modules' go.mod/go.sum (no tests); sorted path→SHA256 JSON, compact separators, then SHA256. Files matched committed source before/after sampling; no uncommitted production bytes entered the build. macOS/arm64, Go 1.26.3, cgo/ghostty tag. Absolute native include/library/rpath was `/private/tmp/phosphor-gaps.ro32ng/ghostty`; another build path can change the binary hash.

[1024-terminal raw](closed-memory-1024.jsonl) /[256-terminal raw](closed-memory-256.jsonl):120 fresh-process samples =five recordings×two engines×plain/compacted×three repeats×two counts. Only a repeat label was added; order alternates and no samples/cells/replies were normalized or discarded.

Workloads `closed-{vim,nvim,nano,less,man}[-compact]` load actual completed rec-X.raw at 100×30 and reject unknown names or a still-active ALT. Both engines stay live while counters are read. Shared initialization is warmed; GC runs before baseline and twice after initialization, without forcing scavenging. Phosphor uses Compact, native uses GHOSTTY_TERMINAL_COMPRESSION_MODE_FULL. Native wrapper heap excludes C allocations; compare RSS. Every table gives median 3 B/terminal.

## Retained memory

1,024 terminals:

| Recording | Phosphor heap | Phosphor heap after Compact | Phosphor RSS | Phosphor RSS after Compact | Native RSS | Native RSS after FULL |
|---|---:|---:|---:|---:|---:|---:|
| vim | 56,198.5 | 46,958.9 | 57,520 | 56,768 | 82,288 | 82,352 |
| nvim | 56,211.2 | 47,316.2 | 57,424 | 55,392 | 82,592 | 82,608 |
| nano | 54,387.4 | 37,812.0 | 55,584 | 45,680 | 82,416 | 82,336 |
| less | 56,185.0 | 30,148.0 | 57,440 | 42,272 | 82,480 | 82,544 |
| man | 56,158.5 | 22,431.0 | 57,456 | 33,824 | 82,352 | 82,320 |

256 terminals:

| Recording | Phosphor heap | Phosphor heap after Compact | Phosphor RSS | Phosphor RSS after Compact | Native RSS | Native RSS after FULL |
|---|---:|---:|---:|---:|---:|---:|
| vim | 56,226.0 | 46,985.4 | 55,552 | 53,120 | 83,904 | 83,968 |
| nvim | 56,250.3 | 47,337.8 | 55,488 | 54,016 | 84,416 | 84,416 |
| nano | 54,447.2 | 37,878.7 | 53,952 | 44,288 | 83,648 | 83,584 |
| less | 56,240.5 | 30,169.8 | 55,424 | 42,368 | 84,160 | 84,032 |
| man | 56,208.5 | 22,457.9 | 55,616 | 36,288 | 83,648 | 83,712 |

Vim/nvim release more heap than RSS because allocator pages remain resident. Native FULL targets PRIMARY scrollback; these no-history runs keep ALT resident, so it changes little. Native page estimates are not RSS.

## Why more cannot simply be dropped

Read-only inspection of the 3000-position ALT grid distinguished literal spaces from empty cells. Prefix compaction retains every literal/styled space and removes only strict zero suffixes:

| Recording | ALT cell capacity before | ALT cell capacity after | Retained cells after | Zero cells in full grid | Default spaces | Styled spaces | Other cells |
|---|---:|---:|---:|---:|---:|---:|---:|
| vim |48,000B |41,024B |2,564 |443 |577 |335 |1,645 |
| nvim |48,000B |41,664B |2,604 |818 |337 |89 |1,756 |
| nano |46,400B |32,544B |2,034 |1,012 |251 |113 |1,624 |
| less |48,000B |26,032B |1,627 |1,373 |275 |0 |1,352 |
| man |48,000B |18,944B |1,184 |1,816 |281 |0 |903 |

Each grid also keeps 720 B row headers. Payload capacities exclude size-class rounding, Terminal, saves and tables; they are not measured heap/RSS. Some zeros remain inside prefixes (especially nvim); none of these five final grids has styled empty cells. Cell 16 B/Terminal 560 B remain unchanged.

Literal/styled spaces are observable. Complete-cell run counts are 1,941/2,038/1,866/1,567/1,069 in table order; a naive 24 B/run representation would exceed compacted prefixes for every recording. Sparse VT history is not an exact cold-screen image: replay/reflow and pool mutation/import require one safe owner. That complexity was deferred; retained ALT row payload is 19.7–42.4 KB.

`TestCursorLifetimeCheckpoint` /`TestNativeCursorLifetimeAndRepaint` exercise later 47 entry, snapshots and ANSI replay. The memory workload requires return to MAIN and never clears ALT or changes input to improve the number; prompt behavior is checked separately.

Build using the [driver instructions](../../README.md), then `PHOSPHOR_DIR=/path/to/phosphor BINARY -mode memory -engine phosphor -workload closed-vim-compact -n 1024`; repeat for both engines, workloads and counts. These hashes/numbers describe this pinned experiment, not a later rebuild.
