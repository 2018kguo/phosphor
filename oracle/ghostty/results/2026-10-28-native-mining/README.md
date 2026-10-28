# What source mining changed

Reading libghostty-vt helped identify repeated work in Phosphor's Unicode and
printing paths. The combined result is about **6.3× faster on mixed graphemes**,
**41% faster on progress output** and **13% faster on budgeted logs** than the
Phosphor baseline in the primary CPU comparison.

Native's mixed-grapheme lead falls from roughly 7.3× in the prior audit to
about **7% with history** and **4% without history** here. Native still leads
budgeted logs by about **21%**. Phosphor leads progress, editor/pager and link
fixtures, while agent-TUI throughput is roughly level. Small differences need
longer samples; these results do not establish an overall fastest terminal.

All [600 timing samples](throughput-summary.json), [126 memory samples](memory-summary.json)
and source manifests for the [baseline](baseline-source.json), [candidate](candidate-source.json)
and [common driver](driver-source.json) are retained. [Metadata](metadata.json)
records toolchains, binaries, the native library and policies. The host is
macOS arm64 on an Apple M5 Pro, with native built in ReleaseFast.

## Ideas that transferred

Ghostty's [property lookup](https://github.com/ghostty-org/ghostty/blob/6ba144158577d033ccb6ed4b574e4013957c9161/src/unicode/props.zig)
keeps width and grapheme information together. Its
[boundary kernel](https://github.com/ghostty-org/ghostty/blob/6ba144158577d033ccb6ed4b574e4013957c9161/src/unicode/grapheme.zig)
uses precomputed transitions, and its
[printing path](https://github.com/ghostty-org/ghostty/blob/6ba144158577d033ccb6ed4b574e4013957c9161/src/terminal/Terminal.zig#L679)
batches stores into cells. Phosphor now uses:

- One generated lookup for width, boundary class and emoji eligibility,
  replacing separate tables and repeated range searches.
- A transition table and cached boundary context in three bytes of existing
  cluster padding. Entries remain 28 bytes on 32-bit targets and 40 bytes on 64-bit targets.
- Bulk encoding for fully joined history clusters whose styles stay the same.
- ARM64 vector stores for longer ASCII runs, with the original scalar path on other architectures.

The core adds no per-terminal fields or checkpoint fields. Unicode lookup data
totals 52,512 shared bytes, and the production module has zero external Go
dependencies. The adapted boundary rules retain their [MIT notice](../../../../licenses/uax29.txt).
The [exploratory records](screening/method.json) include rejected row batching,
wide ASCII scanning and history variants; their losses were not hidden.

## CPU throughput

Median MiB per process CPU second, including GC and native work. Five alternating
samples per build/case, one Go execution thread, 120×40 geometry and 1 KiB writes.
Budgeted samples last 750 ms; no-history samples last 500 ms. Inputs are created
before timing; replies are formatted into discard sinks. Native continuation
retention is disabled outside checkpoint checks.

| Stream | Baseline, budgeted | Phosphor, budgeted | Native, budgeted | Phosphor, no history | Native, no history |
|---|---:|---:|---:|---:|---:|
| Agent TUI | 465.4 | 511.3 | 504.3 | 515.9 | 500.4 |
| Logs | 681.4 | 769.5 | 932.9 | 1,315.1 | 594.1 |
| Progress | 1,449.0 | 2,036.1 | 1,144.9 | 2,101.6 | 1,139.3 |
| vim | 422.8 | 440.1 | 346.5 | 433.3 | 346.3 |
| less | 355.2 | 377.0 | 139.0 | 380.5 | 137.3 |
| nvim | 777.2 | 834.4 | 630.6 | 835.0 | 634.7 |
| nano | 616.7 | 705.7 | 578.6 | 700.1 | 581.7 |
| man | 625.4 | 760.4 | 434.4 | 751.9 | 440.9 |
| Grapheme-heavy Unicode | 23.1 | 146.1 | 156.4 | 180.1 | 187.5 |
| OSC 8 links | 266.4 | 275.0 | 24.3 | 359.1 | 22.4 |

Nominal budgets are 2,000 lines /256 KiB on both engines. Phosphor retained
2,000 logical log lines; native retained 474–699 physical rows in these primary
samples. For graphemes, Phosphor retained 1,691 logical lines and native retained
374–570 physical rows. These are different amounts of work. With history disabled
and zero rows verified, Phosphor is about **2.2× faster on logs**.

[1 KiB](throughput-1024.jsonl), [8 KiB](throughput-8192.jsonl),
[64 KiB](throughput-65536.jsonl) and [no-history](throughput-none-1024.jsonl)
records preserve every sample, input hash, byte total and retention count.
The history-enabled grapheme lead is about 14% at 8 KiB and 20% at 64 KiB.
Chunk size changes some ratios. PTY I/O, HTTP, rendering and client delivery
are outside these emulator CPU measurements.

## Idle storage and allocation

A compacted prompt still retains about **1,512 bytes of Go heap**. Every tested
retained heap is unchanged within measurement noise; the extra context occupies
existing padding. Prompt RSS is roughly 10–26× lower than native across 128- and
1,024-emulator processes. Completed editors/pagers retain inactive buffers and
use about 19–51% less RSS; exactly 100 styled history rows use about 3.1× less.

Both engines compact. Samples use fresh processes, the same collection policy
and no forced scavenging. Native wrapper heap excludes native allocations, so
cross-engine comparisons use RSS. Allocator pages affect those ratios. Child
programs, PTYs, output rings and shared-library startup are excluded.
[Memory medians](memory-summary.json) link the [128-terminal](memory-128.jsonl)
and [1,024-terminal](memory-1024.jsonl) records.

Separate Go benchmarks use wall time and decimal MB/s. The repeated mixed
fixture allocates about **16 KB per MiB operation**, down from 153 MB. A
1,024-distinct-cluster fixture improves about **2.4×** and still allocates about
45 MB per operation, versus 240 MB before. Caching does not make every stream
allocation-free. [Unicode pairs](unicode-paired.txt), [medians](unicode-summary.json),
[variety pairs](variety-paired.txt) and [medians](variety-summary.json) retain all samples.

## Fidelity and provenance

Source inspection found that earlier native Unicode labels were assumptions:
the pinned dependency already used newer data. The width vector was correct;
all 1,114,112 codepoint values are identical before and after the provenance
correction. The generator now verifies the pinned native source and audited data.

Phosphor also adopts the newer boundary rules and data. The official corpus
checks **853 cases and 1,355 boundaries**: 1,329 exact official matches and 26
explicit Control-to-Other tailoring differences. No cases are omitted.

Ten added native traces close **48 failing direct actions**. Across the full
218-case corpus, direct parity improves from **804/853 to 852/853 actions**.
Phosphor continuation remains 851/853 and native continuation 746/853; those
measure each engine against its own untouched control. Whole, one-byte and
seven-byte writes agree. The old DEC graphics mapping difference and two
Phosphor ANSI background variants remain visible; exact snapshots preserve
those cells. These are bounded observations, not a universal compliance score.

[Baseline traces](trace-baseline-0.jsonl), [final traces](trace-final-0.jsonl) and
chunked records retain failures. Burst tests verify all input, replies, history
and future state; the discarded-prefix negative control fails on both engines.
Full race, vet, fuzz, source-generation, scalar enumeration and focused cache/
serializer checks pass. Independent reviews found no blockers.

## Reproduce

Follow the [native build instructions](../../README.md#build). Build both source
revisions with the same driver and input selectors from the metadata. From `oracle/`:

```sh
GOMAXPROCS=1 PHOSPHOR_DIR="$(cd .. && pwd)" \
  COMPARE_CHUNK=1024 COMPARE_DURATION=750ms COMPARE_HISTORY=budgeted \
  bash ghostty/ab.sh /tmp/baseline /tmp/candidate /tmp/throughput.jsonl
```

Repeat the other chunk sizes and `none` policy. Run each memory sample in a
fresh process at both terminal counts. Build the same current benchmark files
against both Go sources for the auxiliary Unicode/variety controls. Compare
actual retained contents alongside throughput.
