# Independent runtime review

Reviewed phosphor 49f49b8..c8ea2cd (HEAD 40a5881 only adds executable bit to comparison driver), and ptyd af36c18..working tree on 231304c. Read-only; no repository edits.

## P1: idle pool collection exposes dangling IDs when resizing history back onto the screen

New trigger: compact.go:75 and compact.go:78, and clearing x at compact.go:82. Unsafe import is resize.go:407 and resize.go:410.

`Compact` collects dead screen lookup entries and resets capacity. On a later height increase, `parseLine` interns serialized history into detached rows. Filling the new small pool triggers its normal compactor, whose live walkers cannot see the imported rows yet. It deletes entries that those rows already reference; the returned screen has aliased or out-of-range IDs. `Repaint` then panics, and public Cell/Line access can misread text/links. This affects both cluster and hyperlink tables. ptyd now calls Compact at idle, making resize/attach after idle a daemon-crash path.

Public-API counterexamples, run with `go run` from the phosphor checkout:

- `/private/tmp/phosphor-round2.7e7BWb/review-history-pools.go`
- `/private/tmp/phosphor-round2.7e7BWb/review-history-links.go`

Both populate 400 distinct live pooled entries across a 100x4 screen, erase rows 2-4, scroll row 1 into serialized history, then leave a plain `$ ` prompt. They run paired paths with the same terminal input and resize to 100x5. No-Compact succeeds. Compact + Resize + Repaint panics (clusters: index 36, length 36; links: index 4, length 4). This uses actual public output and no state normalizer. Smaller all-new history imports can expose the underlying pre-existing importer weakness even without Compact; the paired witnesses specifically show a new regression caused by collection.

## Other lenses

No further confirmed introduced defect found in CSI22 semantics, bitset indexes, saved hyperlink/REP references, or ptyd lock order. `consume` releases t.mu before taking t.scrMu, so settleFired's t.scrMu -> t.mu nesting has no reverse overlapping edge in the inspected production call paths. ckptMu/timerMu are released or acquired outside that nesting. Stop/resize/attach/checkpoint paths retain their existing screen serialization.

The origin-mode replay difference is explicitly disclosed and exercised by TestScrollClearOutsideOriginReplayLimit using public cursor/next-cell assertions. The generic state normalizer already clamped origin-mode cursors before this diff. Treat this as an accepted documented boundary rather than a newly discovered hidden test pass.

Focused existing tests passed: TestScrollClear*, new pool compaction tests (0.338s), and ptyd TestIdleCheckpointKeepsScreenAndContinuation with the local runtime modfile (0.206s). No heavy suite or benchmarks run during review.
