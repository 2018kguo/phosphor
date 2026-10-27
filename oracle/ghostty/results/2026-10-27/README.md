# Idle memory and the libghostty comparison

Ptyd needs many terminal sessions to remain cheap while idle. Keep phosphor
for that workload: the changes reduce fixed session storage and release
erased screen rows when the caller declares the terminal idle. Ghostty
remains the control for compatibility and performance; it still wins several
parsing workloads and has broader Unicode support.

These are local changes, not a published phosphor release or a deployed ptyd
upgrade. Runtime code was tested at `4216f73`; the base is `608a9b8`.
Ghostty is pinned to `919dca5c848d43a489a4cb2e1feade016f0fa2db`, built with
Zig 0.17.0, `-Demit-lib-vt -Doptimize=ReleaseFast`.

## Idle memory

Each scenario retains 1,024 independent 80x24 emulators in a fresh process,
with three repetitions. Inputs, the holder array and shared initialization
are outside the baseline. Two GCs precede both Go heap readings, and all
terminals stay live until after measurement.

Median retained Go heap per phosphor emulator on macOS:

| State | Before idle changes | Current | Change |
| --- | ---: | ---: | ---: |
| Blank | 1,555 B | 1,229 B | -21% |
| Plain prompt | 2,840 B | 2,520 B | -11% |
| Colored prompt | 2,840 B | 2,515 B | -11% |
| Prompt with saved cursor | 2,835 B | 2,664 B | -6% |
| Filled screen erased back to prompt, then idle | 32,297 B | 2,521 B | -92% |
| Filled screen with meaningful cells retained | 32,291 B | 31,977 B | -1% |

The idle baseline is `d4bb0c0`, immediately before the memory changes; it has
the same 880-byte Terminal layout as v0.1.0. Current Terminal storage is
560 bytes. CSI parameters use bounded 32-bit values and a separator bitset;
saved cursors allocate one block only on first use. `Compact()` frees only
literal-zero rows, empty wrap flags and disposable scratch buffers. It
preserves scrollback, meaningful cells, styles, saved cursors, unfinished
input and damage. Further output can allocate those buffers again.

Cross-engine comparisons use **process RSS**, including native memory.
Ghostty's tiny Go heap reading is not used as its memory footprint:

| State | macOS phosphor RSS | macOS Ghostty RSS | Linux phosphor RSS | Linux Ghostty RSS |
| --- | ---: | ---: | ---: | ---: |
| Plain prompt | 2.3 KiB | 30.5 KiB | 2.3 KiB | 18.0 KiB |
| Colored prompt | 2.2 KiB | 46.5 KiB | 2.3 KiB | 22.0 KiB |
| Filled then erased, after idle compaction | 5.3 KiB | 62.6 KiB | 4.1 KiB | 33.4 KiB |
| Meaningful full screen, after compaction | 33.2 KiB | 62.6 KiB | 33.1 KiB | 33.4 KiB |

RSS figures are process deltas divided by session count. Freed Go heap and
RSS do not fall by identical amounts because allocators and the OS retain
pages. Ghostty's page accounting also differs from RSS because reserved
pages can be touched lazily. These tests contain no history, so differing
history-limit policies do not explain the idle comparison.

## Active parsing

Five alternating rounds feed identical bytes in 1 KiB chunks at 120x40.
Numbers are median **MiB per process CPU second**, including GC, cgo and
native work. Wall-time measurements are retained in the raw files.

| Input | macOS v0.1.0 | macOS current | macOS Ghostty | Linux v0.1.0 | Linux current | Linux Ghostty |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Synthetic agent TUI | 427 | 464 | 504 | 428 | 448 | 487 |
| Scrolling logs | 417 | 550 | 912 | 412 | 553 | 896 |
| Progress bars | 1,211 | 1,323 | 1,115 | 1,162 | 1,335 | 1,102 |
| vim recording | 369 | 395 | 345 | 355 | 391 | 337 |
| less recording | 278 | 277 | 136 | 268 | 269 | 128 |
| nvim recording | 577 | 620 | 631 | 565 | 540 | 608 |
| nano recording | 429 | 446 | 576 | 412 | 423 | 563 |
| man recording | 377 | 384 | 436 | 359 | 369 | 419 |

Log parsing improves about 32% on macOS and 34% on Linux. Scrollback reuses
evicted line buffers; serialization copies ASCII runs in bulk; ASCII styles
stay in registers; CSI digit runs avoid dispatching each digit separately.
Small differences on these shared hosts need caution. This is engine-level
work, not whole-daemon latency or throughput.

A [focused nvim repeat](linux-nvim-focus.jsonl), ten alternating pairs with
one-second samples, measured about 565 MiB/CPU-s for v0.1.0 and 609 for the
current code. The shorter Linux sample above therefore does not establish
a sustained nvim regression.

The ASCII experiments also tried NEON expansion, a combined-word serializer,
and four-cell unrolling. Each traded small wins for regressions in another
stream, so none was kept.

## Reading and recovery

For a 120x40 screen retaining 627 history rows of synthetic agent output,
five rounds on macOS gave these median wall times:

| Operation | v0.1.0 | Current phosphor | Ghostty |
| --- | ---: | ---: | ---: |
| Read visible text and selected metadata | 11.1 us | 5.6 us | 6.5 us |
| ANSI export with history | 14.7 us | 14.5 us | 215.6 us |
| Write persistence representation | 14.8 us | 14.7 us | 67.0 us |
| Restore persistence representation | 424.7 us | 410.8 us | 194.4 us |
| Persistence bytes | 98,427 | 98,427 | 317,035 |

Phosphor persists VT repaint bytes; Ghostty uses its stronger binary
snapshot. They are different representations. Ghostty's active-screen VT
formatter is not a complete replacement for phosphor's attach repaint:
it does not export both screens and saved cursor state in one call.
Appending Ghostty's continuation API fixes the partial-input cases but
does not add those missing fields.

Phosphor now preserves unfinished CSI, OSC, DCS, ignored strings and UTF-8
across repaints. All seven focused checkpoint edges pass in both engines.
This is not universal future-output equivalence: phosphor still omits
REP's previous character (`CSI b`), as documented in the library README.

## Compatibility and verification

The explicit semantic probes pass 19/26 in phosphor and 26/26 in Ghostty on
both operating systems. The seven differences intentionally exercise
phosphor's unsupported grapheme features; this is not a compliance score.
Visible text, cursor and selected modes agree on all eight input streams.

The grapheme prototype stays on `experiment/grapheme` (`dac6b39`). It does
not enter this candidate: its scalar ANSI repaint loses a narrowed emoji
after reflow places it at the right edge. REP did not preserve that narrowed
emoji in the native probe, and OSC 66 would require downstream protocol
support. The prototype retains a failing regression instead of weakening
the exact replay oracle.

Validation completed on the runtime head:

- `go vet ./...` and `go test -race ./...` on macOS/arm64 and Linux/arm64.
- Oracle module vet/race tests and example module vet/build on macOS.
- Every byte boundary in seven recordings: 96,795 checkpoint continuations.
- Bounded component fuzz runs: 65,102 continuation, 97,286 compaction,
  108,059 saved-cursor and 122,206 packed-parser executions, two workers each.
- Independent review compared 25,000 public-state observations with the
  baseline across tight history budgets, styles, links, resize and restart.
- Five affected ptyd checkpoint/screen tests pass with the race detector
  against the integrated local phosphor version.

The ptyd integration is saved as [a reviewable patch](ptyd-idle-compaction.patch).
It uses the existing idle checkpoint timer and rechecks quiet/stopped state
under the screen lock before compaction and checkpoint I/O. It adds no
timer or per-output work. The installed daemon and primary ptyd checkout
were not changed. Apply it only with a phosphor version containing Compact;
the pilot used a temporary Go modfile replacement.

## Reproduce and inspect

Use the [comparison driver](../../README.md). The hardware is an Apple M5
Pro with Go 1.26.3 on macOS. Linux is Colima arm64, four vCPUs and 6 GB RAM,
running Go 1.26.10. Both libraries use ordinary optimized builds without PGO.
Recording correctness uses original 100x30 geometry; throughput uses the
existing benchmark convention of 120x40. The agent TUI is deterministic
synthetic output, while editor and pager inputs are recordings.

Raw evidence: [macOS memory](macos-memory.jsonl),
[macOS throughput](macos-throughput.jsonl), [macOS state costs](macos-state.jsonl),
[macOS checks](macos-checks.jsonl), and [Linux measurements](linux.jsonl).
The remaining files hold test/fuzz output, before/after layouts and review
observations. Linux's first transfer included AppleDouble metadata as fuzz
inputs; excluding those transfer artifacts produced the clean passing run.
