# Cold prefixes: reduce real prompt memory

Could avoiding the initial full row lower RSS as well as Compact's retained heap? Yes: plain prompt RSS fell 2528→1056 B/terminal; colored 2544→1648. Shell-style default-background EL kept the saving. Active performance was approximately flat in longer checks, not universally faster.

Before `b4637e58a4145035f980116a3ba38d5c8927abb0`; after adds cold MAIN prefixes, geometric capacity growth, dense ALT allocation and erasure within owned cells. Same driver/recordings; baseline overlay changes only selectors/test benchmark. Apple M5 Pro, macOS/arm64, Go 1.26.3. No production GC/scavenging policy changed.

## Throughput

Five 250 ms rounds alternate before/after/pinned native,1024-byte reads: [raw](cold-prefix-throughput.jsonl). Severe less/nvim scheduling outliers remain; they cannot support a clean timing gate. Arrays are sample-order MiB/s; changes use medians:

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

Five alternating 1 s pairs with GOMAXPROCS=1 and no third engine supplemented those cases: [raw](cold-prefix-targeted.jsonl). Wall/CPU medians differed by less than 2.2%:

| Stream | Before, five samples | After, five samples | Wall median change | CPU median change |
|---|---|---|---:|---:|
| less-recording | 282.5, 273.8, 269.6, 271.9, 247.4 | 276.7, 270.6, 271.0, 274.9, 248.8 | -0.3% | -0.5% |
| rec-nvim-recording | 668.5, 651.1, 634.5, 644.8, 657.1 | 651.3, 653.9, 630.0, 636.3, 642.4 | -1.3% | -1.3% |
| rec-man-recording | 346.1, 336.7, 336.6, 326.1, 336.1 | 336.9, 330.8, 325.9, 329.5, 323.5 | -2.1% | -2.1% |

## Idle memory

Five fresh processes per workload,1,024 live terminals each. RSS arrays and heap medians are B/terminal: [raw](cold-prefix-memory.jsonl). Existing GC procedure is identical before/after. `-erase` appends shell EL; prompts are asserted and history must stay empty.

| Workload | Before RSS, five samples | After RSS, five samples | Before / after heap median |
|---|---|---|---:|
| prompt-compact | 2544, 2400, 2640, 2496, 2528 | 1008, 1056, 1408, 1216, 976 | 1443 / 1443 |
| colored-prompt-compact | 2544, 2384, 2624, 2736, 2496 | 1600, 1648, 1808, 1536, 1824 | 1448 / 1443 |
| prompt-erase-compact | 2496, 2368, 2624, 2560, 2464 | 1216, 960, 1152, 1120, 1184 | 1438 / 1443 |
| colored-prompt-erase-compact | 2624, 2448, 2544, 2544, 2528 | 1632, 1568, 2048, 1520, 1424 | 1443 / 1443 |
| prompt-erase | 2800, 2368, 2544, 2320, 2336 | 1056, 1136, 1296, 1088, 1072 | 2515 / 1443 |
| colored-prompt-erase | 2480, 2256, 2288, 2272, 2208 | 1328, 1440, 1632, 1328, 1280 | 2510 / 1523 |

The generic copy helper/prefix wrapper/blank operation remained inlineable (cost 34/75/76); cold capacity policy was outlined. ALT redraw kept 41 allocations/about 83 KB per repeated rec-man recording. Already-dirty erasures avoid clean-row scans, while damage remains independent of stored zero tails.

Full tests/vet passed;20 s prefix fuzz completed 100,556 executions. Existing cases cover Unicode, byte growth, insert, colored scrolling, EL, half-wide styles, links/underline/protection, damage and native byte traces. Warm forty-row logs/TUI remain zero-allocation; codecs/fixed layouts were unchanged.

Reproduce via the [driver](../../README.md): `COMPARE_DURATION=250ms bash oracle/ghostty/ab.sh BEFORE AFTER OUTPUT`; memory `-workload prompt-erase-compact -n 1024`; controlled throughput `GOMAXPROCS=1 COMPARE_ROUNDS=1 COMPARE_ENGINE=phosphor COMPARE_STREAM=less-recording COMPARE_CHUNK=1024 AFTER -mode throughput -duration 1s`. Set PHOSPHOR_DIR, alternate order and repeat five times.

Binary SHA256: before `de3b37c8eead44983536fc4e04b0d8a09d39073304f69ffcd0bfe2436af6798f`; after `9683e9b4ee8259b7ebe3551bc3b34af41fab8f4bf85c6d84f20c9c8af8bb7b35`.
