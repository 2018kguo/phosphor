These six binary snapshots and expected public observations were generated
using phosphor v0.3.0 (3a732c18eee88e55977b233eb25b905633a669f7), before
the snapshot v2 writer was implemented. Each `.suffix` file is the raw input
written after saving its `.bin`; the `.json` records the released version's
cells, history text, cursor, modes, palette, title and PTY replies afterward.

They cover both screens, saved pens, old hyperlink IDs, mixed grapheme join
modes, saved mouse modes, palette changes and unfinished CSI, OSC, DCS and
UTF-8 input. Do not regenerate them with the current writer.
