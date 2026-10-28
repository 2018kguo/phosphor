# What the native comparison covers

The target is a headless PTY terminal: text, cursor, history, input modes, replies and future output after checkpoint/attach. Passing the finite corpus does not prove universal compatibility. Rendering, fonts/GPU work, accessibility, selection UI, images, audio, notifications and clipboard-provider integration are outside it. Regular Go suites supply independent expectations; performance modes exclude correctness capture allocations.

## Observations

| Surface | Compared state and limits |
|---|---|
| Cells/styles | Full cluster text, width, wide tails, every blank; typed default/indexed/RGB foreground/background/underline, all attributes, underline style and protection. Zero cells differ from literal spaces. Spacer heads project as empty width 1 because Phosphor's public API does not distinguish them. Native background-only cells use public getters. |
| Links | Exact per-cell URI, including wide tails after erase/resize/Compact. Native ABI omits OSC 8 IDs; Phosphor Snapshot tests preserve those separately. |
| Cursor/wrap | Physical x/y, pending wrap, visible/shape/blink and row continuation. Default shape resolves to configured block; outside-origin positions are not clamped. Last-row forward-wrap flags are exposed through suffixes/reflow. |
| Shell prompts | Raw primary/continuation row annotations; semantic cursor/redraw state through future operations. Native construction disables redraw; RIS enables it. DECSC does not save semantic content. Blank marks, history pop, ED 2/EL 2 and width/height changes are compared. ANSI can represent an unmarked prompt cursor as input with equal headless effects; Snapshot retains the enum. GUI per-cell semantic classification is outside scope. |
| Modes | Independent stored bits and effective mouse tracking/encoding, observed through native input encoding (press/release/motion and all five encodings). Mode 2027 and margins are also queried in VT. No invented precedence or last-set order. |
| Charsets/saves | Designation/shift and saved-cursor behavior through written suffixes; not inspection of every native private field. |
| History | Styled physical cells and wrap flags at current width, below both retention budgets. Logical/physical counters may differ. Alternate history is omitted by both APIs; pruning requires separate bounded tests. |
| Replies | Raw DSR/CPR, DECRQM, DECRQSS and keyboard bytes, without replacements or coordinate normalization. Identity and host defaults have separate scopes. |
| Continuation | Write/Resize/Compact, exact binary Snapshot and separate ANSI replay, checked against each engine's untouched control through later writes/resizes. Failures name the exporter; direct parity remains independent. Combined `pass` requires all comparisons. |
| Unicode | Public CodepointWidth matches all 1,112,064 valid scalars. Primary+64 suffix capacity is checked with two-byte accents/four-byte selectors, mode 2027 off/on/mixed and future operations. The helper separately checks 766 Unicode 17 cases with 25 explicit native control-tailoring differences; that is not all terminal dispatch. |

Mode array order: application cursor, application keypad, bracketed paste, focus, autowrap, insert, newline, origin, reverse video, synchronized output, X10/normal/button/any-event mouse, UTF8/SGR/URXVT/SGR-pixel encoding, 132-column selection, permission to switch columns, smooth scroll, mode 47, mode 1047, mode 1049. Stored bits are not collapsed to improve agreement.

`shared_palette: true` initializes both hosts through public APIs with Phosphor's palette, including the native replay host. Configured reset tests can then compare raw replies; the original default-color fixture retains each product's own colors.

## Scopes and remaining differences

| Fixture scope | Meaning |
|---|---|
| `parity` | Any difference remains an unresolved comparison failure and makes trace exit 1. |
| `identity` | Product/device identities may differ; both raw answers remain. |
| `deliberate` | Host configuration differs, currently only initial colors. Configured color commands belong in parity. |
| `limit` | Declared bounded capacity; no current fixture needs it after matching cluster capacity. |

Scope changes need a contract explanation. Finding a failure is not a reason to exclude it. Two classes remain in parity: `dec-graphics-underscore-policy` keeps Phosphor's [xterm blank mapping](https://raw.githubusercontent.com/ThomasDickey/xterm-snapshots/master/charsets.c) versus native underscore; `cursor-1047-resize-background-limit` and its growth variant retain ANSI's mixed-background ALT loss on forced 1047 exit. Returning to MAIN with all independent mode bits preserved erases ALT with one background. Exact Snapshot preserves the original cells and their future use. [Final results](results/2026-10-27-round6/README.md) retain every failure, plus other portable ANSI limits such as REP and some outside-origin/narrowed-cluster states.

Native tests cover 216 three-operation screen-mode combinations and saved-mode restoration. All 47/1047/1049 bits are separate from active-screen state. DECSC has one slot per screen, shared with 1049; it saves single shift/underline style, not the live link. Screen transitions end links;1049 exit retains ALT.

## Translated suites and storage checks

The Go-translated Ghostty suite comes from `521cf2c`; attribution/license: [NOTICE](../../testdata/conformance/NOTICE.md). At `7bc614a`,549 cases meant 459 passing/6 known failures/84 excluded:42 grapheme,15 palette,12 link/semantic/OSC,6 reply,5 DECCOLM,2 image,2 charset. The newer pin has 460 `Terminal.zig` tests and 148 `stream_terminal.zig` tests plus other modules; these are different denominators.

The current translation passes 524, has zero known failures and excludes 25:15 palette,6 reply,2 image,2 charset. Five former link cases now check text/URIs and seven prompt cases check text/cursor/history/raw row marks. Native fixtures exercise every non-image group with raw replies and continuation, without turning unsupported commands into passes.

Prompt/wrap flags share one byte per annotated row. Private history markers are NUL +canonical zero uvarint +`P` +kind 0/1/2; positive uvarints remain sparse-hole runs. Unknown/truncated/noncanonical/out-of-range markers and markers in legacy snapshots are rejected; unannotated history gains no bytes.

Width changes reset tabs to the default eight-column interval; height changes retain custom stops. Five unchanged WezTerm expectations preserve tabs across width changes and remain explicit policy conflicts.

Native traces are independently written public-API sequences, not copied third-party tests. Future translations must retain revision/licensing. Do not copy GPL esctest source into this MIT project; existing esctest-style inputs were independently written.
