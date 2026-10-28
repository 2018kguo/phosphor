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
| Styles | Default/indexed/RGB foreground/background/underline color, bold/faint/italic/blink/inverse/invisible/strike/overline, underline style, protection | Native background-only cells are decoded through public cell getters. Underline color is retained on both sides. |
| Hyperlinks | Exact URI per cell, including wide tails, after overwrite/erase/resize/Compact | Native C API exports URI, not OSC8 ID. Phosphor snapshot tests own exact ID preservation. |
| Wrap and cursor | Row continuation, exact physical x/y, pending wrap, visible/shape/blink | Constructor-default shape resolves to the configured block in both engines. Cursor outside enabled origin margins is retained. Last-row forward-wrap flags are exercised through suffixes/reflow rather than a separate phosphor getter. |
| Shell prompts | Raw per-row primary/continuation annotation through each public API; semantic cursor and redraw policy through future operations | Native constructor redraw is disabled; RIS enables it. DECSC does not save semantic content. Blank annotations, history pop, ED2, EL2, width-only and height-only resize are exercised without normalization. ANSI replay's unmarked prompt cursor may use input state with the same headless text effects; per-cell GUI semantic classification is outside this contract. Binary snapshots retain the exact enum. |
| Modes | Application cursor/keypad, bracketed paste, focus, autowrap, insert, newline, origin, reverse video, sync output, independent mouse mode bits, effective mouse tracking/encoding, kitty flags | Array order is listed below. Native effective mouse state is observed through the public input encoder using distinguishable press/release/motion reports and all five encodings. Mode2027 and margin state are also queried through VT replies. |
| Charsets and private cursor state | Designation/shift and saved-cursor effects in written suffixes | No claim to inspect every private native struct field. Control branches expose lost state after restore. |
| History | Styled physical cells at the current width plus continuation flags, independent of logical/physical storage counters | Tests stay below both history retention budgets. Capacity pruning differences need separate bounded tests. Alternate screen history is omitted in both public projections. |
| Queries | Raw PTY bytes, including DSR/CPR, DECRQM, DECRQSS, keyboard queries | Identity answers and default host colors have separate scopes. No string replacement or cursor-coordinate normalization. |
| Continuation | Per-engine untouched control after Write, Resize, Compact, snapshot restore and ANSI replay | Both engines use their exact binary snapshot APIs. ANSI replay is measured separately. Snapshot effects include subsequent writes/resizes, not only the first restored picture. Snapshot/repaint failures name the affected engine. |
| Unicode capacity | Native per-scalar width audit plus split cluster traces | The public CodepointWidth matches every one of 1,112,064 valid scalars. Both engines retain the primary codepoint plus up to 64 suffix codepoints. Boundary fixtures use two-byte accents and four-byte supplementary selectors with mode2027 off/on/mixed and subsequent Compact, resize, snapshot, ANSI replay and writes. The segmentation helper checks all 766 Unicode 17 corpus cases separately, with 25 explicit control-tailoring differences; it does not prove terminal dispatch for every possible sequence. |

The mode array order is: application cursor, application keypad,
bracketed paste, focus events, autowrap, insert, newline, origin,
reverse video, synchronized output, X10 mouse, normal mouse, button
mouse, any-event mouse, UTF8 mouse, SGR mouse, URXVT mouse, SGR-pixel
mouse, DECCOLM132-column selection, permission to switch columns, and
smooth scrolling. Stored mode bits are compared as reported; the oracle does not
collapse native settings to make them agree.

Each trace also reports direct parity between untouched engines separately
from phosphor's and Ghostty's continuation checks. The untouched controls
never restore snapshots, replay ANSI or compact state. A native exporter
bug therefore remains a recorded failure without making the direct parity
count ambiguous. The combined `pass` still requires all comparisons to pass.

Color protocol fixtures can set `shared_palette: true`. Before any input,
both hosts use phosphor's default palette through their public configuration
APIs. Native replay receives that host configuration before the bytes are
fed. This makes reset behavior comparable while preserving raw replies;
the original host-default fixture retains each product's own defaults.

Scope is fixed in fixture metadata, with these meanings:

- `parity`: a difference is an unresolved compatibility failure and makes
  the command fail.
- `identity`: different product names and device identities are expected;
  both byte strings are still recorded.
- `deliberate`: host configuration differs, currently only the initial
  color scheme. Configured color commands belong in parity tests.
- `limit`: a declared bounded-capacity difference. No current fixture needs
  this scope after the cluster capacities were matched.

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

The current Go translation passes 517 cases, keeps no known failures
and excludes 32 cases. Five former hyperlink exclusions now
assert exact text and per-cell URIs. Remaining exclusions are 15 palette,
seven semantic prompt, six reply, two image and two charset
cases. Native fixtures exercise all non-image groups with raw replies and
continuation checks; they retain failures rather than marking unsupported
commands as passing. Images remain outside the headless text contract.

Prompt and wrap metadata share one byte per annotated screen row. History
uses its existing bounded VT representation with typed private row markers:
NUL, canonical zero uvarint, `P`, then kind 0/1/2. Positive uvarints retain
the existing sparse-hole grammar. Unknown, truncated, noncanonical or
out-of-range markers are rejected, and released v1 snapshots cannot contain
the new grammar. Ordinary history without annotations gains no marker bytes.

Changing width resets tab stops to the default eight-column interval, as
the pinned native engine documents and implements. Height-only resizing
preserves custom stops. Five unchanged translated WezTerm expectations
disagree because WezTerm keeps custom stops through width changes; those
are explicit policy divergences, not universal terminal compatibility.

The native traces are independently written input sequences using the
public C ABI and public phosphor API. They do not copy a third-party
test implementation. Future upstream translations must retain source
revision and licensing; GPL esctest source must not be copied into this
MIT project. Existing esctest-style cases are independently written.
