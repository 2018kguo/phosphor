# More idle storage savings and scroll-and-clear support

This round keeps phosphor's focus on many inexpensive sessions. Explicit
idle compaction now frees stale hyperlink and combining-character tables,
including invalid saved-cursor storage. Meaningful cells, retained history,
live saved pens, pending input and REP's last character remain live. It adds
no Terminal fields or work to ordinary output.

The baseline is the previous round, `49f49b8`; runtime changes end at
`47ad94e`. Timings and memory were measured at `f6046d4`; the final
change collects unused lookup entries after resize and does not run on
those measured paths. Ghostty remains pinned to
`919dca5c848d43a489a4cb2e1feade016f0fa2db`, built with Zig 0.17.0 and
`-Demit-lib-vt -Doptimize=ReleaseFast`. The
[first round](../2026-10-27/README.md) records the comparison with v0.1.0.

## What changed

- `Compact()` marks live pool references before rebuilding a table. Small
  unchanged live tables stay in place without allocation. It frees the
  optional extra state only when every field is unused.
- Scroll-and-clear (`CSI 22 J`) moves written rows into bounded history,
  preserving the cursor's pen and input modes. It follows native Ghostty's
  behavior for margins, protection, trailing empty rows and the alternate
  screen.
- SGR serialization reuses its existing attribute-code table rather than
  copying 192 bytes for every style change. On the measured Go compiler,
  the function's stack frame shrinks from 336 to 144 bytes.
- History imported during resize keeps stable pool IDs until all working
  rows are attached. Review found that freeing the old pools exposed this
  detached-row bug; both combining-character and hyperlink reproductions
  panicked before the fix and preserve their cells afterward.
- Resize collects unused entries after installing all rows. Repeated idle
  grow/shrink cycles otherwise kept adding entries without new output;
  the new 100-cycle regression keeps storage bounded and content intact.

Ptyd uses `Compact()` when its existing idle checkpoint is due, under the
screen lock after a fresh stopped/quiet check. It adds no timer. The stored
checkpoint test reloads the file and continues an unfinished CSI sequence.

## Measurements

Median retained memory per emulator, with Go heap and process RSS shown
separately:

| State | Prior phosphor heap | Current phosphor heap | Current phosphor RSS | Ghostty RSS |
| --- | ---: | ---: | ---: | ---: |
| macOS prompt | 2,510 B | 2,515 B | 2.1 KiB | 30.6 KiB |
| macOS erased ASCII screen, compacted | 2,518 B | 2,516 B | 4.9 KiB | 62.7 KiB |
| macOS erased rich screen, compacted | 115,732 B | 2,563 B | 8.9 KiB | 78.8 KiB |
| macOS meaningful full screen, compacted | 31,977 B | 31,971 B | 33.3 KiB | 62.6 KiB |
| Linux prompt | 2,509 B | 2,509 B | 2.2 KiB | 18.1 KiB |
| Linux erased ASCII screen, compacted | 2,510 B | 2,510 B | 5.0 KiB | 33.5 KiB |
| Linux erased rich screen, compacted | 115,699 B | 2,531 B | 4.5 KiB | 51.3 KiB |
| Linux meaningful full screen, compacted | 31,950 B | 31,950 B | 33.2 KiB | 33.4 KiB |

The rich case saves about 98% of retained Go heap relative to the prior
compaction implementation. Without calling Compact it retains about
119 KB in both versions. Ordinary prompt and meaningful full-screen heap
are unchanged within measurement noise. Terminal storage remains 560 bytes.

Each memory sample retains 1,024 emulators in a fresh process; three
alternating repetitions are used. Inputs, holder arrays and shared
initialization are outside the baseline, with two GCs before both heap
readings. Cross-engine comparisons use process RSS, including native
allocations. Ghostty's Go heap is not its footprint.

The metadata-heavy scenario writes 48 different 2 KiB hyperlink targets and
combining characters on one row, then erases back to the prompt. It has no
history. This isolates stale metadata; it is not a typical shell prompt.

Throughput uses five alternating rounds, 1 KiB writes and 120x40 geometry.
CPU time includes GC, cgo and native work. The agent stream is synthetic;
editor and pager streams are recordings. State costs use a 120x40 screen
with 627 retained history rows. Ghostty's binary snapshot is stronger than
its ANSI export, so persistence size and restore costs compare different
representations.

Median MiB per process CPU second:

| Stream | macOS prior | macOS current | macOS Ghostty | Linux prior | Linux current | Linux Ghostty |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Synthetic agent TUI | 438 | 449 | 498 | 338 | 347 | 346 |
| Logs | 529 | 545 | 912 | 470 | 451 | 699 |
| Progress | 1,280 | 1,284 | 1,121 | 1,055 | 1,082 | 871 |
| vim | 392 | 394 | 340 | 321 | 322 | 264 |
| less | 274 | 275 | 134 | 225 | 225 | 103 |
| nvim | 616 | 615 | 626 | 404 | 400 | 431 |
| nano | 431 | 429 | 574 | 259 | 248 | 430 |
| man | 245 | 275 | 374 | 218 | 236 | 324 |

These shared hosts show time trends and noisy small differences. For
example, Linux log medians look 4% slower while the median paired-round
ratio favors the candidate by about 1%; a changing host affected the
separate medians. The raw samples remain visible. This round does not
establish a universal parsing speedup; Ghostty still leads several streams.

Median wall time for state operations, in microseconds:

| Operation | macOS prior | macOS current | macOS Ghostty | Linux prior | Linux current | Linux Ghostty |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Read visible text and metadata | 6.42 | 6.31 | 7.44 | 6.53 | 6.13 | 7.68 |
| ANSI screen export | 9.87 | 9.46 | 15.33 | 10.23 | 9.28 | 15.18 |
| ANSI export with history | 18.79 | 18.27 | 250.59 | 20.49 | 17.25 | 235.93 |
| Persistence encode | 19.54 | 18.73 | 79.29 | 17.78 | 17.53 | 163.76 |
| Persistence restore | 457.98 | 461.13 | 229.21 | 467.80 | 432.44 | 301.26 |

Persistence stays at 98,427 bytes for phosphor and 317,035 bytes for the
stronger native snapshot. Ghostty still restores that representation faster.
The machines are an Apple M5 Pro, macOS/arm64 with Go 1.26.3, and Colima
Linux/arm64 with four vCPUs, 5 GiB container memory and Go 1.26.10. These
measure engine work, without a font/GPU renderer or whole-daemon forecast.

## Compatibility and limits

The same 39 selected feature probes pass 22/39 on the previous phosphor
head, 32/39 now, and 39/39 in Ghostty. All 13 new scroll-and-clear probes
match native behavior; ten failed on the old head. These are feature
checks, not a compliance score. The seven remaining differences exercise
unsupported grapheme clustering.

The portable ANSI repaint still omits REP's prior character and clamps a
live cursor outside an enabled origin region. The new exact cursor probe
exposes the latter in both engines' ANSI exporters; Ghostty's binary
snapshot preserves it. Tests record those limits rather than treating a
normalized current view as proof of future-output equivalence. Broad
grapheme changes remain outside the released implementation.

## Review and verification

Independent read-only reviewers checked the runtime changes, live pool
references and ptyd's lock order. One reviewer found the detached history
import blocker; it was fixed before release. The regression covers three
history batches across 12 cluster/link/mixed, height/width and main/alternate
screen cases, every public cell, saved pens, future collection and further
resizes. ASCII history import still creates no optional lookup state.

Full vet and race suites passed on macOS/arm64 (159.061 s) and Linux/arm64
(169.283 s) at `f6046d4`, including the oracle module on both platforms.
The final end-of-resize cleanup at `47ad94e` passes the full ordinary
suite on both platforms and affected race tests (4.841 s macOS, 5.333 s
Linux). Two independent reviewers found no blocker in that final change;
the repeat-resize trace retains zero cluster entries after 1, 10 and 100
cycles, while retaining its history. Release checks run the full workflow
on the final head before tagging.

Component checks include 37,493 compaction fuzz executions, 82,906 token
executions, 7,133 stream executions with scroll-and-clear, and 28,266
continuation executions with the race detector. A frozen baseline encoder
matched all 524,288 tested attribute/color combinations. These component
runs supplement the final integrated checks.

The zero-attribute serialization guard was rejected after slower log
samples. An always-rebuild pool collector was rejected because unchanged
live tables allocated on every compaction. The shared-table timing
experiments included contradictory history samples; the retained change
removes known repeated work, without claiming a universal speedup. See
[performance findings](performance-findings.md) and
[idle-memory findings](idle-memory/findings.md).

## Task context

The user requested another head-to-head improvement round, with emphasis
on idle memory and replay, followed by phosphor and ptyd releases. Codex
`/root` integrated the work, added the idle-resize bound and owns final
checks and publication.
`ascii_performance` supplied the SGR change and found the resize blocker;
`correctness_probes` supplied pool compaction and its resize fix;
`grapheme_support` supplied scroll-and-clear and rejected the unproven
combining-margin change. Independent reviewers also crossed those scopes.
After a harness restart, `resize_fix_review` and `release_review` reviewed
the final fix again. This is a task summary, not a full conversation
transcript; private harness logs and unrelated workspace context are not
published.
