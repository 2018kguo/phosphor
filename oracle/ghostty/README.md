# Compare with libghostty-vt

This optional driver feeds the same bytes to phosphor and libghostty-vt.
It measures emulator work, not a whole PTY daemon. Normal builds and tests
do not need Ghostty or cgo; this directory requires the `ghostty` build tag.

The October 2026 experiment pins Ghostty to
`46e9dc9a927b2e7e7cba79e6902c176127dbdf2c` and uses Zig 0.17.0:

```sh
git clone https://github.com/ghostty-org/ghostty.git /tmp/ghostty-comparison
git -C /tmp/ghostty-comparison checkout 46e9dc9a927b2e7e7cba79e6902c176127dbdf2c
cd /tmp/ghostty-comparison
zig build -Demit-lib-vt -Doptimize=ReleaseFast -j4
```

From `oracle/`, build the driver against that checkout. On macOS:

```sh
export GHOSTTY_SOURCE=/tmp/ghostty-comparison
export CGO_CFLAGS="-I$GHOSTTY_SOURCE/include"
export CGO_LDFLAGS="-L$GHOSTTY_SOURCE/zig-out/lib -lghostty-vt -Wl,-rpath,$GHOSTTY_SOURCE/zig-out/lib"
export PHOSPHOR_DIR="$(cd .. && pwd)"
go build -tags ghostty -o /tmp/phosphor-ghostty ./ghostty

/tmp/phosphor-ghostty -mode check -probes ghostty/probes.json > checks.jsonl
/tmp/phosphor-ghostty -mode trace -traces ghostty/traces.json > traces.jsonl
/tmp/phosphor-ghostty -mode trace -trace-chunk 1 -traces ghostty/traces.json > byte-traces.jsonl
/tmp/phosphor-ghostty -mode trace -trace-chunk 7 -traces ghostty/traces.json > seven-byte-traces.jsonl
/tmp/phosphor-ghostty -mode cursor-replies > cursor-replies.jsonl
/tmp/phosphor-ghostty -mode width-table > native-widths.jsonl
go test -tags ghostty ./ghostty
/tmp/phosphor-ghostty -mode throughput -duration 200ms > throughput.jsonl
/tmp/phosphor-ghostty -mode state -duration 300ms > state.jsonl
/tmp/phosphor-ghostty -mode memory -engine phosphor -workload prompt -n 1024
/tmp/phosphor-ghostty -mode memory -engine ghostty -workload prompt -n 1024
```

Each command emits JSON lines. `COMPARE_STREAM=logs` and
`COMPARE_CHUNK=1024` restrict throughput sampling. Each throughput case
runs five rounds, alternating engine order. Inputs are ptyd's deterministic
synthetic agent output, logs and progress bars, plus actual editor/pager
recordings. Throughput uses 120x40 as the existing benchmarks do;
recording correctness uses its original 100x30 geometry. Missing recordings
are an error.

Memory commands must run in fresh processes. Use three or more repeats
and multiple terminal counts. `blank`, `prompt`, `colored-prompt` and
`saved-cursor` measure the corresponding idle terminals;
`equal-100-history` retains the same 100 short, styled history lines in
both engines. `full` fills the configured history, while `compressed`
additionally invokes Ghostty's idle compression. Phosphor already stores
history as compact VT bytes and has no extra compression step.

`cleared` fills every row and erases back to a prompt. `cleared-compact`
then calls phosphor's `Compact()` or Ghostty's idle compression API.
`full-window` and `full-window-compact` retain the filled screen, showing
the bound on what can be freed while keeping meaningful cells. The
`compaction_applied` field distinguishes older phosphor binaries that
have no `Compact` method. `-mode layout` reports actual struct size and
field offsets; memory comparisons still use retained allocations and RSS.

`rich-cleared` writes 48 distinct links (2 KiB targets) and combining
characters on one row, then erases back to the prompt, without history.
`rich-cleared-compact` also runs idle compaction. These isolate retained
metadata from row storage; they do not represent a typical shell prompt.

`ab.sh OLD_BINARY NEW_BINARY OUTPUT.jsonl` alternates the old phosphor,
new phosphor and Ghostty in five rounds. Build both binaries from this
same driver, using a temporary Go modfile to replace the baseline's
phosphor dependency with v0.1.0. Throughput reports wall time and process
CPU time; CPU includes Go GC and native work. Short samples on a shared
host do not establish small differences as regressions or wins.

Both engines are configured with 2000 lines / 256 KiB history. Their
budgets have different meanings: phosphor counts serialized logical lines,
Ghostty counts physical rows and prunes whole pages. The driver reports
retained history and native page estimates alongside process RSS and Go
heap. Native page estimates omit some metadata and can exceed RSS on
macOS because physical pages are touched lazily. Compare the equal-history
case before interpreting the full-history numbers.

Correctness probes check visible text, cursor and selected modes. They
are explicit feature cases, not a compliance percentage. Repaint and
snapshot continuation are separate tests: Ghostty's binary snapshot is
stronger than its active-screen VT formatter. The latter needs parser
continuation appended for attach, and still does not export both screens
and saved cursors in one call. Phosphor's regular tests additionally
verify styles, links, wrapping, bounded memory and replay behavior.

All output is evidence, including failed cases. The legacy `check` mode
does not turn an engine's failed probe into a process exit: inspect the `pass`,
`restored_view_equal` and `continued_view_equal` fields.

The `scroll-clear-outside-origin` replay case exposes a shared VT limit.
CSI 22 J can leave the physical cursor above or left of an enabled origin
region. Both engines' VT exports clamp it back into that region, moving
subsequent output; Ghostty's binary snapshot preserves it. The driver
compares the physical cursor and continued text without normalizing them.

`trace` compares a deeper [text-model contract](coverage.md). It records
full cell text and widths, typed colors and attributes, underline styles
and colors, hyperlink targets, protected cells, row wraps, physical
cursor and appearance, modes, title, history cells and raw PTY replies.
Each action is checked. An untouched control for each engine checks
continuation after snapshots, ANSI replay and compaction, including
later writes and resizes. Cells are collected in one C call per state;
benchmark modes keep their existing extraction path.

`trace` exits 1 when any `parity` action fails. Identity strings,
deliberate host defaults and documented capacity limits have separate
scope counts and retain their raw failures. Counts describe these named
traces; they are not a compliance percentage. JSON includes input bytes,
geometry, action index, differing cells and raw reply hex. `-trace-chunk`
splits each Write action at byte boundaries; it does not change where
the trace observes state.

Fixture input is limited to 2 MiB, 512 cases, 256 actions per case and
1 MiB written per case. Dimensions are 1..256 columns and 1..128 rows.
Native capture rejects more than 256 total history+screen rows, 32768
cells, 1024 codepoints per cell, 4096 URI bytes per cell or 1 MiB of PTY
replies. It reports capture failure instead of truncating state. These
are oracle bounds, independent of either engine's supported limits.

`width-table` emits all native scalar width-0/2 ranges (other scalars
have width 1), excluding surrogates. `widths` instead lists differences
against the baseline runewidth dependency. Neither measures grapheme
folding. Table generation requires `GHOSTTY_SOURCE` to point at the
clean pinned checkout used to build the linked library. The library
currently reports `0.1.0-dev` without commit metadata, so output records
that version separately from the verified source revision. The source
and linked library must come from the same build command above.
