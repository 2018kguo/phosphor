# Prompt metadata and native resize continuation

The native source is pinned to `46e9dc9a927b2e7e7cba79e6902c176127dbdf2c`.
These are the unmodified outputs of the native public-ABI driver against
`traces.json`, with whole writes, one-byte writes and seven-byte writes.

Each run has 194 named cases and 703 parity actions. Direct untouched-engine
comparison passes 703/703, and Phosphor's snapshot/repaint/Compact
continuation passes 703/703. Ghostty's own continuation passes 618/703;
all 85 failing native exporter actions remain in these files. Two identity
actions and one deliberately different host-default action keep their
separate scopes and raw answers. These finite counts are not a general
terminal compatibility percentage.

The new cases capture raw primary/continuation row annotations and future
effects of semantic cursor state, redraw policy, history metadata, erase,
screen switches, DECSTR pending wrap and live cursor mapping on resize.
Height growth while the alternate screen is active distinguishes the actual
inactive primary cursor from DECSC and 1049 saves. A non-bottom primary
cursor must not pull history, regardless of the alternate cursor position.

The Go suite and native driver tests pass. Bounded fuzzing also passed
155,143 snapshot continuation executions and 87,759 sparse-history grammar
executions. Terminal remains 560 bytes and each cell remains 16 bytes on
64-bit platforms. ASCII writes and input-only semantic state allocate zero
times after warmup. Prompt cursor bytes occupy existing padding; annotated
rows share the canonical wrap metadata byte.
