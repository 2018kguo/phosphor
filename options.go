package phosphor

import "io"

// An Option configures a Terminal in New.
type Option func(*config)

// config holds the options that are not plain Terminal fields, and the
// colors the program changed. A Terminal has one only if it needs it.
type config struct {
	sbLines, sbBytes int
	reply            io.Writer
	bell             func()
	palette          *Palette // the base palette, nil for DefaultPalette
	colors           []colorSet
}

// colorSet is a palette entry the program changed: 0-255 are the indexed
// colors, then the foreground, background and cursor.
type colorSet struct {
	i uint16
	c RGB
}

const (
	slotFg     = 256
	slotBg     = 257
	slotCursor = 258
)

// WithScrollback keeps up to lines lines of scrollback (a line that soft
// wrapped on screen counts once). Zero or less disables the scrollback.
// The default is DefaultScrollbackLines.
func WithScrollback(lines int) Option {
	return func(c *config) { c.sbLines = lines }
}

// WithScrollbackBytes bounds the memory the scrollback holds to about n
// bytes (lines are stored serialized, styles included), dropping the
// oldest lines first; the newest line is always kept. Zero or less means
// no byte limit. The default is DefaultScrollbackBytes.
func WithScrollbackBytes(n int) Option {
	return func(c *config) { c.sbBytes = n }
}

// WithReply makes the terminal answer the queries programs send, by
// writing the replies to w: usually the pseudo-terminal's input side, so
// they reach the program. Without it queries are ignored, as when a real
// terminal is attached to answer them instead.
//
// Answered: primary, secondary and tertiary device attributes (DA1, DA2,
// DA3), device status and cursor position reports (DSR 5 and 6, DECXCPR,
// and DSR ?15, ?26 and ?996), mode reports (DECRQM, ANSI and DEC), DECRQSS
// for SGR, DECSTBM, DECSLRM, DECSCUSR, DECSCA and DECSCL, XTVERSION,
// XTGETTCAP for TN, Co and colors, the kitty keyboard flags (CSI ? u), the
// text area size (XTWINOPS 18), and color queries (OSC 4, 10, 11, 12,
// answered from the palette). Errors from w are ignored. w is called
// during Write, with a slice it must not keep.
func WithReply(w io.Writer) Option {
	return func(c *config) { c.reply = w }
}

// WithBell calls fn when the program rings the bell (BEL), during Write.
func WithBell(fn func()) Option {
	return func(c *config) { c.bell = fn }
}

// WithPalette sets the colors the terminal starts with: those it reports
// to programs that ask (see WithReply), and that Terminal.Palette returns.
// The default is DefaultPalette.
func WithPalette(p Palette) Option {
	return func(c *config) { c.palette = &p }
}
