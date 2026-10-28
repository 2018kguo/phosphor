# First comparison: idle sessions

This pre-release experiment cut erased-screen Go heap by 92% while preserving meaningful cells. Ghostty still led several parsing streams and had broader Unicode support. These are historical measurements, not current release claims.

Source: candidate `4216f73`, base `608a9b8`; idle-only baseline `d4bb0c0` has the original 880-byte Terminal layout. Ghostty `919dca5c848d43a489a4cb2e1feade016f0fa2db`, `-Demit-lib-vt -Doptimize=ReleaseFast`. No PGO. macOS: Apple M5 Pro; Linux: Colima arm64, four vCPUs, 6 GB RAM.

Build toolchains are recorded in [environment metadata](../environments.json).

## Memory

Fresh processes retain 1,024 terminals at 80×24; medians of three runs. Shared input, holder arrays and initialization are outside the baseline; two GCs precede each heap reading. All terminals remain live. Bytes of retained Go heap per terminal:

| State | Before idle changes | Candidate | Change |
| --- | ---: | ---: | ---: |
| Blank | 1,555 B | 1,229 B | -21% |
| Plain prompt | 2,840 B | 2,520 B | -11% |
| Colored prompt | 2,840 B | 2,515 B | -11% |
| Prompt with saved cursor | 2,835 B | 2,664 B | -6% |
| Filled screen erased back to prompt, then idle | 32,297 B | 2,521 B | -92% |
| Filled screen with meaningful cells retained | 32,291 B | 31,977 B | -1% |

Terminal shrank to 560 bytes through bounded CSI storage and saved cursors allocated on first use. `Compact()` released literal-zero rows, unused wrap flags and scratch; it kept history, styles, saved state, pending input and damage. New output can allocate again.

Process RSS per terminal includes native allocations; Ghostty's Go wrapper heap is not its footprint:

| State | macOS phosphor RSS | macOS Ghostty RSS | Linux phosphor RSS | Linux Ghostty RSS |
| --- | ---: | ---: | ---: | ---: |
| Plain prompt | 2.3 KiB | 30.5 KiB | 2.3 KiB | 18.0 KiB |
| Colored prompt | 2.2 KiB | 46.5 KiB | 2.3 KiB | 22.0 KiB |
| Filled then erased, after idle compaction | 5.3 KiB | 62.6 KiB | 4.1 KiB | 33.4 KiB |
| Meaningful full screen, after compaction | 33.2 KiB | 62.6 KiB | 33.1 KiB | 33.4 KiB |

These cases have no history. Allocator retention and lazily touched native pages explain why heap, RSS and native page estimates differ.

## Parsing and recovery

Five alternating rounds, identical bytes in 1 KiB chunks, 120×40. Median MiB per process CPU second includes GC, cgo and native work; wall samples remain in the raw data:

| Input | macOS baseline | macOS candidate | macOS Ghostty | Linux baseline | Linux candidate | Linux Ghostty |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Synthetic agent TUI | 427 | 464 | 504 | 428 | 448 | 487 |
| Scrolling logs | 417 | 550 | 912 | 412 | 553 | 896 |
| Progress bars | 1,211 | 1,323 | 1,115 | 1,162 | 1,335 | 1,102 |
| vim recording | 369 | 395 | 345 | 355 | 391 | 337 |
| less recording | 278 | 277 | 136 | 268 | 269 | 128 |
| nvim recording | 577 | 620 | 631 | 565 | 540 | 608 |
| nano recording | 429 | 446 | 576 | 412 | 423 | 563 |
| man recording | 377 | 384 | 436 | 359 | 369 | 419 |

Log CPU throughput rose about 32% on macOS /34% on Linux through reused history buffers, ASCII serialization and CSI digit runs. Shared-host noise limits small claims: [ten one-second nvim pairs](linux-nvim-focus.jsonl) gave about 565 to 609 MiB/CPU-s, contradicting the short Linux result. NEON expansion, word packing and four-cell unrolling were rejected for regressions elsewhere.

For a 120×40 screen with 627 history rows, median wall times over five macOS rounds:

| Operation | Baseline | Candidate phosphor | Ghostty |
| --- | ---: | ---: | ---: |
| Read visible text and selected metadata | 11.1 us | 5.6 us | 6.5 us |
| ANSI export with history | 14.7 us | 14.5 us | 215.6 us |
| Write persistence representation | 14.8 us | 14.7 us | 67.0 us |
| Restore persistence representation | 424.7 us | 410.8 us | 194.4 us |
| Persistence bytes | 98,427 | 98,427 | 317,035 |

Phosphor then persisted ANSI repaint; Ghostty used a stronger binary snapshot, so these restore costs compare different representations. Ghostty's VT export did not include both screens/saved cursors; adding parser continuation fixed partial-input cases only. Phosphor preserved unfinished CSI/OSC/DCS/ignored strings/UTF-8; all seven checkpoint edges passed in both engines. REP's prior character remained an ANSI limit.

## Compatibility and evidence

Selected probes: Phosphor 19/26, Ghostty 26/26 on both OSes; seven differences exercised unsupported graphemes. All eight input streams agreed on visible text, cursor and selected modes. The rejected grapheme prototype `experiment/grapheme` (`dac6b39`) lost a narrowed emoji after edge reflow; native REP did not repair it, and OSC 66 required downstream support.

Validation: macOS/Linux vet and full race; oracle race/vet and example builds; 96,795 byte-boundary continuations across seven recordings; component fuzz totals 65,102 continuation, 97,286 compaction, 108,059 saved-cursor, 122,206 packed-parser (two workers each). Review compared 25,000 public states. Five affected ptyd tests passed with race. Linux's first transfer included AppleDouble fuzz inputs; excluding those transfer artifacts produced the passing run.

The [ptyd patch](ptyd-idle-compaction.patch) reused its idle checkpoint timer and rechecked quiet/stopped state under the screen lock. It added no per-output work; no installed daemon or primary checkout changed.

Reproduce with the [driver](../../README.md): correctness at original 100×30 geometry, throughput at 120×40. Raw: [macOS memory](macos-memory.jsonl), [throughput](macos-throughput.jsonl), [state costs](macos-state.jsonl), [checks](macos-checks.jsonl), [Linux](linux.jsonl). Other files retain layouts, tests, fuzz and review output.
