# Compare with libghostty-vt

This optional tool compares headless terminal state and emulation cost through
the public native C API. Production Phosphor needs neither cgo nor Ghostty.
Read the [measured results](../../docs/benchmarks.md),
[evidence index](../../docs/benchmarks.json) and [comparison contract](coverage.md)
before interpreting a speed or compatibility result.

## Build the native library

Use the Zig toolchain required by the pinned native
[build manifest](https://github.com/ghostty-org/ghostty/blob/6ba144158577d033ccb6ed4b574e4013957c9161/build.zig.zon).
From a clean native checkout:

```sh
git clone https://github.com/ghostty-org/ghostty.git /tmp/ghostty-comparison
git -C /tmp/ghostty-comparison checkout 6ba144158577d033ccb6ed4b574e4013957c9161
cd /tmp/ghostty-comparison
zig version
zig build -Demit-lib-vt -Doptimize=ReleaseFast -j4
```

Record the toolchain and library checksum with new results. The native runtime
build-info API does not expose a trustworthy source revision; the driver's
expected source pin is a declaration, not observed proof of the linked library.
Verify the checkout and build yourself. The historical timing metadata includes
a library hash and optimization but did not preserve the Zig executable checksum.

From the Phosphor repository root on macOS:

```sh
export PHOSPHOR_DIR="$PWD"
export GHOSTTY_SOURCE=/tmp/ghostty-comparison
export CGO_CFLAGS="-I$GHOSTTY_SOURCE/include"
export CGO_LDFLAGS="-L$GHOSTTY_SOURCE/zig-out/lib -lghostty-vt -Wl,-rpath,$GHOSTTY_SOURCE/zig-out/lib"
cd oracle
go test -tags ghostty ./ghostty
```

Keep native headers and library from the same checkout. On Linux configure the
runtime library path for the same library. Keep the native optimization,
continuation policy and reply handling identical across the compared builds.
The burst checks include deliberate input loss; checkpoint/reply witnesses and
timed-constructor witnesses serve different configurations.

## Build matched drivers

Use the same driver against two explicit core revisions. For the recorded
baseline and candidate:

```sh
ghostty/build.sh ecfe310cd6da699df77796d35a091423075b30b3 \
  e706934f1da337140282911de72fd725262f42e3 "$PHOSPHOR_DIR/.bench/native"
```

The builder uses disposable core checkouts and alternate module files, leaving
the checkout and public module unchanged. It rejects unequal benchmark fixture
source. Both binaries use this driver's code and the recordings selected through
`PHOSPHOR_DIR`. Keep the generated build/source records with the samples and
inspect them for uncommitted measurement changes.

## Measure

```sh
GOMAXPROCS=1 COMPARE_CHUNK=1024 COMPARE_DURATION=1s COMPARE_HISTORY=budgeted \
  ghostty/ab.sh "$PHOSPHOR_DIR/.bench/native/baseline" \
  "$PHOSPHOR_DIR/.bench/native/candidate" "$PHOSPHOR_DIR/.bench/throughput.jsonl"
```

The maintained runner balances all three sample positions. Its default six
outer repeats differ from the five-repeat historical experiment. Each record
retains the outer repeat and position, input hash, actual bytes, history policy
and retention. Repeat at 8 KiB and 64 KiB writes; run `COMPARE_HISTORY=none` as a
separate zero-retention control. Keep every sample and failure.

| Mode | Observation |
|---|---|
| `trace -traces ghostty/traces.json` | Cells, replies and future behavior against untouched controls. Nonzero exit on parity failure. Use `-trace-chunk 1` and `7` for byte boundaries. |
| `throughput -duration 750ms` | Process CPU and wall throughput, including GC and the C boundary. Select input, chunk and policy with `COMPARE_*`. |
| `memory -n 1024` | Incremental retained heap/RSS for each state; run every sample in a fresh process. |
| `state` | Snapshot, repaint and restore costs, with checkpoint bookkeeping enabled where required. |
| `cursor-replies` | Raw cursor query answers. |
| `check -probes ghostty/probes.json` | Legacy selected probes. Inspect per-record failure fields; this mode does not fail the process. Prefer `trace`. |

Use RSS for cross-engine memory: Go heap excludes native allocations. The
`equal-100-history` fixture verifies identical styled history; full configured
history retains different row counts. Prompt names distinguish ordinary prompts,
erase-to-end-of-line and compaction. Closed editor/pager fixtures preserve the
inactive alternate screen. Do not turn one fixture into a general per-terminal
memory estimate.

[Coverage](coverage.md) defines capture bounds, normalization and exclusions.
Captures reject oversized input rather than silently truncating it. These bounds
belong to the oracle, not the engines. Rendering, child processes, PTYs and HTTP
remain outside these emulator measurements.
