# Terminal work, retention and idle storage

Phosphor improves the eight original streams by 4.2–21.1% over its baseline in
this run. Its storage remains small. A broader workload set also shows a large
native lead on grapheme-heavy Unicode. Neither engine wins every case.

[Metadata](metadata.json) records the exact source, native library and binary
hashes, toolchains, policies and counts. Production source manifests cover the
[candidate](candidate-source.json) and [baseline](baseline-source.json); separate
[budget](driver-budget-source.json) and [control](driver-control-source.json)
manifests identify the common driver used on both sides of each phase.
The native checkout matched upstream HEAD when checked. The host was macOS
arm64 on an Apple M5 Pro; native used ReleaseFast.

## What the audit changed

[The input-drop warning](https://x.com/mitchellh/status/2115292243443519571)
asks whether throughput counts bytes actually emulated. Both drivers call their
parsers synchronously for every chunk. There is no frame queue overwriting
unparsed input. The [burst witness](../../measurement_test.go) checks 86,420
bytes, all 128 replies, ordered history, every styled/linked cell, graphemes,
modes, title and future output across four chunk sizes. Keeping only the last
64 KiB fails on both engines. [Test output](burst-witness.txt) preserves that
negative control.

The audit did find setup issues. Native continuation retention was enabled
outside checkpoint tests; its tracker reserves storage even at a prompt.
Ordinary throughput and memory now use native's disabled tracking policy.
Checkpoint tests opt in explicitly. Both timed engines enable reply formatting
with discard sinks. The old larger prompt RSS ratios should not be carried
forward to this setup.

Identical byte budgets do not imply identical retained work. Phosphor counts
serialized logical lines; native counts physical rows and prunes pages.
Every throughput sample now records actual retention and input hashes. The
history-disabled control requires zero retained rows before emitting a result.
Graphics, GUI rendering and oversized ignored control payloads are outside
these text-model workloads.

## CPU throughput

Median MiB per process CPU second, including GC/native work. Inputs are generated
before timing. Geometry is 120×40, one Go execution thread, five alternating
rounds and 1 KiB writes. Budgeted samples last 750 ms; history-disabled controls
last 500 ms. Both engines receive identical bytes and discard formatted replies.

Budgeted means the same nominal 2,000-line /256 KiB settings. For logs, Phosphor
retains 2,000 logical lines while native retains 357–678 physical rows in these
samples. The table measures that policy tradeoff, not equal retained history.
The last two columns isolate the zero-history configuration.

| Stream | Baseline, budgeted | Phosphor, budgeted | Native, budgeted | Phosphor, no history | Native, no history |
|---|---:|---:|---:|---:|---:|
| Agent TUI | 443.9 | 462.5 | 479.3 | 474.9 | 474.7 |
| Logs | 556.2 | 673.8 | 880.2 | 1,058.4 | 564.7 |
| Progress | 1,241.7 | 1,441.8 | 1,072.6 | 1,469.9 | 1,034.4 |
| vim | 359.0 | 418.9 | 340.0 | 417.7 | 331.0 |
| less | 316.7 | 354.5 | 134.8 | 351.9 | 137.4 |
| nvim | 693.2 | 774.8 | 627.9 | 773.9 | 629.5 |
| nano | 512.7 | 616.4 | 557.0 | 625.5 | 548.5 |
| man | 547.5 | 617.8 | 427.2 | 618.0 | 413.8 |
| Grapheme-heavy Unicode | 22.9 | 22.7 | 165.4 | 25.7 | 187.0 |
| OSC 8 links | 244.9 | 264.9 | 24.5 | 355.1 | 21.9 |

Logs improve 21.1% over the Phosphor baseline with history. Native remains about
31% faster in that budgeted case. With history disabled, Phosphor processes
about 1.87× as many bytes per CPU second. The difference matters when choosing
an embedding policy; disabling history also removes its contents.

The Unicode fixture explicitly enables grapheme mode on both engines. Native
is about 7.3× faster there; the Phosphor change is within 1% of its baseline.
OSC 8 links favor Phosphor in this fixture. These added cases prevent the
ASCII-heavy recordings from standing in for every terminal workload.

All [150 primary samples](throughput-1024.jsonl), [150 zero-history samples](throughput-none-1024.jsonl)
and [medians](throughput-summary.json) remain available. Separate
[8 KiB](throughput-8192.jsonl) and [64 KiB](throughput-65536.jsonl) phases use
five 400 ms samples per build/case. They retain all 300 additional samples;
chunk size changes some ratios. Short shared-host samples do not establish
small differences reliably. These are emulator CPU costs, not PTY, network,
input-to-display latency or frame rate.

## Idle memory

Median bytes per terminal, three fresh processes per build/workload, 1,024 live
emulators. Phosphor calls Compact; native calls FULL compression. Both have
reply sinks and native continuation retention is disabled. Collection is the
same on both sides, without forced page scavenging. Native wrapper Go heap
excludes native allocations, so cross-engine comparisons use incremental RSS.

| Idle state | Phosphor Go heap | Phosphor RSS growth | Native RSS growth |
|---|---:|---:|---:|
| prompt | 1,512 | 1,184 | 23,968 |
| colored prompt | 1,512 | 1,680 | 40,400 |
| cleared | 1,512 | 3,904 | 56,784 |
| rich cleared | 1,512 | 3,968 | 73,296 |
| 100 styled history rows | 36,952 | 39,648 | 122,288 |
| closed vim | 47,000 | 60,384 | 75,040 |
| closed nvim | 47,352 | 62,528 | 75,216 |
| closed nano | 37,848 | 58,880 | 75,008 |
| closed less | 30,184 | 47,408 | 75,104 |
| closed man | 22,472 | 38,528 | 74,976 |

The prompt retains 1,512 bytes of Go heap. At 1,024 terminals, plain/colored
prompt RSS is about 20–24× lower. The [128-terminal samples](memory-128.jsonl)
show about 9–16× lower RSS, exposing allocator/page amortization. Use the range,
not a universal per-terminal ratio. Completed editor/pager RSS is 17–49% lower
at 1,024 terminals; inactive alternate buffers remain available for re-entry.

The 100-row fixture has exactly equal retained contents before and after both
compactions, checked by TestEqualHistoryMemoryFixture. Its RSS is about 3.1×
lower here. Baseline and candidate retained Go heaps are unchanged. These
measurements exclude the child process, PTY, output rings and shared-library
startup cost. Freed Go objects need not immediately leave RSS.
[All 126 samples](memory-summary.json) link to the
[1,024-terminal raw records](memory-1024.jsonl) and the 128-terminal records.

## Other Go cores

The separate [Go samples](go-throughput.txt) use library defaults, 8 KiB writes
and three 300 ms samples. They are not an equal-feature comparison: reply,
history and parser policies differ. [Measured throughput](go-throughput-summary.json)
is reported in decimal MB/s, as Go's benchmark tool emits it.

[Idle Go samples](go-idle.txt) use 200 terminals per process, five repeats,
defaults and no Compact. They measure HeapInuse, not RSS. Native allocations
are not involved in that separate harness. [Pins and policies](metadata.json)
record the tested libraries; defaults allocating large parser/history buffers
must not be presented as equivalent state.

## Compatibility and reproduction

All baseline/candidate trace records match across whole, one-byte and seven-byte
writes. The 208-case corpus retains 772/773 direct parity actions, 771/773
Phosphor continuations and 666/773 native continuations. The policy and ANSI
failures remain in the [raw summaries](trace-summaries.json). Native-linked
checks and all 1,112,064 scalar widths pass. Finite corpus results are not a
percentage of universal terminal compatibility.

Build both revisions with the same driver, following the
[native build instructions](../../README.md). Run the comparison from Phosphor's
root:

```sh
COMPARE_DURATION=750ms COMPARE_CHUNK=1024 PHOSPHOR_DIR=/path/to/phosphor bash oracle/ghostty/ab.sh BASELINE CANDIDATE budgeted.jsonl
COMPARE_DURATION=500ms COMPARE_CHUNK=1024 COMPARE_HISTORY=none PHOSPHOR_DIR=/path/to/phosphor bash oracle/ghostty/ab.sh BASELINE CANDIDATE no-history.jsonl
```

Memory uses a fresh process for every sample: select `-mode memory -n 1024`
and a named workload, repeating at least three times and multiple counts.
Run native-linked tests, the rich trace corpus and the burst negative control
before interpreting throughput. Preserve failures and every raw sample.
