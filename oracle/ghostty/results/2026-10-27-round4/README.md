# Fourth comparison: find the remaining gaps

This intermediate audit expanded the corpus and exposed missing commands, policy differences and exporter failures. It was not a release pass. Native: `46e9dc9a927b2e7e7cba79e6902c176127dbdf2c`, macOS/arm64, `zig build -Demit-lib-vt -Doptimize=ReleaseFast -j4`; build-info omits the commit, so the source pin identifies the build.

| Dataset | Historical result |
|---|---|
| [Baseline](traces-v030.jsonl) |150/174 original parity actions. |
| [Mouse bits/tabs](traces-modes-tabs.jsonl) |158/174, including independently observed effective mouse state. |
| [Expanded, before fixes](traces-expanded-before.jsonl) |112 cases/340 actions: direct 293, Phosphor continuation 338, native 334, combined 289 passes. Trace exited 1. |
| [Old policies](traces-existing-policies.jsonl) |Pre-fix differences, including prints after autowrap was re-enabled. |
| [Clipped/policy expansion](traces-policies-clipped.jsonl) |149 cases/468 actions,21 mode bits (including 3/40/4), eight styled cases resized from four columns to one; 12 ALT transitions. Only two semantic annotation cases still differed directly; exporter failures stayed separate. |

Twelve cluster-boundary cases passed:64/65 suffixes, two-byte accents/four-byte selectors, mode 2027 off/on/mixed, Compact/resize/snapshot/ANSI/future writes. Native keeps primary+64 and ignores extra suffixes without moving the cursor; the old capacity exclusion became parity.

Mouse press/release/button/free-motion and a column beyond byte coordinates distinguished tracking modes and all five encodings through the public native encoder. Native's own ANSI export lost effective state in three fixtures; Phosphor passed. Two Phosphor continuation failures were outside-origin ANSI restoration. Untouched controls kept these separate from direct parity.

Build toolchains are recorded in [environment metadata](../environments.json).

## Commands found by the translated-suite audit

Five former hyperlink exclusions gained exact text/URI checks:507 passes/5 known differences/37 excluded. The remaining groups were:

| Group | Finding |
|---|---|
| 15 palette/dynamic color | Standard OSC 4/104 and dynamic set/query pass. Kitty OSC 21 set/reset/query is missing. Fixtures supply an identical host palette before input; raw reply bytes remain unchanged. |
| 7 semantic prompts | OSC 133 A/L/N fresh-line cursor movement is missing; P and N at column zero already agree. Prompt annotations also change native resize/reflow and ED 2 history retention. |
| 6 replies | DECRQSS SGR passes. XTGETTCAP Smulx should advertise `\\E[4:%p1%dm`; phosphor currently replies unsupported. ANSI DECRQM mode 1/19 are permanently reset in native (status 4), and DEC mode 4 is recognized/reset (status 2). TN has a separate identity scope. |
| 5 DECCOLM | Native recognizes modes 3/40; mode 40 gates 80/132-column switching. Mode 3 reset with mode 40 set changes width to 80, erases display with current background, clears pending wrap and resets margins while preserving the pen. |
| 2 national charsets | British designation A maps # to pound; single shift must be consumed by a combining mark. Native REP maps the saved input scalar under the current charset. |
| 2 images | Outside the declared headless text contract; not counted as newly passing. |

All five old translated native differences were subsequently closed: dropping unprintable wide characters with autowrap off, preserving pending wrap through mode/tab changes and one-column dropped-wide behavior. Conflicting Alacritty/libvterm references stayed unchanged. Reflow cleared clipped-wide attributes but retained narrowed-cluster attributes/cursor mapping; cases also covered same-size column switches, smooth scroll, sync resize and mode 69 reset.

Width changes reset tabs per native Terminal.resize; height changes keep them. Five unchanged WezTerm expectations retain conflicting width-resize policy.

[Comparison review](normalization-review.md) records hidden-state masks and concrete failures at the reviewed heads. [Cold-prefix measurements](cold-prefix-measurements.md) record the separate idle-memory experiment. Build/run with the [driver](../../README.md); these files retain intermediate failures rather than current-release claims.
