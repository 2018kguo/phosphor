# Second comparison: release unused metadata

Could idle compaction also reclaim dead links and combining characters? Yes: the erased rich-screen case saved about 98% of Go heap, without changing ordinary prompt/full-screen storage. Scroll-and-clear (`CSI22J`) also gained native-compatible behavior. This phase predates exact binary checkpoints.

Baseline `49f49b8`; measured `f6046d4`; final runtime `47ad94e` adds post-resize pool collection outside the measured paths. Ghostty `919dca5c848d43a489a4cb2e1feade016f0fa2db`, Zig 0.17.0, `-Demit-lib-vt -Doptimize=ReleaseFast`. Compare the [first phase](../2026-10-27/README.md) for v0.1.0 results.

## Retained memory

Medians of three alternating fresh-process runs holding 1,024 terminals; shared inputs/holder arrays excluded, two GCs before each heap reading. Rich input writes 48 distinct 2 KiB URLs and combining characters, erases to a prompt and keeps no history. It isolates metadata churn, not a typical prompt. Without Compact both versions retained about 119 KB.

| State | Prior phosphor heap | Candidate phosphor heap | Candidate phosphor RSS | Ghostty RSS |
| --- | ---: | ---: | ---: | ---: |
| macOS prompt | 2,510 B | 2,515 B | 2.1 KiB | 30.6 KiB |
| macOS erased ASCII screen, compacted | 2,518 B | 2,516 B | 4.9 KiB | 62.7 KiB |
| macOS erased rich screen, compacted | 115,732 B | 2,563 B | 8.9 KiB | 78.8 KiB |
| macOS meaningful full screen, compacted | 31,977 B | 31,971 B | 33.3 KiB | 62.6 KiB |
| Linux prompt | 2,509 B | 2,509 B | 2.2 KiB | 18.1 KiB |
| Linux erased ASCII screen, compacted | 2,510 B | 2,510 B | 5.0 KiB | 33.5 KiB |
| Linux erased rich screen, compacted | 115,699 B | 2,531 B | 4.5 KiB | 51.3 KiB |
| Linux meaningful full screen, compacted | 31,950 B | 31,950 B | 33.2 KiB | 33.4 KiB |

Small unchanged live pools stay in place without allocation; large pools need a transient mark set. The same live walkers cover both screens, pens, saved cursors and REP. History already contains VT bytes. Terminal stays 560 bytes; no Write hook was added.

Earlier pool prototypes held 128 terminals per fresh process; medians of three Go-heap deltas, two GCs, one warmup terminal, base `49f49b8`:

| Trace | Base | Always rebuild | Mark before rebuild |
| --- | ---: | ---: | ---: |
| links | 151,216.125 | 2,497.000 | 2,497.125 |
| unicode | 5,432.250 | 2,496.125 | 2,496.125 |
| mixed-live | 5,617.125 | 2,785.000 | 2,785.000 |
| closed-1049 | 154,185.250 | 5,072.250 | 2,496.250 |

`links`/`unicode` churn 64 unique URLs (2 KiB each)/clusters then erase; `mixed-live` keeps one saved pair; `closed-1049` churns both, enters/exits 1049 with the last URL active, then closes/erases. Always-rebuild allocated twice on every unchanged Compact; mark-before-rebuild kept zero allocations through 256 live entries. The mixed-live URL already triggers active collection, so its reduction is mostly cluster storage. See [prototype samples](idle-memory/pool-memory-prototypes.jsonl), [counterexample](idle-memory/rebuild-noop-counterexample.txt) and [fuzz log](idle-memory/pool-compact-fuzz.txt).

## Parsing and state costs

Five alternating rounds, 1 KiB writes, 120×40; median MiB per process CPU second includes GC/cgo/native work. Apple M5 Pro, Go 1.26.3 macOS/arm64; Colima Linux/arm64, four vCPUs, 5 GiB container memory, Go 1.26.10.

| Stream | macOS prior | macOS candidate | macOS Ghostty | Linux prior | Linux candidate | Linux Ghostty |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Synthetic agent TUI | 438 | 449 | 498 | 338 | 347 | 346 |
| Logs | 529 | 545 | 912 | 470 | 451 | 699 |
| Progress | 1,280 | 1,284 | 1,121 | 1,055 | 1,082 | 871 |
| vim | 392 | 394 | 340 | 321 | 322 | 264 |
| less | 274 | 275 | 134 | 225 | 225 | 103 |
| nvim | 616 | 615 | 626 | 404 | 400 | 431 |
| nano | 431 | 429 | 574 | 259 | 248 | 430 |
| man | 245 | 275 | 374 | 218 | 236 | 324 |

Host drift matters: Linux logs' separate medians suggest -4%, while paired rounds favor +1%. These samples do not establish a universal speedup. State costs below use 627 history rows; median wall microseconds:

| Operation | macOS prior | macOS candidate | macOS Ghostty | Linux prior | Linux candidate | Linux Ghostty |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Read visible text and metadata | 6.42 | 6.31 | 7.44 | 6.53 | 6.13 | 7.68 |
| ANSI screen export | 9.87 | 9.46 | 15.33 | 10.23 | 9.28 | 15.18 |
| ANSI export with history | 18.79 | 18.27 | 250.59 | 20.49 | 17.25 | 235.93 |
| Persistence encode | 19.54 | 18.73 | 79.29 | 17.78 | 17.53 | 163.76 |
| Persistence restore | 457.98 | 461.13 | 229.21 | 467.80 | 432.44 | 301.26 |

Persistence remained 98,427 bytes of Phosphor ANSI versus 317,035 bytes of native binary snapshot. Native restored its stronger representation faster.

The retained SGR-table change `0d6b7e872bca679be95bbac95e420d000c4450f4` (base `49f49b8`) removed a 192-byte copy; appendSGR's frame shrank 336→144 bytes. The baseline profile attributed 37% of log CPU to appendRow, 16% printASCII, 10% rotation, 8% appendSGR. Six rotating 200 ms control/table/guard rounds gave these baseline/table speed ratios (>1 favors the table):

| Work | Ratio |
| --- | ---: |
| vim | 1.104 |
| less | 0.990 |
| synthetic Claude output | 0.996 |
| scrolling logs | 1.041 |
| progress bars | 1.068 |
| visible repaint | 1.065 |
| repaint with history | 0.980 |
| Lines, unchanged control | 0.964 |
| ScrollbackLines, unchanged control | 1.108 |
| Replay | 1.017 |

Unchanged controls moved too. Initial CPU pairs suggested 2.4% less log CPU /2.2% less visible-repaint CPU but 8.5% worse history repaint; a balanced four-pair repeat later gave 3% lower CPU /5% lower wall median, all CPU pairs +0.2–5.1%. The extra guard was rejected after ~8% worse log timing. Retain the removal of known work, not a universal timing claim: [three rotating logs](perf-final-baseline.txt), [table](perf-final-sgr-table.txt), [rejected guard](perf-final-sgr-guard.txt), [CPU counter-evidence](perf-cpu.jsonl), [repeat](perf-cpu-repeat.jsonl).

## Correctness and the resize fix

Selected probes improved 22/39→32/39 versus native 39/39; all 13 new scroll-clear cases match (ten formerly failed). Seven remaining differences were graphemes. ANSI still omitted REP and clamped an outside-origin cursor; both exporters exposed the latter, native Snapshot preserved it.

Review of Phosphor `49f49b8..c8ea2cd` (`40a5881` only changed a driver executable bit) and ptyd `af36c18` to the working tree based on `231304c` found a crash: detached resize imports were invisible to collection after Compact. Public 100×4→100×5 witnesses with 400 entries panicked at cluster index 36/length 36 or link 4/length 4; the focused red case also reached index 138/length 138. [Before/after pool](history-pools-before.txt) /[after](history-pools-after.txt) and [link](history-links-before.txt) /[after](history-links-after.txt) outputs retain the evidence.

Fix `f936a21` after `e4d756c` counts imported references (including repeats), temporarily raises existing collection thresholds, then restores them after attaching rows. It keeps the uint16 link bound; plain ASCII imports allocate no extra state. Twelve cluster/link/mixed ×height/width ×main/ALT cases cross three history batches, saved pens, future collection and resizes: [red](idle-memory/resize-import-red.txt), [green](idle-memory/resize-import-green.txt). Final resize cleanup keeps zero unused entries after 1/10/100 idle cycles. Lock review found no reverse overlapping `t.mu`/`t.scrMu` edge in ptyd.

Full vet/race passed at `f6046d4`: macOS 159.061 s /Linux 169.283 s, including oracle. Final `47ad94e` passed both ordinary suites and affected races (4.841/5.333 s); two reviewers found no blocker. Component totals: 37,493 Compact (30 s, two workers), 82,906 tokens (20 s, two workers; [log](perf-fuzz.txt)), 7,133 stream, 28,266 race continuation; 524,288 attribute/color encodings matched the frozen encoder. The focused import tests took 0.524 s; earlier review tests 0.338 s and ptyd idle checkpoint 0.206 s. These supplement integrated checks, not release CI proof.

Ptyd reused its idle checkpoint timer, reloaded the stored checkpoint and continued unfinished CSI. No timer or ordinary-output work was added. Raw integrated evidence: [macOS memory](macos-memory.jsonl), [throughput](macos-throughput.jsonl), [state](macos-state.jsonl), [checks](final-checks.jsonl), [tests](final-tests.txt), [Linux](linux.jsonl) /[final tests](linux-final-tests.txt). Use the [driver](../../README.md) to rerun; measurements are historical.
