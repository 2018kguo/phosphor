# Develop and verify Phosphor

The library's toolchain requirement is in [go.mod](../go.mod).
It is pure Go; ordinary builds and tests do not need a native terminal
library, Zig or cgo. The interactive examples
and optional comparison tools have separate Go modules and dependencies.

## Run the demo

From a clone in an interactive terminal:

```sh
cd examples
go run ./mirror vim
```

Replace `vim` with a command and arguments of your own. The mirror starts a
PTY, feeds output to Phosphor and redraws from cells and dirty-row reports.
The [examples module](../examples/go.mod) declares its own toolchain requirement.

## Run checks

From the repository root:

```sh
go test ./...
go vet ./...
go test -race ./...
```

Check the example and Go differential modules separately:

```sh
(cd examples && go build -o /dev/null ./... && go vet ./...)
(cd oracle && go test ./... && go vet ./...)
```

Print the external conformance counts:

```sh
VT_CONF_REPORT=1 go test -run 'Conformance|Libvterm|Alacritty' -v
```

Known terminal-policy differences have explicit reasons in
`conformance_expectations_test.go`; unexpected failures and unexpectedly passing
known cases fail the suite. Exclusions do not count as passing coverage.
Preserve upstream expectations and the [source notices](../testdata/conformance/NOTICE.md).

For a bounded snapshot continuation fuzz run:

```sh
go test -run '^$' -fuzz '^FuzzSnapshotContinuation$' -fuzztime 30s
```

Other targets cover raw bytes, chunking, replies, Unicode, repaint, damage,
compaction and sparse rows. [CI](../.github/workflows/test.yml) lists the
maintained targets and tests the minimum supported and current Go toolchains.
A change to storage or persistence should preserve future writes and resize, not just a
restored first frame. ANSI export has [declared limits](reference.md#attach-a-client):
mechanism tests assert those losses explicitly and keep exact Snapshot checks
separate. Do not skip a failing input or treat a known loss as native parity. Checkpoints from an earlier writer are kept under
[`testdata/snapshot-v1`](../testdata/snapshot-v1/README.md).

## Measure your workload

The root package benchmarks parsing, recorded PTY output, checkpointing,
replay and memory:

```sh
go test -run '^$' -bench . -benchmem
go test -run '^$' -bench '^BenchmarkWriteRecordedPTY$' -benchmem -count 5
```

The separate Go comparison uses the same streams across three libraries:

```sh
cd oracle
go test -run '^$' -bench Compare -benchmem -count 5
```

Keep terminal geometry, byte chunks, history policy and retained state equal.
Use fresh processes for RSS and retained heap; native allocations do not
appear in Go heap counters. Include closed-editor recordings when that is
the idle state you intend to host. [Benchmarks](benchmarks.md) explains the
current results and why prompt memory is not a universal per-session cost.

## Compare with libghostty-vt

The optional driver uses public native APIs for compatibility, throughput,
state costs and memory. Follow the pinned build and environment instructions
in the [oracle README](../oracle/ghostty/README.md). Native builds require
the pinned build's Zig toolchain, cgo headers and the built library; none is a
production dependency of Phosphor.

Run named traces with whole writes and byte chunks, keep raw replies and
cells unchanged, and compare future behavior against untouched controls.
The trace command exits nonzero for remaining parity failures, including
documented policy/export differences. An exit status alone does not tell
you which engine lost state. The [comparison contract](../oracle/ghostty/coverage.md)
describes the captured fields and separate continuation counts.

## Repository guide

| Open | For |
|---|---|
| [terminal.go](../terminal.go), [read.go](../read.go), [style.go](../style.go) | Stream processing and public screen values |
| [resize.go](../resize.go), [compact.go](../compact.go) | Reflow, history and idle storage |
| [modes.go](../modes.go), [reply.go](../reply.go) | Terminal modes, queries and replies |
| [serialize.go](../serialize.go), [snapshot.go](../snapshot.go) | ANSI attach and exact checkpoints |
| [example_test.go](../example_test.go), [mirror](../examples/mirror) | Runnable API use and a renderer |
| [testdata](../testdata), `*_test.go` | Recordings, references and regression checks |
| [oracle](../oracle) | Optional differential and native comparisons |

Go files in the root implement one public package. Related regressions belong
in the existing subject suite: `history_test.go`, `unicode_test.go`,
`repaint_test.go`, `modes_test.go` and the other matching files. Conformance
sources share the runner in `conformance_test.go`; translated Ghostty cases
live in `conformance_ghostty_test.go`. The optional native tests use shared
measurement, protocol and repaint suites as well. Preserve case names, upstream expectations,
source notices and explicit exclusions when extending a suite.

Keep maintained docs in `docs/` and reproducible inputs in `testdata/`.
Write temporary benchmark output to the ignored `.bench/` directory. The
[benchmark guide](benchmarks.md) links the published evidence bundles; raw run
logs, old charts and per-experiment reports do not belong in the source tree.

X11 color names can be regenerated with `go run ./internal/colorgen`;
`-source /path/to/rgb.txt` uses an offline source. The generator checks its
input checksum. Preserve [the data's notices](../licenses/X11.txt).
