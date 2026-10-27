# Native text-model comparison

Use `traces.json` and `-mode trace` for headless text-model differences.
Use the regular Go conformance suites for larger sets of independent
expected behaviors. Use throughput/state/memory modes for performance;
correctness capture allocates and is intentionally outside those timers.

The target is a terminal state machine for PTY hosting: text and cursor
state, history, input modes, query replies, and exact continuation after
an attach checkpoint. A GUI renderer, fonts, GPU work, accessibility,
selection UI, images, audio, desktop notifications and clipboard-provider
integration are outside this contract. Passing every current trace means
only that this finite corpus passed. It does not establish universal
terminal compatibility or equality to every terminal's policy.

| State or behavior | What the oracle observes | Current boundary |
|---|---|---|
| Cells | Full base+combining/cluster text, physical width, wide tails, every blank | Empty cells and literal spaces remain different. Spacer heads project as empty width-1 cells because phosphor's public API does not distinguish them. |
| Styles | Default/indexed/RGB foreground/background, bold/faint/italic/blink/inverse/invisible/strike/overline, underline style, protection | Native background-only cells are decoded through public cell getters. Underline color is observed as an explicit phosphor gap, never dropped. |
| Hyperlinks | Exact URI per cell, including wide tails, after overwrite/erase/resize/Compact | Native C API exports URI, not OSC8 ID. Phosphor snapshot tests own exact ID preservation. |
| Wrap and cursor | Row continuation, exact physical x/y, pending wrap, visible/shape/blink | Constructor-default shape resolves to the configured block in both engines. Cursor outside enabled origin margins is retained. Last-row forward-wrap flags are exercised through suffixes/reflow rather than a separate phosphor getter. |
| Modes | Application cursor/keypad, bracketed paste, focus, autowrap, insert, newline, origin, reverse video, sync output, mouse mode bits, kitty flags | Array order is listed below. Phosphor exposes a single effective mouse mode/encoding; native mode bits can remain simultaneously enabled. The discrepancy is retained. Mode2027 and margin state are also queried through VT replies. |
| Charsets and private cursor state | Designation/shift and saved-cursor effects in written suffixes | No claim to inspect every private native struct field. Control branches expose lost state after restore. |
| History | Styled physical cells at the current width plus continuation flags, independent of logical/physical storage counters | Tests stay below both history retention budgets. Capacity pruning differences need separate bounded tests. Alternate screen history is omitted in both public projections. |
| Queries | Raw PTY bytes, including DSR/CPR, DECRQM, DECRQSS, keyboard queries | Identity answers and default host colors have separate scopes. No string replacement or cursor-coordinate normalization. |
| Continuation | Per-engine untouched control after Write, Resize, Compact, snapshot restore and ANSI replay | Both engines use their exact binary snapshot APIs. ANSI replay is measured separately. Snapshot effects include subsequent writes/resizes, not only the first restored picture. Snapshot/repaint failures name the affected engine. |
| Unicode capacity | Native per-scalar width audit plus split cluster traces | The public CodepointWidth matches every one of 1,112,064 valid scalars. Phosphor's 32-byte cluster limit is explicit. Long-cluster differences use `limit` scope and retain both strings. The segmentation helper checks all 766 Unicode 17 corpus cases separately, with 25 explicit control-tailoring differences; it does not prove terminal dispatch for every possible sequence. |

The mode array order is: application cursor, application keypad,
bracketed paste, focus events, autowrap, insert, newline, origin,
reverse video, synchronized output, X10 mouse, normal mouse, button
mouse, any-event mouse, UTF8 mouse, SGR mouse, URXVT mouse, SGR-pixel
mouse. Stored mode bits are compared as reported; the oracle does not
collapse native settings to make them agree.

Scope is fixed in fixture metadata, with these meanings:

- `parity`: a difference is an unresolved compatibility failure and makes
  the command fail.
- `identity`: different product names and device identities are expected;
  both byte strings are still recorded.
- `deliberate`: host configuration differs, currently only the initial
  color scheme. Configured color commands belong in parity tests.
- `limit`: a declared bounded-capacity difference, currently a cluster
  exceeding phosphor's documented 32-byte limit.

Changing a scope requires a contract explanation. A newly found failure
does not become an exclusion merely to improve the passing count.

The old Go-translated Ghostty suite comes from `521cf2c`, with license
and attribution in `testdata/conformance/NOTICE.md`. At the `7bc614a`
baseline, its 549 cases report 459 passing, 6 known failures and 84
out-of-scope. Those exclusions include 42 grapheme, 15 palette/dynamic
color, 12 hyperlink/semantic/OSC, 6 reply, 5 DECCOLM, 2 image and 2
charset cases. A translation often checks only selected fields; it is
not a native paired run. The newer pinned source has 460 tests in
`Terminal.zig` and 148 in `stream_terminal.zig`; other Zig modules add
further tests. Those counts are not interchangeable denominators.

The current Go translation passes 502 cases, keeps five documented
differences and excludes 42 cases. The expanded native fixture corpus
continues to expose differences beyond those translated assertions.

The native traces are independently written input sequences using the
public C ABI and public phosphor API. They do not copy a third-party
test implementation. Future upstream translations must retain source
revision and licensing; GPL esctest source must not be copied into this
MIT project. Existing esctest-style cases are independently written.
