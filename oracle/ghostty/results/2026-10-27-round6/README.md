# Phosphor v0.4.0: terminal state and idle memory

This round closes measured headless text-model gaps against libghostty-vt
and reduces memory for idle shell sessions. It does not establish universal
terminal compatibility or a win on every throughput stream.

Performance and prompt-memory source: `7a2e8f2f76e4b85a342ee87241e9738c7a39fad3`.
Final correctness source includes the cold formatter repair in
`2c459f56413b5497b1391cd1898508f2d1295994`; it unifies empty-row output
across allocated and implicit storage without changing the input hot path.
Baseline: released v0.3.0, `3a732c18eee88e55977b233eb25b905633a669f7`.
Native source: `46e9dc9a927b2e7e7cba79e6902c176127dbdf2c`, Zig 0.17.0,
ReleaseFast, build-info version 0.1.0-dev. macOS arm64, Apple M5 Pro,
Go 1.26.3; Linux checks use Go 1.26.10 on arm64 in Docker.
Production remains pure Go; native code is used only by the comparison tool.

## Compatibility

The original 72-case corpus now passes 174/174 direct and 174/174 Phosphor
continuation actions, versus 150/174 in v0.3.0. The original historical
cluster-capacity case also matches now. `final-original-corpus.jsonl`
retains the entire output, including native export failures and its old scopes.

The expanded corpus has 208 named cases and 773 parity actions. Whole,
one-byte and seven-byte writes have identical results. Final macOS outputs
are byte-identical across both sides of the performance-only fix, and the
Linux whole-write output is byte-identical to macOS:

| Comparison | Pass / actions |
|---|---:|
| Untouched engines, direct input |772/773 |
| Phosphor checkpoint/repaint/Compact continuation |771/773 |
| Native checkpoint/repaint continuation |666/773 |
| Combined, requiring all comparisons |665/773 |

`final-trace-{0,1,7}.jsonl` retains every raw difference. The command exits 1
because the policy difference and two known ANSI-export variants remain; running it successfully is not a
passing result. Product identity and initial host colors keep their separate
scopes and raw replies. The remaining parity differences are:

- DEC special-graphics underscore: Phosphor retains xterm's blank mapping;
  the pinned native engine keeps an underscore. This deliberate policy
  difference remains in the direct denominator.
- Forced 1047 exit after height or width growth: old and new inactive ALT
  regions have different backgrounds. An ANSI replay that preserves the independent mode
  bits must erase ALT with one background when returning to MAIN. Exact
  snapshots preserve the original cells and their future use.

See the [coverage contract](../../coverage.md) for observations and further
portable ANSI limits, including REP and some outside-origin/narrowed-cluster
states. An exact binary checkpoint has a stronger contract than ANSI output.

The changes include SGR 58/59 underline colors; primary+64 suffix codepoints
with overflow width effects; independent mouse modes and encoding 1016;
British charset/single shift; gated DECCOLM; smooth scrolling; Kitty OSC 21
and named X11 colors; exact empty/literal-space history; native tab and resize
policies; OSC 133 row/cursor/redraw state; two per-screen DECSC slots shared
with 1049; live hyperlink lifetime; independent 47/1047/1049 bits and unsaved
mode-restore defaults. Cold MAIN prefixes avoid full-row prompt allocations.

The translated Ghostty suite passes 524 cases, has 0 known failures and keeps
25 exclusions. Seven former prompt exclusions now assert text, cursor,
history and public row metadata; `translated-prompts.jsonl` also checks those
same inputs through the native ABI. Other suites retain unchanged references
and explicit policy conflicts, rather than rewriting expectations to agree.

## Throughput

Median MiB/s, five alternating rounds, 500 ms per sample, 120x40, 1 KiB writes.
Both Go versions use the same driver selectors and data. Raw wall and process
CPU samples are in `throughput.jsonl`; no outliers are removed.

| Stream | v0.3.0 | v0.4.0 candidate | Ghostty |
|---|---:|---:|---:|
| claude-synthetic | 441.8 | 427.4 | 467.7 |
| logs | 521.8 | 537.9 | 884.7 |
| progress | 1286.4 | 1237.2 | 1098.6 |
| vim-recording | 385.4 | 362.9 | 317.3 |
| less-recording | 260.0 | 308.9 | 131.7 |
| rec-nvim-recording | 587.8 | 707.5 | 601.0 |
| rec-nano-recording | 499.2 | 508.9 | 563.5 |
| rec-man-recording | 409.7 | 519.9 | 407.8 |

Phosphor leads native on progress, vim, less, nvim and man in these samples;
native leads logs, the synthetic TUI and nano. Compared with v0.3.0, gains
are mixed: several recordings improve, while TUI/progress/vim medians are
about 3–6% lower. Host scheduling and clock variation remain visible; CPU
and wall deltas are reported separately in the raw data. These short samples
do not prove small regressions or universal speedups.

An intermediate build lost bulk zeroing after semantic erasure moved through
fillCells. Matched profiles identified that cost. Restoring clear for default
backgrounds and skipping needless SGR0 pool work recovered about 8% CPU on
Claude-shaped output and 4% on progress versus that intermediate build.
[Performance followup](performance-followup.md) retains all probes, rejected
row-access work, profile evidence and scatter. The final matrix above uses
the accepted fix. `cursor-throughput.jsonl` and `cursor-controlled.jsonl`
retain the earlier 8d1511 measurements, not the release candidate.

## Idle memory

Fresh processes, 1024 live terminals, three samples per engine/workload;
medians are bytes per terminal. `Compact()` and native FULL compression are
both applied. `memory.jsonl` retains all samples. Go heap covers Go objects;
native memory is measured through process RSS, not its tiny wrapper heap.

| Workload | v0.3 heap | Candidate heap | v0.3 RSS delta | Candidate RSS delta | Native RSS delta |
|---|---:|---:|---:|---:|---:|
| prompt-erase-compact | 2,521 | 1,449 | 2,432 | 1,264 | 31,280 |
| colored-prompt-erase-compact | 2,515 | 1,443 | 2,576 | 1,760 | 47,840 |
| cleared-compact | 2,537 | 1,460 | 5,808 | 5,152 | 64,208 |
| rich-cleared-compact | 2,546 | 1,472 | 6,688 | 5,072 | 80,720 |

Prompt heap is about 42% lower. Prompt RSS also falls; clearing large old
screens can retain allocator pages, so its RSS savings are smaller than its
heap savings. A Terminal remains 560 B and a cell 16 B on 64-bit platforms.
No production GC/scavenging policy changed. The comparison driver warms
shared initialization and uses the same GC procedure for every engine.

Correct 1049 exit retains inactive editor contents. The separately pinned
[completed-editor measurements](closed-memory.md) retain 120 fresh-process
samples at 256 and 1024 terminals. After Compact, Phosphor keeps 22–47 KB heap
and 34–57 KB RSS per terminal, versus about 82 KB native RSS after FULL.
Those are larger states than fresh prompts. No new cold-screen codec or
unimplemented compression saving is claimed.

## Persistence and validation

Snapshot format 2 preserves the new state and still decodes actual released
v0.3.0 format 1 fixtures. The fixtures are not regenerated by the new writer.
Ptyd tests cross old/new checkpoints, tail replay, resize and future output;
its outer checkpoint header remains 2. New inner snapshots require the new
library; the ptyd README states the corresponding minimum reader version.

Go 1.24/current-Go tests, race/vet, examples, oracle tests, native ABI tests
and bounded fuzz runs cover the change. New fuzz targets compare prefix
storage with dense production storage and validate private history markers.
Independent reviews found and fixed wrapped-row repair; final cursor and
snapshot reviews reported no findings. Release fuzzing also found a test
that compared allocated versus implicit blank ALT storage. It now compares
physical cells and explicitly checks active-screen and mode bits. The
known forced 1047 rectangle-growth loss has a precise, tested ANSI expectation:
only zero ALT padding acquires the erase background. All other repaint
observations still compare, and exact Snapshot plus future output must
preserve the original cells. Raw native failures remain visible. Near-miss
cases, including width-only 80/132 destinations, keep strict repaint checks. Compact fuzzing also found storage-dependent dummy
glyph selection in one-column repaint; both old exports had equal tested
futures. The formatter now emits identical bytes before/after Compact,
with the original byte-equality assertion retained.

Native tests include 216 bounded
screen-mode sequences and future writes, not only the restored picture.

Linux daemon crash tests require a container init process to reap orphaned
children. The first container used sleep as PID 1 and left killed children as
zombies; those tests passed with --init. This was a test-environment failure,
not a daemon patch. No installed binary or running user daemon was changed.

## Reproduce

Build the pinned native library and driver as in the [oracle README](../../README.md).
Run `ab.sh BASELINE CANDIDATE OUTPUT`, setting PHOSPHOR_DIR and
COMPARE_DURATION=500ms. Run memory mode with workload names from the table,
engine phosphor/ghostty and n=1024. Repeat in fresh processes, alternate order
and retain every sample. Baseline driver overlays only the identical
benchmark selectors; its production library remains released v0.3.0.

Measured binary SHA256:

- final-baseline-driver: `321741b81907d024e01cacb10afc462b48de99370db84215c4f0b7d8d686cbdd`.
- release-candidate: `540bfd237b83da011506818d8ac747bb1bb94d13f730e7b42cbb017715e5f5f4`.
