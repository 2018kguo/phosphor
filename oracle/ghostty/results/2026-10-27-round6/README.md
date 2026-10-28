# Idle-state comparison: cheaper prompts, fuller terminal state

Prompt heap fell about 42%; several editor/pager streams became faster. Native still led logs, synthetic TUI and nano. Compatibility expanded, with explicit policy/ANSI failures retained. These measurements do not establish a win on every workload or universal compatibility.

Measured performance/prompt memory: `7a2e8f2f76e4b85a342ee87241e9738c7a39fad3`. Final correctness includes `2c459f56413b5497b1391cd1898508f2d1295994`, which canonicalizes empty-row repaint without changing Write. Baseline: `3a732c18eee88e55977b233eb25b905633a669f7`. Native: `46e9dc9a927b2e7e7cba79e6902c176127dbdf2c` ReleaseFast; build-info omits the commit. Apple M5 Pro, macOS/arm64; Linux checks: Docker/arm64. Production stays pure Go.

Build toolchains are recorded in [environment metadata](../environments.json).

## Compatibility

In the original 72-case corpus, direct input and Phosphor continuation each passed 174/174 actions; the baseline passed 150/174. The additional historical cluster-capacity case also matches: [raw](final-original-corpus.jsonl). Expanded 208-case/773-action results:

| Comparison | Pass / actions |
|---|---:|
| Untouched engines, direct input |772/773 |
| Phosphor checkpoint/repaint/Compact continuation |771/773 |
| Native checkpoint/repaint continuation |666/773 |
| Combined, requiring all comparisons |665/773 |

Whole/[one-byte](final-trace-1.jsonl)/[seven-byte](final-trace-7.jsonl) results agree; [Linux](final-trace-linux.jsonl) is byte-identical to [macOS whole writes](final-trace-0.jsonl), also unchanged across the performance fix. Trace exits 1. Identity/initial host-color scopes retain raw replies separately. Remaining parity failures:

- DEC-graphics underscore follows xterm's blank in Phosphor versus native underscore; it remains in the direct denominator.
- Forced 1047 exit after height/width growth erases mixed-background inactive ALT with one background when preserving independent mode bits. Exact Snapshot preserves the cells and future behavior.

The [contract](../../coverage.md) explains these observations and portable ANSI limits (REP, some outside-origin/narrowed-cluster states). New coverage spans SGR 58/59, primary+64 suffixes/overflow width effects, independent mouse/1016, British charset/single shift, DECCOLM/smooth scroll, Kitty OSC 21/X11 colors, empty-versus-space history, tabs/resize, OSC 133, per-screen DECSC shared with 1049, link lifetime and independent 47/1047/1049/saved-mode defaults. MAIN prefixes avoid full-row prompt allocation.

Translated Ghostty:524 passing/0 known failures/25 exclusions; seven former prompt cases now check text/cursor/history/public row marks. [Native translation inputs](translated-prompts.jsonl) check those same sequences. Other suites retain conflicting policies unchanged.

## Parsing

Five alternating 500 ms rounds,120×40,1 KiB writes; median MiB/s. Identical selectors/data, no discarded outliers; wall/process CPU: [raw](throughput.jsonl).

| Stream | Baseline | Candidate | Ghostty |
|---|---:|---:|---:|
| claude-synthetic | 441.8 | 427.4 | 467.7 |
| logs | 521.8 | 537.9 | 884.7 |
| progress | 1286.4 | 1237.2 | 1098.6 |
| vim-recording | 385.4 | 362.9 | 317.3 |
| less-recording | 260.0 | 308.9 | 131.7 |
| rec-nvim-recording | 587.8 | 707.5 | 601.0 |
| rec-nano-recording | 499.2 | 508.9 | 563.5 |
| rec-man-recording | 409.7 | 519.9 | 407.8 |

Versus the baseline, recordings improve unevenly while TUI/progress/vim medians are about 3–6% lower. Clock/scheduling scatter prevents small universal claims. [Performance followup](performance-followup.md) explains the accepted bulk-clear/SGR 0 repair (about 8% Claude CPU/4% progress recovery versus an intermediate build), rejected row probe and all raw samples. `cursor-throughput.jsonl` /`cursor-controlled.jsonl` measure earlier 8d1511, not this candidate.

## Idle memory

Three fresh processes per engine/workload,1,024 live terminals, medians in B/terminal. Compact/native FULL both applied; shared initialization and GC procedure identical. Use RSS for native memory, not wrapper heap: [raw](memory.jsonl).

| Workload | Baseline heap | Candidate heap | Baseline RSS delta | Candidate RSS delta | Native RSS delta |
|---|---:|---:|---:|---:|---:|
| prompt-erase-compact | 2,521 | 1,449 | 2,432 | 1,264 | 31,280 |
| colored-prompt-erase-compact | 2,515 | 1,443 | 2,576 | 1,760 | 47,840 |
| cleared-compact | 2,537 | 1,460 | 5,808 | 5,152 | 64,208 |
| rich-cleared-compact | 2,546 | 1,472 | 6,688 | 5,072 | 80,720 |

Terminal 560 B/cell 16 B; no production GC/scavenging change. Heap can fall while allocator pages remain resident, especially after clearing large screens. Correct 1049 exit also keeps old ALT contents: separately pinned [completed-editor results](closed-memory.md) retain 120 samples at 256/1024 terminals, with 22–47 KB heap/34–57 KB RSS after Compact versus about 82 KB native RSS. Fresh prompts are a smaller state; no new cold-screen codec saving is claimed.

## Persistence and validation

Inner Snapshot 2 preserves new state and reads the archived earlier-format fixtures; they were not regenerated. Ptyd's outer header stays 2; tests cross old/new checkpoints, tail replay, resize and future output. New inner snapshots require the documented newer reader.

Minimum-supported/current-Go tests, race/vet, examples, oracle/native ABI and bounded fuzz passed. Reviews fixed wrapped-row repair and reported no final cursor/snapshot findings. Native tests cover 216 screen-mode sequences plus futures. Release fuzz separately fixed a test's storage-dependent ALT comparison, named the forced 1047 background expectation (only zero padding gains erase BG; all other observations/snapshot/futures stay strict), and canonicalized one-column dummy selection without weakening Compact byte equality. Width-only 80/132 near misses retain strict ANSI checks. Linux daemon-crash tests needed Docker `--init` to reap children; sleep-as-PID 1 caused the first environment failure. No installed daemon changed.

Reproduce with the [driver](../../README.md): `COMPARE_DURATION=500ms bash oracle/ghostty/ab.sh BASELINE CANDIDATE OUTPUT`; set PHOSPHOR_DIR. Run table workload names with `-mode memory -engine phosphor` /`ghostty -n 1024`, repeating fresh processes in alternating order. Baseline overlay changes only benchmark selectors, not released production code.

Measured binary SHA256: baseline `321741b81907d024e01cacb10afc462b48de99370db84215c4f0b7d8d686cbdd`; candidate `540bfd237b83da011506818d8ac747bb1bb94d13f730e7b42cbb017715e5f5f4`.
