# Historical audit: what the comparison hid

Reviewed `f98890e..4e91b48` on 27 October 2026. These findings describe those heads. Later fixes and remaining limits are in the [release results](../2026-10-27-round6/README.md); they must not be read as current defects.

## Two observable failures

| Input and continuation | Uninterrupted source | ANSI replay |
|---|---|---|
|4×3: `abcd ESC7 CR X`, repaint, then `ESC8 Z` |Lines `['Xbcd','Z','']`, cursor(1,1) |`['XbcZ','','']`, cursor(3,0,pending) |
|4×2: `Q CSI2J CSI H`, repaint, then `CSI3b` |`['QQQ','']` |`['','']` |

The first lost DECSC's saved pending wrap through CUP+ESC 7; Snapshot preserved it. The second exposed ANSI's missing REP register, also omitted from stateOf. REP remains a declared ANSI limit; saved-wrap replay was repaired later.

## Masks and omissions at those heads

| Helper behavior | What future output could expose |
|---|---|
|Clamp origin cursor and clear its pending wrap |Saved physical positions outside margins. |
|Clear active DECSC pending; compare only 1049 X/Y; omit inactive DECSC |Saved pen/origin/charset/wrap after restore or screen switch. |
|Zero wide-tail styles/rare attributes |Narrowing after a current-pen tail exposes its erase colors. |
|Conflate zero/literal space/spacer head |Public Cell, history joining and reflow. |
|Omit REP, sync and grapheme from stateOf |Modes covered the latter two only through assertContinuationState; TestRepaintRoundTrip used state directly. |
|Decode private history tags through ordinary Write, trim trailing spaces/zeros/spacers/protection and ignore backgrounds |ScrollbackCells and later Resize bringing history onto screen. |

Resolved pool-ID comparison is valid because collection renumbers IDs. Dead invalid saved fields cannot restore state. The then-current exact Snapshot helper compared every cursor flag and all three valid save slots. Different VT encodings need semantic history comparison; portable ANSI limits need named expected differences, not blanket normalization.

## Storage, decode and color findings

Cells 16 B/Terminal≤560 B; rare pairs reused 16-bit IDs, replacing links. A pair costs 24 B versus 16 B/string on 64-bit hosts; ordinary terminals allocate no table. The 65535 live-pair cap tests older-pair reuse/capacity recovery. Walkers covered both screens/current pen/two DECSC saves/the then-separate 1049 save; resize guarded detached imports. Decoder budgeted records/strings, validated IDs before install, converted legacy links to default-underline pairs, checked legacy cluster bits at UTF-8 boundaries before applying codepoint masks and retained legacy charset bits. `f98890e` already fixed Compact's private-mode predicate.

Keep X11 high-bit `#RGB`: `#3a7` means `#3000a0007000` ([Xlib specification](https://xorg.freedesktop.org/archive/current/doc/libX11/libX11/libX11.html)); native scales instead. For 12/16-bit `rgb:`, Phosphor rounds/native truncates; scaling does not settle final 8-bit rounding. Phosphor's exponent/clipping `rgbi:` grammar follows [kitty](https://sw.kovidgoyal.net/kitty/color-stack/#color-value-encoding), native accepts less. Native DECRQSS omits underline color; Phosphor includes it. Matching native alone would not justify breaking X11 expectations.
