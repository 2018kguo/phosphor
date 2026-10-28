# Cold-prefix allocation measurements

Measured on Apple M5 Pro, macOS arm64, Go 1.26.3. The before core is `b4637e58a4145035f980116a3ba38d5c8927abb0`; the after core adds cold main-screen prefixes, geometric capacity growth, dense alternate-screen allocation and default-background erasure limited to owned cells. Both binaries use the same benchmark driver and recordings. The before checkout changes only driver selectors and adds the identical test-only recording benchmark.

No production GC or scavenging policy changed. The existing measurement driver collects before and after in both builds. RSS below is incremental process RSS divided by 1024 terminals, not total process size. Every memory sample starts a fresh process; prompt cases assert the expected prompt and zero history.

The generic copy/clear helper remains inlineable (cost 34), as does the prefix wrapper (75) and blank-row operation (76). Capacity policy stays in an outlined cold helper. Main-screen writes allocate only their used prefix; alternate-screen redraws allocate a full row once. The rec-man benchmark retains 41 allocations and about 83 KB per repeated recording, matching before.

Default-background erasure scans for actual changes only when a tracked row is clean. Already-dirty rows retain the fast clear path; clearing a wrap flag marks the row dirty. Damage therefore does not depend on whether a zero tail owns storage.

## Default-runtime throughput

Five 250 ms samples alternate before, after and pinned Ghostty with `oracle/ghostty/ab.sh`, 1024-byte reads. All samples remain in the raw file, including severe scheduling outliers. Arrays are MiB/s in sample order; deltas are ratios of medians. This run cannot establish a clean timing gate for less/nvim: their wall rates vary by more than tenfold. It is retained as evidence, not filtered into a flattering result.

| Stream | Before, five samples | After, five samples | Wall median change | CPU median change |
|---|---|---|---:|---:|
| claude-synthetic | 453.5, 433.5, 446.9, 434.3, 434.2 | 433.2, 449.2, 451.8, 451.8, 440.1 | +3.4% | +3.4% |
| logs | 523.1, 539.7, 538.0, 508.4, 485.6 | 485.6, 516.5, 511.4, 466.7, 506.1 | -3.3% | -3.3% |
| progress | 1263.2, 1255.7, 1208.8, 1231.5, 1232.0 | 1230.4, 1252.9, 1264.4, 1211.8, 1263.5 | +1.7% | +1.7% |
| vim-recording | 364.4, 371.8, 367.2, 358.5, 366.6 | 367.7, 367.4, 368.7, 364.6, 366.9 | +0.2% | +0.2% |
| less-recording | 273.3, 234.5, 219.7, 216.0, 20.3 | 144.5, 234.7, 205.5, 61.7, 122.1 | -34.3% | -6.4% |
| rec-nvim-recording | 424.4, 330.4, 564.9, 523.1, 531.5 | 119.2, 100.3, 361.9, 524.4, 541.4 | -30.8% | -4.0% |
| rec-nano-recording | 348.3, 403.0, 398.5, 393.6, 396.8 | 362.0, 393.3, 394.3, 394.8, 382.3 | -0.9% | -1.3% |
| rec-man-recording | 380.6, 364.8, 350.9, 350.2, 354.1 | 360.0, 330.1, 352.5, 345.1, 313.6 | -2.5% | -10.0% |

## Longer controlled checks

The three questionable recordings were rerun as five alternating before/after pairs, one second each, with GOMAXPROCS=1 for both binaries and no native third process. This supplements rather than replaces the default-runtime run. All controlled wall/CPU medians differ by less than 2.2%; together with unchanged dense-recording allocations, the data supports approximately flat active performance. It does not establish a universal speedup.

| Stream | Before, five samples | After, five samples | Wall median change | CPU median change |
|---|---|---|---:|---:|
| less-recording | 282.5, 273.8, 269.6, 271.9, 247.4 | 276.7, 270.6, 271.0, 274.9, 248.8 | -0.3% | -0.5% |
| rec-nvim-recording | 668.5, 651.1, 634.5, 644.8, 657.1 | 651.3, 653.9, 630.0, 636.3, 642.4 | -1.3% | -1.3% |
| rec-man-recording | 346.1, 336.7, 336.6, 326.1, 336.1 | 336.9, 330.8, 325.9, 329.5, 323.5 | -2.1% | -2.1% |

## Idle memory

Five fresh-process samples per workload, 1024 terminals per sample. RSS arrays and heap medians are bytes per terminal. `-erase` appends the default-background EL shells commonly issue. A `-compact` suffix additionally calls Compact; unsuffixed erase cases prove that erasing an already implicit tail does not allocate a full row.

| Workload | Before RSS, five samples | After RSS, five samples | Before / after heap median |
|---|---|---|---:|
| prompt-compact | 2544, 2400, 2640, 2496, 2528 | 1008, 1056, 1408, 1216, 976 | 1443 / 1443 |
| colored-prompt-compact | 2544, 2384, 2624, 2736, 2496 | 1600, 1648, 1808, 1536, 1824 | 1448 / 1443 |
| prompt-erase-compact | 2496, 2368, 2624, 2560, 2464 | 1216, 960, 1152, 1120, 1184 | 1438 / 1443 |
| colored-prompt-erase-compact | 2624, 2448, 2544, 2544, 2528 | 1632, 1568, 2048, 1520, 1424 | 1443 / 1443 |
| prompt-erase | 2800, 2368, 2544, 2320, 2336 | 1056, 1136, 1296, 1088, 1072 | 2515 / 1443 |
| colored-prompt-erase | 2480, 2256, 2288, 2272, 2208 | 1328, 1440, 1632, 1328, 1280 | 2510 / 1523 |

## Reproduce

Build the same `oracle/ghostty` driver against both cores using the pinned Ghostty source and flags in its README. Then run:

```sh
PHOSPHOR_DIR=/path/to/checkout COMPARE_DURATION=250ms bash oracle/ghostty/ab.sh BEFORE AFTER throughput.jsonl
AFTER -mode memory -engine phosphor -workload prompt-erase-compact -n 1024
GOMAXPROCS=1 PHOSPHOR_DIR=/path/to/checkout COMPARE_ENGINE=phosphor COMPARE_ROUNDS=1 COMPARE_STREAM=less-recording COMPARE_CHUNK=1024 AFTER -mode throughput -duration 1s
```

Run memory and controlled timing commands as separate processes, alternate binary order, and repeat five times for each workload in the tables. The raw default-runtime throughput file also preserves native Ghostty samples.

Measured binary SHA-256:

- prefix_before: `de3b37c8eead44983536fc4e04b0d8a09d39073304f69ffcd0bfe2436af6798f`.
- cold_after: `9683e9b4ee8259b7ebe3551bc3b34af41fab8f4bf85c6d84f20c9c8af8bb7b35`.

## Correctness

Full `go test ./...` and `go vet ./...` pass. The cold-prefix fuzzer compares real production code using compact prefixes against the same code with full-width storage; the 20-second run completed 100,556 executions. Targeted cases cover cold Unicode, byte-sized growth, insert mode, colored scrolling, default EL, half-wide erasure and linked/underline/protected styles. Exact damage regressions cover allocated zero tails versus snapshot-restored/compacted prefixes, with and without wrap state. Native byte-by-byte traces cover prompt erasure, wide styles, alternate-screen insertion and scrolling. Warm forty-row ASCII logs and TUI frames remain zero allocation. Snapshot codecs and fixed Terminal/cell layouts are unchanged.
