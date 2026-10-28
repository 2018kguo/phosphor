# State comparison review

Reviewed integrated heads `f98890e` through `4e91b48` on 2026-10-27.
This is a read-only audit of the comparison helpers and future behavior.
It does not turn translated-suite passes into a universal compatibility claim.

## Confirmed continuation failures

Saved pending wrap is lost by `appendSavedCursor`. It restores a DECSC slot
with CUP followed by ESC 7, which does not recreate the saved pending wrap.
The existing `stateOf` helper clears that flag before comparing it.

Public API reproduction, using two terminals with size 4 by 3:

```go
source.Write([]byte("abcd\x1b7\rX"))
restored.Write(source.Repaint(false))
source.Write([]byte("\x1b8Z"))
restored.Write([]byte("\x1b8Z"))
```

Observed source lines: `["Xbcd", "Z", ""]`, cursor `(1,1)`.
Observed restored lines: `["XbcZ", "", ""]`, cursor `(3,0,pending)`.
The history workstream owns the repair, including the independent 1049 slot.
Snapshot preserves this flag; the defect is in ANSI replay.

The REP register is absent from `stateOf`. Printing the screen does not
necessarily restore the last scalar remembered for a subsequent REP.
Public API reproduction at size 4 by 2:

```go
source.Write([]byte("Q\x1b[2J\x1b[H"))
restored.Write(source.Repaint(false))
source.Write([]byte("\x1b[3b"))
restored.Write([]byte("\x1b[3b"))
```

Observed source lines: `["QQQ", ""]`; restored: `["", ""]`.
Snapshot includes the REP register. ANSI replay needs either a concrete
repair or an explicit continuation limit with this regression retained.

## Comparison masks and omissions

`terminal_test.go:stateOf` currently makes these behavior-changing comparisons:

- It clamps an origin-mode cursor to the current margins and clears its
  pending wrap. Saved cursors can put a physical cursor outside those margins.
- It clears pending wrap in the active DECSC slot. This masks the failure above.
- It reports only X and Y for the 1049 save slot, omitting its pen, origin,
  character sets and pending wrap.
- It omits the inactive screen's DECSC slot entirely. Switching screens and
  restoring that slot can expose any difference later.
- It replaces every wide tail's complete style and rare attributes with zero.
  Widening a grapheme can give its tail the current pen while the head retains
  its original pen. A later narrowing can expose the tail's old erase colors.
- It treats zero cells, literal spaces and spacer heads alike. Public cells
  distinguish literal spaces; spacer heads affect reflow and history joining.
- It does not report `lastChar`, the REP register. `sync` and `grapheme` are
  also omitted there. `assertContinuationState` compares public Modes, but
  `TestRepaintRoundTrip` calls `state` directly.

`normLine` decodes history with ordinary `Write`, so it does not decode the
new private sparse-hole tags. It also trims trailing literal spaces, zeros,
spacer heads and protected blanks while ignoring their background. That can
hide differences observable through ScrollbackCells or after history is
brought back onto the screen by Resize. Compare resolved history cells and
wrap state, with any unavoidable ANSI differences isolated explicitly.

Pool-ID normalization is legitimate: compaction can renumber IDs, and the
helpers compare the resolved hyperlink/underline pair instead. The exact
Snapshot helper also compares every cursor flag and all three valid saved
slots. Dead fields in an invalid saved slot cannot restore state and need
not be compared. Equivalent VT encodings likewise need semantic comparison,
not byte-for-byte history equality across independent repaint encoders.

Do not remove every mask blindly. Exact cell and saved-slot comparisons
should follow their corresponding replay fixes; portable VT limits need
named fixtures and expected differences instead of hidden normalization.

## Storage and decoder review

Cells remain 16 bytes and Terminal remains at most 560 bytes. Rare metadata
uses the existing 16-bit reference, replacing the old hyperlink table.
Each rare pair takes 24 bytes on a 64-bit host instead of a 16-byte string;
ordinary terminals still allocate no rare table. The 65535 live-pair bound
is documented and tested, including older-pair reuse and capacity recovery.

The reference walker covers both screens, the current pen, both valid DECSC
slots and the valid 1049 slot. Resize retains the existing collection guard
while imported rows are detached. Snapshot bounds the allocated metadata
records and their strings, checks IDs before use and installs state only
after validation succeeds. V1 links decode as pairs with default underline
color. V1 cluster mode bits are checked at UTF-8 boundaries before conversion
to the V2 codepoint mask; V1 charset flags retain their old one-bit layout.

An earlier Compact predicate omitted new private-mode bits. Head `f98890e`
already includes `x.privateModes == 0`; the reviewed head retains those modes.

## Color differences verified against primary sources

Keep `#RGB` high-bit behavior: Xlib explicitly specifies that `#3a7` means
`#3000a0007000`. Ghostty scales those digits instead. Changing phosphor to
match would break existing X11/xterm expectations. See the
[Xlib RGB Device String Specification](https://xorg.freedesktop.org/archive/current/doc/libX11/libX11/libX11.html)
and [kitty's color encoding](https://sw.kovidgoyal.net/kitty/color-stack/#color-value-encoding).

For 12/16-bit `rgb:` values, phosphor rounds and Ghostty truncates to 8 bits.
The protocol describes scaling but does not settle this final rounding
choice; native agreement alone is insufficient reason to change it.
Phosphor's `rgbi:` exponent handling and range clipping follow kitty's
documented grammar; the pinned native parser accepts a narrower subset.
Ghostty's DECRQSS SGR response omits underline color; phosphor reports it.
