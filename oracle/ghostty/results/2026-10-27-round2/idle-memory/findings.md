# Idle lookup-pool experiment

Base: 49f49b8. Each measurement uses a fresh process holding 128 terminals; shared input is allocated before the baseline, one terminal warms the implementation, and two GCs bracket HeapAlloc deltas. Values are medians of three process runs in bytes per terminal. This measures retained Go heap, not whole-process RSS.

| Trace | Base | Always rebuild | Mark before rebuild |
| --- | ---: | ---: | ---: |
| links | 151,216.125 | 2,497.000 | 2,497.125 |
| unicode | 5,432.250 | 2,496.125 | 2,496.125 |
| mixed-live | 5,617.125 | 2,785.000 | 2,785.000 |
| closed-1049 | 154,185.250 | 5,072.250 | 2,496.250 |

Traces: links = 64 unique 2 KiB URLs then erase/prompt; unicode = 64 unique combining clusters then erase/prompt; mixed-live = both churn types followed by one live saved URL/cluster; closed-1049 = both churn types, 1049 enter/exit while the last long URL is active, then close URL and erase/prompt. Opening a new URL in mixed-live already triggers the base active link collector; its reduction is predominantly unused cluster storage.

Always-rebuild negative control fails TestCompactDoesNotRebuildLivePools: two allocations on every unchanged live Compact. Selected implementation keeps pointers stable and has zero allocations for unchanged small live tables (up to 256 total entries). Large live tables allocate a transient mark set; no Terminal fields or Write hooks added.

Live roots use the same visitors as active pool collection: all cells on both screens, current pen, valid saved pens, and REP lastChar. Scrollback rows are already serialized VT and hold no indexes. All-invalid saved blocks are explicitly released; invalid pens no longer retain stale URL indexes.

Verification: focused Compact/saved-cursor/hyperlink/soft-reset tests passed, go vet ./... passed, diff check passed. FuzzCompactTrace passed for 30s with two workers: 37,493 executions. Full suite/race deferred to integrated parent verification by request. Tests compare every public cell on both screens, exact repaint, damage, saved pens, REP, kept alt, history/grow resize, custom tabs/saved modes, and split OSC/CSI/DCS/UTF-8 future input.
