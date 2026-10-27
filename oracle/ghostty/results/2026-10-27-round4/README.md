# Expanded native compatibility audit

Native source: `46e9dc9a927b2e7e7cba79e6902c176127dbdf2c`, macOS arm64,
Zig 0.17.0, `zig build -Demit-lib-vt -Doptimize=ReleaseFast -j4`.
Library build-info version is `0.1.0-dev`; the source pin identifies the build.

- `traces-v030.jsonl`: released v0.3.0 reproduced 150/174 original parity actions.
- `traces-modes-tabs.jsonl`: independent mouse bits/1016 and width-reset tabs
  passed 158/174 original actions, including newly observed effective mouse state.
- `traces-expanded-before.jsonl`: 112 cases / 340 parity actions after underline
  color, native cluster capacity and released mouse snapshot conversion were
  integrated, before history, national charset and the newly exposed commands.
  Direct untouched-engine parity: 293/340. Phosphor continuation: 338/340.
  Native continuation: 334/340. Combined: 289/340. The command correctly exits 1.

The 12 new native cluster boundary cases all pass. They exercise 64 and 65
suffix codepoints, two-byte accents and four-byte supplementary selectors,
mode2027 off/on/mixed, Compact, resize, exact snapshot, ANSI repaint and
subsequent writes. Native retains primary +64 suffix codepoints and silently
ignores additional suffixes, with no cursor movement. The previous long-cluster
`limit` fixture is now a parity fixture.

Effective mouse state is independently observed through the public native input
encoder, with a fixed renderer geometry. Press/release/button-motion/free-motion
distinguish all tracking modes, and a column beyond byte-coordinate capacity
distinguishes all five report encodings. Native mode bits are still captured
directly; no mode-bit precedence or last-set order is invented by the oracle.

Native's own ANSI export loses effective mouse state when other mode bits remain
enabled. Three new mouse fixtures retain that failure, with explicit native
repaint and untouched-control differences. Phosphor's corresponding roundtrips
pass. The two phosphor continuation failures are the already exposed outside-
origin ANSI cursor restoration issue. Direct parity compares untouched controls,
so these export failures do not contaminate the direct-input count.

## Audit of the 42 translated exclusions

Five hyperlink exclusions now assert exact text and per-cell URI, bringing the
translated Ghostty suite to 507 passing /5 known differences /37 excluded.
New native cases exercise the remaining non-image groups:

| Group | Finding |
|---|---|
| 15 palette/dynamic color | Standard OSC4/104 and dynamic set/query pass. Kitty OSC21 set/reset/query is missing. Fixtures supply an identical host palette before input; raw reply bytes remain unchanged. |
| 7 semantic prompts | OSC133 A/L/N fresh-line cursor movement is missing; P and N at column zero already agree. Prompt annotations also change native resize/reflow and ED2 history retention. |
| 6 replies | DECRQSS SGR passes. XTGETTCAP Smulx should advertise `\\E[4:%p1%dm`; phosphor currently replies unsupported. ANSI DECRQM mode1/19 are permanently reset in native (status4), and DEC mode4 is recognized/reset (status2). TN has a separate identity scope. |
| 5 DECCOLM | Native recognizes modes3/40; mode40 gates 80/132-column switching. Mode3 reset with mode40 set changes width to80, erases display with current background, clears pending wrap and resets margins while preserving the pen. |
| 2 national charsets | British designation A maps # to pound; single shift must be consumed by a combining mark. Native REP maps the saved input scalar under the current charset. |
| 2 images | Outside the declared headless text contract; not counted as newly passing. |

## Remaining five translated native differences

Two disabled-autowrap wide-character cases retain a deliberate difference:
native drops a wide character that does not fit; phosphor and libvterm back up
to fit it. Another case concerns native setting pending-wrap in a one-column
terminal for an unprintable wide character. Two CBT cases retain native pending
wrap while autowrap is disabled; the existing translated suffix cases that print
before enabling wrap agree. These distinctions need explicit future-observable
probes before deciding whether to change phosphor's policy.

Column-width changes now reset tab stops to defaults, matching native's explicit
Terminal.resize contract. Five unchanged WezTerm translated expectations preserve
custom tabs through width changes and are documented as conflicting policies.
Height-only resize preserves custom tabs in both engines.
