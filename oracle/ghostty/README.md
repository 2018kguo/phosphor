# Compare Phosphor with libghostty-vt

This tool asks whether the same PTY bytes produce the same terminal state, and what each engine costs in time and memory. It measures headless emulation; fonts, rendering and whole-daemon latency are outside the experiment. Production Phosphor needs neither cgo nor Ghostty.

Start with the [idle-state and compatibility results](results/2026-10-27-round6/README.md) and [comparison contract](coverage.md). Earlier phases cover [idle rows](results/2026-10-27/README.md), [metadata pools](results/2026-10-27-round2/README.md), [Unicode/checkpoints](results/2026-10-27-round3/README.md), [gap audit](results/2026-10-27-round4/README.md) and [shell prompts](results/2026-10-27-round5/README.md). They describe their measured revisions, not the current release.

## Build

The comparison pins Ghostty `46e9dc9a927b2e7e7cba79e6902c176127dbdf2c` and ReleaseFast. Use the Zig toolchain required by its [build manifest](https://github.com/ghostty-org/ghostty/blob/46e9dc9a927b2e7e7cba79e6902c176127dbdf2c/build.zig.zon). From a clean native checkout:

```sh
git clone https://github.com/ghostty-org/ghostty.git /tmp/ghostty-comparison
git -C /tmp/ghostty-comparison checkout 46e9dc9a927b2e7e7cba79e6902c176127dbdf2c
cd /tmp/ghostty-comparison
zig build -Demit-lib-vt -Doptimize=ReleaseFast -j4
```

Then, from Phosphor's `oracle/` directory on macOS:

```sh
export GHOSTTY_SOURCE=/tmp/ghostty-comparison
export CGO_CFLAGS="-I$GHOSTTY_SOURCE/include"
export CGO_LDFLAGS="-L$GHOSTTY_SOURCE/zig-out/lib -lghostty-vt -Wl,-rpath,$GHOSTTY_SOURCE/zig-out/lib"
export PHOSPHOR_DIR="$(cd .. && pwd)"
go build -tags ghostty -o /tmp/phosphor-ghostty ./ghostty
go test -tags ghostty ./ghostty
/tmp/phosphor-ghostty -mode trace -traces ghostty/traces.json > traces.jsonl
```

Use that same checkout to generate tables and build the linked library. Native build-info omits commit metadata; the verified source pin identifies the measured build.

## Choose a measurement

Commands emit JSON lines. Keep failures as well as successes.

| Mode / arguments | What it measures |
|---|---|
| `check -probes ghostty/probes.json` | Selected text/cursor/mode probes. Legacy mode does **not** fail the process; inspect `pass`, `restored_view_equal`, `continued_view_equal`. |
| `trace -traces ghostty/traces.json` | Cells, replies and future state after every action, against untouched controls. Exits 1 on a parity failure. `-trace-chunk 1` or `7` splits writes without moving observation points. |
| `cursor-replies` | Raw cursor-query answers. |
| `throughput -duration 200ms` | Five rounds alternating engine order, at 120×40. `COMPARE_STREAM=logs` /`COMPARE_CHUNK=1024` select a case. |
| `state -duration 300ms` | View, repaint, snapshot and restore costs. Native binary snapshots and ANSI export have different contracts. |
| `memory -engine phosphor -workload prompt -n 1024` | Live retained heap and process RSS. Run each sample in a fresh process; repeat at least three times and use multiple counts. |
| `layout` | Struct size and offsets; these alone do not measure memory footprint. |
| `width-table` /`widths` | All native width 0/2 scalar ranges /differences from the baseline runewidth dependency; excludes surrogates, not a grapheme test. |
| `grapheme-properties` | Audit `[U+0600, scalar]`; consumed 2/width 1 identifies neutral continuation under Prepend. Native control behavior is unspecified; this is pinned-build evidence. |

Throughput uses generated agent TUI/log/progress streams and real editor/pager recordings; missing recordings are errors. Correctness uses original 100×30 recording geometry. `ab.sh OLD NEW OUTPUT.jsonl` alternates old Phosphor, new Phosphor and native for five rounds. Build both revisions with the same driver selectors/data; baseline overlays must leave production code untouched. Wall and process CPU time include GC/cgo/native work. Short shared-host samples cannot resolve small differences reliably.

## Read memory results

Use RSS for cross-engine comparisons: the native Go heap contains only its wrapper. Both engines get 2000 lines /256 KiB of history, but Phosphor counts serialized logical lines and native counts physical rows/prunes pages. Compare `equal-100-history` before interpreting full-history results. Native page estimates omit metadata and can exceed RSS when pages are touched lazily.

| Workload family | Retained state |
|---|---|
| `blank`, `prompt`, `colored-prompt`, `saved-cursor` | Small idle states. `prompt-compact` /`colored-prompt-compact` invoke Compact. |
| `prompt-erase`, `colored-prompt-erase`, plus `-compact` variants | Shell prompts followed by default-background EL, with zero history. They must not enter the default log workload. |
| `cleared[-compact]`, `full-window[-compact]` | Filled screen erased to a prompt /meaningful cells retained. Compaction uses Compact or native idle compression. |
| `rich-cleared[-compact]` |48 distinct 2 KiB URLs and combining characters, erased without history; isolates stale metadata. |
| `closed-{vim,nvim,nano,less,man}[-compact]` | Completed recordings with inactive ALT retained; see [costs and exact build](results/2026-10-27-round6/closed-memory.md). |
| `equal-100-history`, `full`, `compressed` | Equal 100 styled lines /full configured history /native idle compression. Phosphor already stores compact VT history. |

`compaction_applied` distinguishes old binaries without Compact. Earlier compaction freed dead rows; the measured candidate also avoids initial full-width prompt allocation and frees zero tails. Freed heap need not immediately leave RSS; the tool does not impose a production GC policy.

Trace capture rejects oversized input rather than truncating: fixtures 2 MiB/512 cases/256 actions per case/1 MiB written per case; geometry 1..256×1..128; native capture 256 total history+screen rows/32768 cells/1024 codepoints per cell/4096 URI bytes per cell/1 MiB replies. These are oracle bounds, not engine capacities. See [coverage](coverage.md) for scopes, policies and export limits.
