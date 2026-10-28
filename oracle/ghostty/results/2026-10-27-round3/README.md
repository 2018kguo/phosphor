# Third comparison: Unicode and exact checkpoints

This phase matched native scalar widths and added exact binary checkpoints. Dense restores became much faster than ANSI replay; snapshot creation cost more. Compatibility still had measured gaps. These are v0.3-era results, not the current release.

Runtime `15c8763fd294c892b37e00f93e56cdd47608cff1`; baseline v0.2.0 `7bc614a8503d40c48e0c09a62c170ef0a892cb0f`; Ghostty `46e9dc9a927b2e7e7cba79e6902c176127dbdf2c`, Zig 0.17.0 ReleaseFast. Apple M5 Pro, macOS 26.6.1/arm64, Go 1.26.3; Linux checks: four-CPU Docker VM, arm64, Go 1.26.10. Later evidence commits did not change measured runtime.

## Compatibility

The 72 named cases had 174 parity actions: baseline 100 passes→candidate 150. Three other actions retained identity, host-color and cluster-capacity differences. Trace exited 1; successful execution was not a passing result. Whole/one-byte/seven-byte/Linux outputs agreed after removing the chunk label: [baseline](traces-baseline.jsonl), [final](traces-final.jsonl), [byte](traces-byte.jsonl), [seven](traces-seven.jsonl), [Linux](traces-linux.jsonl). [Checks](checks.jsonl) retain 39/39 original probes per engine, eight agreeing streams,32 checkpoint continuations and eight exact-snapshot edges per engine; ANSI failures stay separate. See [contract](../../coverage.md).

Public CodepointWidth matched all 1,112,064 scalars using 109 deduplicated packed pages (11,328 shared bytes, no heap/per-terminal field). A range-search prototype slowed TUI parsing and was rejected. Regenerate with `go run ./internal/unicodegen -widths PATH` using [native widths](native-widths.jsonl).

The segmentation helper checked 766 Unicode 17 cases/1,224 boundaries:1,199 official matches plus 25 explicit native control-tailoring differences. Terminal dispatch was not fully covered. Clusters then capped at 32 UTF-8 bytes; mode 2027 began disabled. The translated suite passed 502, retained five differences and excluded 42. Remaining paired gaps covered underline color, mouse bits, width-reset tabs, empty/literal-space history and outside-origin ANSI cursor export.

## Time and memory

Five alternating 250 ms rounds,120×40,1 KiB writes. Median MiB/s; process CPU (including GC/native work) and wall samples: [throughput](throughput.jsonl).

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

Native led logs, synthetic TUI and nano. Small changes on a shared host are uncertain: seven 750 ms `man` rounds gave 467.6/472.0/440.2 MiB/s (baseline/candidate/native), with large Go outliers; [confirmation](man-confirmation.jsonl).

Fresh-process memory,1,024 terminals, three samples; median bytes per terminal. Use native RSS, not its wrapper heap: [raw](memory.jsonl).

| Workload | Phosphor Go heap | Phosphor RSS delta | Ghostty RSS delta |
|---|---:|---:|---:|
| prompt | 2515 | 2192 | 31200 |
| colored-prompt | 2520 | 2256 | 47648 |
| cleared-compact | 2526 | 5552 | 64128 |
| rich-cleared-compact | 2535 | 6800 | 80608 |

Prompt heap stayed near 2.5 KB; Terminal 560 B/cell 16 B and warm ASCII allocation counts stayed unchanged with mode 2027 off/on. RSS reflects page/allocator behavior, not fleet savings. Excluding URL/escape bytes from replay geometry separately reduced rich-history allocation 1,928,102→82,496 B/op (`BenchmarkGraphemeHistoryReplay`).

## Checkpoint tradeoff

Median microseconds over three 200 ms Go samples: [raw](snapshots.txt).

| Workload | Snapshot creation | ANSI creation | Exact restore | ANSI replay |
|---|---:|---:|---:|---:|
| Prompt | 0.35 | 0.44 | 0.56 | 0.71 |
| Sparse 200x50 | 1.00 | 1.28 | 1.17 | 1.46 |
| Full screen/history | 32.17 | 15.10 | 107.45 | 401.35 |
| Rich history | 42.55 | 18.72 | 164.96 | 1241.61 |

Full/rich restore was about 3.7×/7.5× faster; rich restore fell 731 KB/6,495 allocations→320 KB/2,070. Dense snapshot creation cost about twice ANSI creation. Reusable append allocated zero times. Full/rich snapshots were slightly smaller than ANSI, sparse/prompt substantially smaller.

Snapshots preserve both screens, all saved cursors, palette base/overrides, REP, tabs, modes, damage and unfinished input. Restore validates bounds/control payloads before replacing state, copies input and preserves callbacks without invoking them. Its checksum detects corruption, not forgery. Ptyd wrote outer checkpoint 2 and read legacy 1 ANSI checkpoints; old binaries could not read the new format. Its owner-only disk geometry contract stayed broader than the library defaults. Tests cross tail data, resize, restart and restore-decline checkpoints.

Validation: Go 1.24 full suite; macOS current-Go race/vet, examples and oracle race/vet; identical native macOS/Linux outputs. Linux's corrected tracked-file archive also passed the initially omitted libvterm fixture race test. Fuzz totals:4,360,186 malformed snapshots/30 s,812,677 continuation/45 s,865,377 grapheme transitions/30 s; CI runs each target 30 s. Unicode generation reproduced both tables. Review fixed malformed breakout, accepted-geometry rejection, version/body mismatch and modifier joining across intervening scalars; pool tests compare resolved values, not IDs. No installed daemon changed. Use the [driver](../../README.md) to rerun.
