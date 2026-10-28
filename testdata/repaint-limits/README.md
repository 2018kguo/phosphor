These raw Go fuzz inputs reproduce portable ANSI export losses. They live
outside `testdata/fuzz` because exact repaint equality is impossible for the
covered shapes. `TestRepaintHeightGrowthLimit`, `TestRepaintWidthGrowthLimit`,
and `TestNativeRepaintRectangleGrowthLimit` retain explicit checks of the loss.

With mode 47 and 1049 set, 1047 reset, and MAIN active, leaving ALT requires 1047
reset and erases its unprotected cells with one background. Host growth can
leave a uniformly erased top-left rectangle plus default zero padding that
this final transition cannot retain. Width-only cases at 80 or132 columns stay
outside this exception because DECCOLM can stage those dimensions. At the
pinned Ghostty source, `Terminal.zig:4069` restricts DECCOLM to 80/132 columns;
`Terminal.zig:5015` erases ALT on 1047 reset.

The repaint fuzz oracle recognizes only this erased rectangle shape. It still
compares the complete repaint result against an explicit export expectation:
only the zero padding's background changes to the erased rectangle background.
MAIN and cursor comparison retain the existing `stateOf` portable ANSI
contract (including its documented blank-cell and origin limitations); this
exception adds no new comparison masks. ALT public cells, row flags, modes,
and saved cursors retain additional exact checks. It separately requires exact Snapshot restoration and continued writes/resizes
against the original state. All other states retain strict repaint comparisons.
`FuzzSnapshotContinuation` is the universal exact checkpoint oracle.

Raw native parity cases remain failures rather than being normalized:
`cursor-1047-resize-background-limit` and
`cursor-1047-width-growth-background-limit` in `oracle/ghostty/traces.json`.
