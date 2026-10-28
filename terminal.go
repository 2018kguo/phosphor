package phosphor

import (
	"bytes"
	"encoding/binary"
	"strings"
	"unicode/utf8"
	"unsafe"
)

// cell is one screen position. R is 0 for a blank, wideTail for the right
// half of a double-width character, spacerHead for the blank a wide
// character left at the end of a row it wrapped from, and clusterBase+i
// for a character with combining marks (the Terminal's clusters[i]).
type cell struct {
	R  rune
	Fg Color
	Bg Color
	A  Attr
	E  uint16 // rare attributes: 0 for default, else the Terminal's rare[E-1]
}

const (
	wideTail rune = -1
	// spacerHead is the blank left at the end of a row when a wide
	// character did not fit there and wrapped: not part of the text.
	spacerHead  rune = -2
	clusterBase rune = 0x40000000
)

// The scrollback budget a Terminal has unless WithScrollback or
// WithScrollbackBytes set another.
const (
	DefaultScrollbackLines = 2000
	DefaultScrollbackBytes = 256 << 10
	maxOSC                 = 4096
	maxParams              = 32
	maxClusterRunes        = 65 // base plus 64 suffix codepoints, matching libghostty-vt
	maxCluster             = maxClusterRunes * utf8.UTFMax
	maxLogicalLine         = 32 << 10
)

type charset uint8

const (
	charsetASCII charset = iota
	charsetGraphics
	charsetBritish
)

type cursor struct {
	x, y        int
	pen         cell
	wrapPending bool
	origin      bool
	g           [4]charset // G0-G3 designated character sets
	gl          uint8      // which of G0-G3 is invoked into GL
	ss          uint8      // pending single shift (2 or 3)
}

type savedCursor struct {
	cursor
	valid bool
}

// savedCursors is allocated on the first save and reused until a full
// reset or Compact finds every slot invalid. A shell prompt pays for none
// of the two slots; repeated active saves and screen switches reuse it.
type savedCursors struct {
	screen [2]savedCursor // DECSC slots: main, alternate
}

// Terminal is a VT/xterm screen model: write a program's output to it and
// read back what a terminal would show. Create one with New.
//
// A Terminal is not safe for concurrent use: callers that write from one
// goroutine and read from another must hold a lock around both. Writers
// and callbacks given as options run synchronously inside Write and must
// not call back into the Terminal.
type Terminal struct {
	cols, rows int
	main, alt  [][]cell // alt is allocated on first entry and retained for later switches
	// Canonical row metadata, nil until a row wraps or a shell annotates it.
	// rowWrap means row y continues on row y+1.
	mainWrap, altWrap []rowFlags
	altActive         bool
	cur               cursor
	saved             *savedCursors
	top, bot          int  // scroll region, inclusive
	left, right       int  // left and right margins, inclusive
	lrmm              bool // DECLRMM: left and right margins enabled
	protISO           bool // protection was last set by SPA (ISO), not DECSCA
	grapheme          bool // 2027: Unicode grapheme clustering
	// These bytes occupy existing padding before x; Terminal stays560 bytes.
	semantic     [2]semanticContent // screen content state, independent of DECSC
	promptRedraw promptRedraw
	x            *extra    // state few programs use, nil until one does
	cfg          *config   // options and palette changes, nil if none
	dirty        *dirtySet // damage, nil until ClearDirty is first called

	// modes
	autowrap     bool
	revWrap      bool // 45: BS and CUB continue on the previous wrapped line
	revWrapExt   bool // 1045: ... on any previous line, and top to bottom
	insert       bool
	newline      bool // LNM
	cursorHidden bool
	reverseVideo bool // DECSCNM
	appCursor    bool
	appKeypad    bool
	bracketed    bool
	focus        bool
	sync         bool   // 2026: synchronized update in progress
	mouse        uint16 // independent tracking bits and effective selection; see mouse_modes.go
	mouseEnc     uint16 // independent encoding bits and effective selection
	cursorStyle  cursorAppearance
	kitty        []uint16
	title        string
	lastChar     rune

	// scrollback, oldest first; each entry is a serialized logical line
	// (rows joined where they soft-wrapped)
	sb               [][]byte
	sbStart, sbBytes int
	sbMaxLines       int
	sbMaxBytes       int
	sbOpen           bool // the last line continues on screen row 0
	sbSpacer         bool // ... after a spacer head left out of it
	scratch          []byte

	// parser
	params   [maxParams]uint32 // saturated at 65536, including on 64-bit hosts
	osc      []byte
	sub      uint32 // bit i marks a ':' before params[i]
	utf8buf  [utf8.UTFMax]byte
	nparams  uint8 // at most maxParams; also the partial CSI's current slot
	state    uint8
	utf8n    uint8 // an unfinished UTF-8 scalar has at most four bytes
	paramSet bool  // the current CSI parameter has digits
	marker   byte  // private marker: ? > < =
	inter    byte  // last intermediate byte
	oscBel   bool  // the OSC string was ended by BEL (replies end the same way)
	oscOver  bool
}

// extra holds the state few programs use, kept out of Terminal so an idle
// terminal stays small.
type extra struct {
	tabs         []uint64    // tab stops, nil for every 8 columns
	savedModes   []savedMode // XTSAVE
	privateModes uint8       // column, scroll and independent alternate-screen mode bits
	// The primary screen's actual cursor while the alternate is active.
	// A DECSC slot can name a different position and cannot replace this fact.
	inactivePrimary point
	// characters with combining marks, referenced by cells
	clusters   []cluster
	clusterCap int
	// Uncommon cell attributes share the cell's 16-bit index.
	rare    []cellExtra
	rareCap int
}

type cellExtra struct {
	link      string // OSC 8 target ("params;uri")
	underline Color
}

// ext returns t.x, allocating it.
func (t *Terminal) ext() *extra {
	if t.x == nil {
		t.x = &extra{}
	}
	return t.x
}

type savedMode struct {
	mode int
	on   bool
}

// Parser states. The string states (stOSC and after) come last: C0
// controls act in every state before them.
const (
	stGround uint8 = iota
	stEsc
	stEscInter
	stCSI
	stCSIIgnore
	stOSC
	stStr // SOS, PM, APC, and DCS strings past their start: ignored until ST
	stDCS // DCS: collected (into osc) for the queries answered
)

// New returns a blank terminal of cols x rows (each at least 1),
// configured by opts.
func New(cols, rows int, opts ...Option) *Terminal {
	t := newTerminal(cols, rows, DefaultScrollbackLines, DefaultScrollbackBytes)
	if len(opts) > 0 {
		c := &config{sbLines: DefaultScrollbackLines, sbBytes: DefaultScrollbackBytes}
		for _, o := range opts {
			o(c)
		}
		t.sbMaxLines, t.sbMaxBytes = max(c.sbLines, 0), c.sbBytes
		if t.sbMaxBytes <= 0 {
			t.sbMaxBytes = int(^uint(0) >> 1)
		}
		if c.reply != nil || c.bell != nil || c.palette != nil {
			t.cfg = c
		}
	}
	return t
}

func newTerminal(cols, rows, sbLines, sbBytes int) *Terminal {
	cols, rows = max(cols, 1), max(rows, 1)
	return &Terminal{
		cols: cols, rows: rows,
		main:       make([][]cell, rows),
		bot:        rows - 1,
		right:      cols - 1,
		autowrap:   true,
		sbMaxLines: max(sbLines, 0),
		sbMaxBytes: sbBytes,
	}
}

func (t *Terminal) screen() [][]cell {
	if t.altActive {
		return t.alt
	}
	return t.main
}

// wraps returns the active screen's soft-wrap flags (nil if none is set).
func (t *Terminal) wraps() []rowFlags {
	if t.altActive {
		return t.altWrap
	}
	return t.mainWrap
}

// setWrap sets whether row y continues on row y+1.
func (t *Terminal) setWrap(y int, on bool) {
	w := t.wraps()
	if w == nil {
		if !on {
			return
		}
		w = make([]rowFlags, t.rows)
		if t.altActive {
			t.altWrap = w
		} else {
			t.mainWrap = w
		}
	}
	if on {
		w[y] |= rowWrap
	} else {
		w[y] &^= rowWrap
	}
}

// wrapped reports whether row y continues the row above it (for row 0 of
// the main screen, the last scrollback line).
func (t *Terminal) wrapped(y int) bool {
	if y == 0 {
		return !t.altActive && t.sbOpen
	}
	w := t.wraps()
	return w != nil && w[y-1]&rowWrap != 0
}

// Write feeds the terminal a program's output, implementing io.Writer.
// Escape sequences and UTF-8 characters may be split across calls. It
// consumes all of p and never fails; it does not keep p. Replies to
// queries (see WithReply) and the bell callback run before it returns.
func (t *Terminal) Write(p []byte) (int, error) {
	for i := 0; i < len(p); i++ {
		c := p[i]
		if t.state == stCSI && c >= '0' && c <= '9' {
			// Parameters are lexical digit runs. Accumulate each run once
			// instead of dispatching every digit through the state machine.
			j := i
			v := uint32(0)
			if t.nparams < maxParams {
				v = t.params[t.nparams]
			}
			for j < len(p) && p[j] >= '0' && p[j] <= '9' {
				v = csiDigit(v, p[j])
				j++
			}
			if t.nparams < maxParams {
				t.params[t.nparams] = v
			}
			t.paramSet = true
			i = j - 1
			continue
		}
		if t.state == stGround {
			if t.utf8n > 0 {
				t.utf8Continue(c)
				continue
			}
			if c >= 0x20 && c < 0x7f {
				// Fast path: a run of printable ASCII, scanned eight
				// bytes at a time.
				j := i + 1
				for j+8 <= len(p) && printable8(binary.LittleEndian.Uint64(p[j:])) {
					j += 8
				}
				for j < len(p) && p[j] >= 0x20 && p[j] < 0x7f {
					j++
				}
				t.printASCII(p[i:j])
				i = j - 1
				continue
			}
			if c >= 0x80 {
				i += t.printUTF8(p[i:]) - 1
				continue
			}
			if c == 0x1b && i+1 < len(p) && p[i+1] == '[' {
				// Dispatch complete short commands without re-entering the
				// parser for each byte. Longer or split commands still enter
				// the same parameter state as the byte-wise introducer.
				t.nparams, t.paramSet, t.marker, t.inter = 0, false, 0, 0
				t.params[0], t.sub = 0, 0
				if n, value := shortCSI(p[i+2:]); n != 0 {
					if n > 1 {
						t.params[0], t.nparams, t.paramSet = value, 1, true
					}
					final := p[i+n+1]
					if final == 'm' {
						t.sgr()
					} else {
						t.csiDispatch(final)
					}
					i += n + 1
					continue
				}
				t.state = stCSI
				i++
				continue
			}
		}
		t.step(c)
	}
	return len(p), nil
}

// shortCSI recognizes a complete standard final with at most two decimal
// digits. Private markers, intermediates, longer parameters and split input
// keep using the canonical parser.
func shortCSI(p []byte) (n int, value uint32) {
	for n < len(p) && n < 2 && p[n] >= '0' && p[n] <= '9' {
		value = value*10 + uint32(p[n]-'0')
		n++
	}
	if n < len(p) && p[n] >= 0x40 && p[n] <= 0x7e {
		return n + 1, value
	}
	return 0, 0
}

// printable8 reports whether all eight bytes of w are printable ASCII
// (0x20-0x7e): subtracting 0x20 borrows into the top bit of a byte under
// 0x20, adding 1 carries into it for 0x7f, and bytes from 0x80 have it
// set already.
func printable8(w uint64) bool {
	const lo, one, top = 0x2020202020202020, 0x0101010101010101, 0x8080808080808080
	return ((w-lo)|(w+one)|w)&top == 0
}

// printUTF8 prints the run of non-ASCII characters at the start of p
// (p[0] >= 0x80) and returns how many bytes it consumed. Complete, valid
// sequences are decoded at once; anything else (invalid, or cut at the
// end of p) goes through the byte-at-a-time decoder, which ends the run.
func (t *Terminal) printUTF8(p []byte) int {
	i := 0
	for i < len(p) && p[i] >= 0x80 {
		r, size := utf8.DecodeRune(p[i:])
		if size == 1 {
			t.utf8Start(p[i])
			return i + 1
		}
		i += size
		if r < 0xa0 {
			// A C1 control (U+0080-U+009F): it may start a sequence, so
			// the rest goes through the state machine.
			t.c1(r)
			return i
		}
		t.printRune(r)
	}
	return i
}

func (t *Terminal) utf8Start(c byte) {
	t.utf8buf[0] = c
	t.utf8n = 1
	if c < 0xc2 || c > 0xf4 { // stray continuation or invalid lead
		t.utf8n = 0
		t.printRune(utf8.RuneError)
	}
}

func (t *Terminal) utf8Continue(c byte) {
	if c&0xc0 != 0x80 {
		// Truncated sequence: emit a replacement, then process c afresh.
		t.utf8n = 0
		t.printRune(utf8.RuneError)
		t.Write([]byte{c})
		return
	}
	t.utf8buf[t.utf8n] = c
	t.utf8n++
	if utf8.FullRune(t.utf8buf[:t.utf8n]) {
		r, size := utf8.DecodeRune(t.utf8buf[:t.utf8n])
		t.utf8n = 0
		if size == 1 {
			// Invalid (overlong, surrogate, out of range).
			t.printRune(utf8.RuneError)
			return
		}
		if r < 0xa0 {
			t.c1(r)
			return
		}
		t.printRune(r)
	}
}

// c1 runs a C1 control received as a UTF-8 code point: like xterm, it is
// the 7-bit ESC Fe equivalent.
func (t *Terminal) c1(r rune) {
	t.step(0x1b)
	t.step(byte(r - 0x40))
}

func (t *Terminal) printRune(r rune) {
	// A single shift applies to the next character, including a Unicode
	// combining mark. National character sets only translate GL bytes.
	if r > 0x7f {
		t.cur.ss = 0
	}
	if t.grapheme && r > 255 && t.extendGrapheme(r) {
		return
	}
	w := CodepointWidth(r)
	if w == 0 {
		if !t.grapheme {
			t.combine(r)
		}
		return
	}
	t.print(r, w)
}

// step runs one byte through the escape-sequence state machine.
func (t *Terminal) step(c byte) {
	// C0 controls act in every state but strings. Only control bytes can
	// end a string or abort a sequence; printable CSI bytes skip this work.
	if c < 0x20 {
		if c == 0x1b || c == 0x18 || c == 0x1a {
			state := t.state
			t.state = stGround
			if c == 0x1b {
				t.state = stEsc
			}
			if state == stDCS {
				// DCS handlers finish when unhooked, including cancellation.
				t.dcsDispatch()
			} else if state == stOSC && c == 0x1b {
				t.oscBel = false
				t.oscDispatch()
			}
			return
		}
		if t.state < stOSC {
			t.execute(c)
			return
		}
	}
	switch t.state {
	case stGround:
		// 0x7f (DEL) is ignored.
	case stEsc:
		t.inter = 0
		switch {
		case c == '[':
			t.nparams, t.paramSet, t.marker, t.inter = 0, false, 0, 0
			t.params[0], t.sub = 0, 0
			t.state = stCSI
		case c == ']':
			t.osc, t.oscOver = t.osc[:0], false
			t.state = stOSC
		case c == 'P':
			t.osc = t.osc[:0]
			t.state = stDCS
		case c == 'X' || c == '^' || c == '_':
			t.state = stStr
		case c >= 0x20 && c <= 0x2f:
			t.inter = c
			t.state = stEscInter
		case c >= 0x30 && c <= 0x7e:
			t.state = stGround
			t.escDispatch(c)
		default:
			t.state = stGround
		}
	case stEscInter:
		switch {
		case c >= 0x20 && c <= 0x2f:
			t.inter = c
		case c >= 0x30 && c <= 0x7e:
			t.state = stGround
			t.escDispatch(c)
		default:
			t.state = stGround
		}
	case stCSI:
		switch {
		case c >= '0' && c <= '9':
			if t.nparams < maxParams {
				t.params[t.nparams] = csiDigit(t.params[t.nparams], c)
			}
			t.paramSet = true
		case c == ';' || c == ':':
			if t.nparams < maxParams-1 {
				t.nparams++
				t.params[t.nparams] = 0
				if c == ':' {
					t.sub |= 1 << uint(t.nparams)
				} else {
					t.sub &^= 1 << uint(t.nparams)
				}
			} else {
				t.nparams = maxParams
			}
			t.paramSet = false
		case c >= '<' && c <= '?':
			if t.paramSet || t.nparams != 0 || t.marker != 0 {
				t.state = stCSIIgnore
			} else {
				t.marker = c
			}
		case c >= 0x20 && c <= 0x2f:
			t.inter = c
		case c >= 0x40 && c <= 0x7e:
			t.state = stGround
			missingColon := !t.paramSet && t.nparams < maxParams && t.subParam(int(t.nparams))
			if t.paramSet || t.nparams != 0 {
				t.nparams = min(t.nparams+1, maxParams)
			}
			if missingColon && c == 'm' && t.marker == 0 && t.inter == 0 {
				// A final colon has no implicit zero sub-parameter in SGR.
				t.nparams--
			}
			t.csiDispatch(c)
		default:
			t.state = stCSIIgnore
		}
	case stCSIIgnore:
		if c >= 0x40 && c <= 0x7e {
			t.state = stGround
		}
	case stOSC:
		switch c {
		case 0x07:
			t.state = stGround
			t.oscBel = true
			t.oscDispatch()
		default:
			if len(t.osc) < maxOSC {
				t.osc = append(t.osc, c)
			} else {
				t.oscOver = true
			}
		}
	case stStr:
	case stDCS:
		t.osc = append(t.osc, c)
		// Only the queries answered are kept (DECRQSS, XTGETTCAP); the
		// rest (sixel, tmux passthrough...) is skipped.
		if len(t.osc) == 2 && !isDCSQuery(t.osc) || len(t.osc) > 256 {
			t.state = stStr
		}
	}
}

func csiDigit(value uint32, digit byte) uint32 {
	return min(value*10+uint32(digit-'0'), 1<<16)
}

func (t *Terminal) execute(c byte) {
	switch c {
	case '\b':
		t.cursorLeft(1)
	case '\t':
		t.cur.x = t.nextTab(t.cur.x)
	case '\n', '\v', '\f':
		t.cur.wrapPending = false
		t.index()
		if t.newline {
			t.carriageReturn()
		}
	case '\r':
		t.carriageReturn()
	case 0x07:
		if t.cfg != nil && t.cfg.bell != nil {
			t.cfg.bell()
		}
	case 0x0e:
		t.cur.gl = 1
	case 0x0f:
		t.cur.gl = 0
	}
}

func (t *Terminal) escDispatch(c byte) {
	switch t.inter {
	case 0:
	case '(', ')', '*', '+':
		g := charsetASCII
		if c == '0' {
			g = charsetGraphics
		} else if c == 'A' {
			g = charsetBritish
		}
		t.cur.g[t.inter-'('] = g
		return
	case '#':
		if c == '8' {
			t.decaln()
		}
		return
	default:
		return
	}
	switch c {
	case '7':
		t.saveCursor()
	case '8':
		t.restoreCursor()
	case 'D':
		t.cur.wrapPending = false
		t.index()
	case 'E':
		t.cur.wrapPending = false
		t.index()
		t.carriageReturn()
	case 'H':
		t.setTab(t.cur.x, true)
	case 'M':
		t.cur.wrapPending = false
		t.reverseIndex()
	case 'V': // SPA
		t.protect(true, true)
	case 'W': // EPA
		t.protect(false, true)
	case 'N':
		t.cur.ss = 2
	case 'O':
		t.cur.ss = 3
	case 'n':
		t.cur.gl = 2
	case 'o':
		t.cur.gl = 3
	case 'c':
		t.reset()
	case '=':
		t.appKeypad = true
	case '>':
		t.appKeypad = false
	}
}

// decaln fills the screen with E (the alignment test) and resets the
// margins.
func (t *Terminal) decaln() {
	t.touchAll()
	scr := t.screen()
	for y := range scr {
		row := t.row(y)
		for x := range row {
			row[x] = cell{R: 'E'}
		}
	}
	clear(t.wraps())
	t.top, t.bot, t.left, t.right = 0, t.rows-1, 0, t.cols-1
	t.cur.x, t.cur.y, t.cur.wrapPending = 0, 0, false
}

func (t *Terminal) reset() {
	sb, sbStart, sbBytes := t.sb, t.sbStart, t.sbBytes
	cfg, dirty := t.cfg, t.dirty
	if cfg != nil {
		cfg.colors = nil // RIS resets the palette
	}
	*t = Terminal{
		cfg: cfg, dirty: dirty,
		cols: t.cols, rows: t.rows,
		main:         make([][]cell, t.rows),
		bot:          t.rows - 1,
		right:        t.cols - 1,
		autowrap:     true,
		promptRedraw: promptRedrawAll, // native RIS differs from headless construction
		sb:           sb,
		sbStart:      sbStart,
		sbBytes:      sbBytes,
		sbMaxLines:   t.sbMaxLines,
		sbMaxBytes:   t.sbMaxBytes,
		osc:          t.osc[:0],
	}
	t.touchAll()
}

func (t *Terminal) saveCursor() {
	i := 0
	if t.altActive {
		i = 1
	}
	c := t.cur
	c.pen.E = t.internExtra(cellExtra{underline: t.extraAt(c.pen.E).underline})
	t.cursorSaves().screen[i] = savedCursor{cursor: c, valid: true}
}

func (t *Terminal) cursorSaves() *savedCursors {
	if t.saved == nil {
		t.saved = &savedCursors{}
	}
	return t.saved
}

func (t *Terminal) restoreCursor() {
	i := 0
	if t.altActive {
		i = 1
	}
	link := t.link(t.cur.pen.E)
	if t.saved == nil || !t.saved.screen[i].valid {
		t.cur = cursor{}
	} else {
		t.setCursor(t.saved.screen[i].cursor)
	}
	t.setLink(link)
}

// setCursor installs a saved cursor, clamping it to the current size.
// Pending wrap belongs to the saved cursor, even if margins or DECAWM
// changed since DECSC. The next print consumes it according to the mode.
func (t *Terminal) setCursor(c cursor) {
	t.cur = c
	t.cur.x = min(t.cur.x, t.cols-1)
	t.cur.y = min(t.cur.y, t.rows-1)
}

// param returns parameter i (ignoring sub-parameters), or def if it is
// absent or zero.
func (t *Terminal) param(i, def int) int {
	if i >= int(t.nparams) || t.params[i] == 0 {
		return def
	}
	return int(t.params[i])
}

// subParam reports whether parameter i is a colon-separated sub-parameter.
func (t *Terminal) subParam(i int) bool { return t.sub&(1<<uint(i)) != 0 }

func (t *Terminal) csiDispatch(c byte) {
	if t.cfg != nil && t.cfg.reply != nil && t.csiQuery(c) {
		return
	}
	switch t.marker {
	case '?':
		if t.inter != 0 {
			return
		}
		switch c {
		case 'h', 'l':
			for i := 0; i < int(t.nparams); i++ {
				if !t.subParam(i) {
					t.decMode(int(t.params[i]), c == 'h')
				}
			}
		case 's', 'r':
			for i := 0; i < int(t.nparams); i++ {
				if !t.subParam(i) {
					t.xtSaveMode(int(t.params[i]), c == 's')
				}
			}
		case 'J':
			t.eraseDisplay(t.param(0, 0), true)
		case 'K':
			t.eraseLine(t.param(0, 0), true)
		}
		return
	case '>':
		if c == 'u' && t.inter == 0 { // kitty keyboard: push flags
			if len(t.kitty) >= 16 {
				t.kitty = t.kitty[1:]
			}
			t.kitty = append(t.kitty, uint16(t.param(0, 0)))
		}
		return
	case '<':
		if c == 'u' && t.inter == 0 { // kitty keyboard: pop n
			n := min(t.param(0, 1), len(t.kitty))
			t.kitty = t.kitty[:len(t.kitty)-n]
			if len(t.kitty) == 0 {
				t.kitty = nil
			}
		}
		return
	case '=':
		if c == 'u' && t.inter == 0 { // kitty keyboard: set flags
			f := uint16(t.param(0, 0))
			if len(t.kitty) == 0 {
				t.kitty = append(t.kitty, 0)
			}
			top := &t.kitty[len(t.kitty)-1]
			switch t.param(1, 1) {
			case 1:
				*top = f
			case 2:
				*top |= f
			case 3:
				*top &^= f
			}
		}
		return
	}
	switch t.inter {
	case 0:
	case ' ':
		if c == 'q' {
			if p := t.param(0, 0); p <= 6 {
				t.cursorStyle = cursorAppearance{shape: CursorShape((p + 1) / 2), blink: p%2 == 1}
			}
		}
		return
	case '!':
		if c == 'p' {
			t.softReset()
		}
		return
	case '"':
		if c == 'q' { // DECSCA
			t.protect(t.param(0, 0) == 1, false)
		}
		return
	default:
		return
	}
	switch c {
	case '@':
		t.insertCells(t.param(0, 1))
	case 'A':
		t.cursorUp(t.param(0, 1))
	case 'B', 'e':
		t.cursorDown(t.param(0, 1))
	case 'C', 'a':
		edge := t.cols - 1
		if t.cur.x <= t.right {
			edge = t.right
		}
		t.setX(min(t.cur.x+t.param(0, 1), edge))
	case 'D', 'j':
		t.cursorLeft(t.param(0, 1))
	case 'E':
		t.cursorDown(t.param(0, 1))
		t.carriageReturn()
	case 'F':
		t.cursorUp(t.param(0, 1))
		t.carriageReturn()
	case 'G', '`':
		_, y := t.relPos()
		t.moveTo(t.param(0, 1)-1, y)
	case 'H', 'f':
		t.moveTo(t.param(1, 1)-1, t.param(0, 1)-1)
	case 'd':
		x, _ := t.relPos()
		t.moveTo(x, t.param(0, 1)-1)
	case 'k':
		t.cursorUp(t.param(0, 1))
	case 'I':
		for range min(t.param(0, 1), t.cols) {
			t.cur.x = t.nextTab(t.cur.x)
		}
	case 'Z':
		lim := 0
		if t.cur.origin {
			lim = t.left
		}
		for range min(t.param(0, 1), t.cols) {
			t.cur.x = max(t.prevTab(t.cur.x), min(lim, t.cur.x))
		}
	case 'g':
		switch t.param(0, 0) {
		case 0:
			t.setTab(t.cur.x, false)
		case 3:
			t.ext().tabs = make([]uint64, (t.cols+63)/64)
		}
	case 'J':
		t.eraseDisplay(t.param(0, 0), false)
	case 'K':
		t.eraseLine(t.param(0, 0), false)
	case 'L':
		t.insertLines(t.param(0, 1))
	case 'M':
		t.deleteLines(t.param(0, 1))
	case 'P':
		t.deleteCells(t.param(0, 1))
	case 'X':
		n := min(t.param(0, 1), t.cols-t.cur.x)
		t.cur.wrapPending = false
		t.erase(t.cur.y, t.cur.x, t.cur.x+n, t.protISO)
		t.setWrap(t.cur.y, false) // as ghostty: the row no longer wraps
	case 'S':
		t.scrollUp(t.top, t.bot, t.param(0, 1))
	case 'T':
		if t.nparams <= 1 {
			t.scrollDown(t.top, t.bot, t.param(0, 1))
		}
	case 'b':
		if t.lastChar != 0 {
			w := t.runeWidth(t.lastChar)
			for range min(t.param(0, 1), t.cols*t.rows) {
				t.print(t.lastChar, w)
			}
		}
	case 'm':
		t.sgr()
	case 'h', 'l':
		for i := 0; i < int(t.nparams); i++ {
			switch t.params[i] {
			case 4:
				t.insert = c == 'h'
			case 20:
				t.newline = c == 'h'
			}
		}
	case 'r':
		top, bot := t.param(0, 1)-1, t.param(1, t.rows)-1
		bot = min(bot, t.rows-1)
		if top < bot {
			t.top, t.bot = top, bot
			t.moveTo(0, 0)
		}
	case 's':
		if !t.lrmm {
			t.saveCursor()
			break
		}
		// DECSLRM
		left, right := t.param(0, 1)-1, min(t.param(1, t.cols), t.cols)-1
		if left < right {
			t.left, t.right = left, right
			t.moveTo(0, 0)
		}
	case 'u':
		t.restoreCursor()
	}
}

func (t *Terminal) softReset() {
	t.cursorHidden = false
	t.cursorStyle = cursorAppearance{}
	t.insert = false
	t.cur.origin = false
	t.autowrap = true
	t.revWrap, t.revWrapExt = false, false
	t.protISO = false
	t.appCursor = false
	t.appKeypad = false
	t.lrmm = false
	t.top, t.bot, t.left, t.right = 0, t.rows-1, 0, t.cols-1
	t.cur.pen = cell{}
	t.cur.g, t.cur.gl, t.cur.ss = [4]charset{}, 0, 0
	if t.saved != nil {
		slot := 0
		if t.altActive {
			slot = 1
		}
		t.saved.screen[slot] = savedCursor{}
	}
}

// modeValue reports the state of a DEC private mode XTSAVE can save, and
// whether it is one.
func (t *Terminal) modeValue(mode int) (on, ok bool) {
	switch mode {
	case 1:
		return t.appCursor, true
	case 3:
		return t.privateModes()&privateMode132 != 0, true
	case 4:
		return t.privateModes()&privateModeSlowScroll != 0, true
	case 47, 1047, 1049:
		return t.privateModes()&screenModeBit(mode) != 0, true
	case 40:
		return t.privateModes()&privateModeAllowColumns != 0, true
	case 5:
		return t.reverseVideo, true
	case 6:
		return t.cur.origin, true
	case 7:
		return t.autowrap, true
	case 12:
		return t.cursorStyle.blink, true
	case 45:
		return t.revWrap, true
	case 69:
		return t.lrmm, true
	case 1045:
		return t.revWrapExt, true
	case 25:
		return !t.cursorHidden, true
	case 9, 1000, 1002, 1003:
		return t.mouse&mouseModeBit(mode, mouseTrackingNumbers) != 0, true
	case 1005, 1006, 1015, 1016:
		return t.mouseEnc&mouseModeBit(mode, mouseEncodingNumbers) != 0, true
	case 1004:
		return t.focus, true
	case 2004:
		return t.bracketed, true
	case 2027:
		return t.grapheme, true
	}
	return false, false
}

// xtSaveMode saves (XTSAVE) or restores (XTRESTORE) a DEC private mode.
func (t *Terminal) xtSaveMode(mode int, save bool) {
	x := t.ext()
	for i, s := range x.savedModes {
		if s.mode != mode {
			continue
		}
		if save {
			x.savedModes[i].on, _ = t.modeValue(mode)
		} else {
			t.decMode(mode, s.on)
		}
		return
	}
	if on, ok := t.modeValue(mode); ok {
		if save && len(x.savedModes) < len(savedModeNumbers) {
			x.savedModes = append(x.savedModes, savedMode{mode, on})
		} else if !save {
			t.decMode(mode, mode == 7 || mode == 25)
		}
	}
}

// tabs returns the custom tab stops (nil: every 8 columns).
func (t *Terminal) tabs() []uint64 {
	if t.x == nil {
		return nil
	}
	return t.x.tabs
}

func (t *Terminal) decMode(mode int, on bool) {
	switch mode {
	case 1:
		t.appCursor = on
	case 3:
		t.setColumnMode(on)
	case 4:
		if on {
			t.ext().privateModes |= privateModeSlowScroll
		} else if t.x != nil {
			t.x.privateModes &^= privateModeSlowScroll
		}
	case 40:
		if on {
			t.ext().privateModes |= privateModeAllowColumns
		} else if t.x != nil {
			t.x.privateModes &^= privateModeAllowColumns
		}
	case 5:
		if t.reverseVideo != on {
			t.touchAll() // every cell's colors change
		}
		t.reverseVideo = on
	case 6:
		t.cur.origin = on
		t.moveTo(0, 0)
	case 7:
		t.autowrap = on
	case 12:
		t.cursorStyle.blink = on
	case 45:
		t.revWrap = on
	case 69:
		t.lrmm = on
		if !on {
			t.left, t.right = 0, t.cols-1
		}
	case 1045:
		t.revWrapExt = on
	case 25:
		t.cursorHidden = !on
	case 9, 1000, 1002, 1003:
		t.mouse = setMouseMode(t.mouse, mode, on, mouseTrackingNumbers)
	case 1005, 1006, 1015, 1016:
		t.mouseEnc = setMouseMode(t.mouseEnc, mode, on, mouseEncodingNumbers)
	case 1004:
		t.focus = on
	case 2004:
		t.bracketed = on
	case 2026:
		t.sync = on
	case 2027:
		t.grapheme = on
	case 47, 1047, 1049:
		t.setScreenMode(mode, on)
	case 1048:
		if on {
			t.saveCursor()
		} else {
			t.restoreCursor()
		}
	}
}

// setScreenMode separates each reported mode bit from the active screen.
func (t *Terminal) setScreenMode(mode int, on bool) {
	bit := screenModeBit(mode)
	if on {
		t.ext().privateModes |= bit
	} else if t.x != nil {
		t.x.privateModes &^= bit
	}
	if mode == 1049 && on {
		t.saveCursor()
	}
	if mode == 1047 && !on && t.altActive {
		t.eraseDisplay(2, false)
	}
	changed, pending := on != t.altActive, t.cur.wrapPending
	t.switchScreen(on, mode)
	if mode == 1049 {
		if on {
			t.eraseDisplay(2, false)
			if changed {
				t.cur.wrapPending = pending
			}
		} else {
			t.restoreCursor()
		}
	}
}

// switchScreen ends the live hyperlink only when the screen changes.
// Charset and ordinary pen state travel with the cursor; DECSC does not.
func (t *Terminal) switchScreen(on bool, mode int) {
	if on == t.altActive {
		return
	}
	t.setLink("")
	t.touchAll()
	content := t.semanticContent()
	if on {
		t.ext().inactivePrimary = point{t.cur.x, t.cur.y, t.cur.wrapPending}
	} else if t.x != nil {
		t.x.inactivePrimary = point{}
	}
	t.altActive = on
	if on || mode != 1049 {
		t.setSemanticContent(content)
	}
	if on && t.alt == nil {
		t.alt = make([][]cell, t.rows)
	}
}

func (t *Terminal) altBlank() bool {
	for _, row := range t.alt {
		if !zeroRow(row) {
			return false
		}
	}
	return true
}

func (t *Terminal) clearScreen() {
	t.eraseRows(0, t.rows)
}

// eraseRows erases rows [from, to), reusing rows already allocated.
func (t *Terminal) eraseRows(from, to int) {
	scr := t.screen()
	w := t.wraps()
	for y := from; y < to; y++ {
		t.touch(y)
		if row := scr[y]; row != nil {
			scr[y] = t.blank(row)
		} else {
			scr[y] = t.blankRow()
		}
		if w != nil {
			w[y] = 0
		}
	}
}

// blank erases row with the current background, retaining its allocation.
func (t *Terminal) blank(row []cell) []cell {
	if t.cur.pen.Bg == 0 {
		return row[:0]
	}
	row = growRow(row, t.cols, t.cols)
	blank := cell{Bg: t.cur.pen.Bg}
	for x := range row {
		row[x] = blank
	}
	return row
}

// blankRow returns an erased row: nil when the erase color is the default.
func (t *Terminal) blankRow() []cell {
	if t.cur.pen.Bg == 0 {
		return nil
	}
	row := make([]cell, t.cols)
	for x := range row {
		row[x].Bg = t.cur.pen.Bg
	}
	return row
}

// row returns a full-width screen row for writing and marks it dirty.
// Compact can retain only its nonzero prefix; newly exposed cells are zero
// even when a previous width reduction left old data in the capacity.
func (t *Terminal) row(y int) []cell {
	t.touch(y)
	scr := t.screen()
	if len(scr[y]) < t.cols {
		scr[y] = growRow(scr[y], t.cols, t.cols)
	}
	return scr[y]
}

// ensureRowPrefix prepares only the columns a print will write. Capacity
// grows geometrically for byte-sized writes, bounded by the physical row.
// Alternate-screen writers use full rows to avoid reallocating dense redraws.
func ensureRowPrefix(row *[]cell, end, cols int) []cell {
	if len(*row) < end {
		*row = growRowPrefix(*row, end, cols)
	}
	return *row
}

// Keep capacity policy and allocation outside the printing loop's hot wrapper.
//
//go:noinline
func growRowPrefix(row []cell, end, limit int) []cell {
	capacity := cap(row)
	if capacity < end {
		capacity = limit
		if cap(row) <= limit/2 {
			capacity = max(end, 2*cap(row))
		}
	}
	return growRow(row, end, capacity)
}

// ASCII replaces every cell in [from, end). Only a gap before that run
// needs clearing when an erased or narrowed row reuses its old capacity.
//
//go:noinline
func growASCIIRow(row []cell, from, end, cols int) []cell {
	if cap(row) < end {
		return growRowPrefix(row, end, cols)
	}
	old := len(row)
	row = row[:end]
	if from > old {
		clear(row[old:from])
	}
	return row
}

func growRow(row []cell, end, capacity int) []cell {
	if cap(row) < end {
		next := make([]cell, end, capacity)
		copy(next, row)
		return next
	}
	old := len(row)
	row = row[:end]
	clear(row[old:])
	return row
}

// rowCell reads an implicit zero outside a row's owned prefix.
func rowCell(row []cell, x int) cell {
	if uint(x) < uint(len(row)) {
		return row[x]
	}
	return cell{}
}

// fillCells erases cells [from, to) of row y with the current background.
// Erasing to the end of the row ends a soft wrap there.
func (t *Terminal) fillCells(y, from, to int) {
	from, to = max(from, 0), min(to, t.cols)
	if from >= to {
		return
	}
	if to == t.cols {
		if w := t.wraps(); w != nil && w[y]&rowWrap != 0 {
			w[y] &^= rowWrap
			t.touch(y)
		}
	}
	scr := t.screen()
	var row []cell
	if t.cur.pen.Bg == 0 {
		row = scr[y]
		if from >= len(row) {
			return // the requested range already consists of implicit zeros
		}
		to = min(to, len(row))
		if !t.RowDirty(y) {
			// Clean rows need an actual-change check so damage does not
			// depend on whether an all-zero tail owns storage. A row that
			// is already dirty can take the normal fast clear path.
			changed := false
			for _, c := range row[from:to] {
				if c != (cell{}) {
					changed = true
					break
				}
			}
			if !changed {
				return
			}
		}
		t.touch(y)
	} else {
		row = t.row(y)
	}
	blank := cell{Bg: t.cur.pen.Bg}
	// Erasing half of a wide character erases all of it.
	if from > 0 && row[from].R == wideTail {
		row[from-1] = blank
	}
	if to < len(row) && row[to].R == wideTail {
		row[to] = blank
	}
	if blank.Bg == 0 {
		clear(row[from:to])
		return
	}
	for x := from; x < to; x++ {
		row[x] = blank
	}
}

var decGraphics = [...]rune{
	'◆', '▒', '␉', '␌', '␍', '␊', '°', '±', '␤', '␋', '┘', '┐', '┌', '└', '┼', '⎺',
	'⎻', '─', '⎼', '⎽', '├', '┤', '┴', '┬', '│', '≤', '≥', 'π', '≠', '£', '·',
}

// characterSet consumes a single shift when choosing the next character's
// mapping. REP uses the designation in effect when the repeat runs.
func (t *Terminal) characterSet() charset {
	if t.cur.ss != 0 {
		g := t.cur.g[t.cur.ss]
		t.cur.ss = 0
		return g
	}
	return t.cur.g[t.cur.gl]
}

func (t *Terminal) print(r rune, w int) {
	// REP reprints the input scalar with the charset in effect when REP
	// runs. Saving the mapped box character would ignore later SI/SO or SCS.
	t.lastChar = r
	g := t.characterSet()
	if g == charsetBritish && r == '#' {
		r = '£'
	}
	if g == charsetGraphics && r >= 0x5f && r <= 0x7e {
		if r == 0x5f {
			r = ' '
		} else {
			r = decGraphics[r-0x60]
		}
	}
	if t.cur.wrapPending && t.autowrap {
		t.wrapLine()
		t.cur.x = t.left
	}
	t.cur.wrapPending = false
	if w > t.cols {
		// A one-column terminal cannot print a wide glyph. Native keeps
		// an empty cell with the pen and a pending wrap for the next print.
		blank := t.cur.pen
		blank.R = 0
		t.touch(t.cur.y)
		ensureRowPrefix(&t.screen()[t.cur.y], t.cur.x+1, t.cols)[t.cur.x] = blank
		t.cur.wrapPending = true
		return
	}
	// Text runs up to the right margin, unless the cursor is already
	// past it.
	edge := t.cols - 1
	if t.cur.x <= t.right {
		edge = t.right
	}
	if t.cur.x+w > edge+1 {
		if !t.autowrap {
			return
		} else {
			t.fillCells(t.cur.y, t.cur.x, edge+1)
			if w == 2 {
				spacer := t.cur.pen
				spacer.R = 0
				if t.fullWidth() {
					spacer.R = spacerHead
				}
				if t.altActive {
					t.row(t.cur.y)[t.cur.x] = spacer
				} else {
					t.touch(t.cur.y)
					ensureRowPrefix(&t.screen()[t.cur.y], t.cur.x+1, t.cols)[t.cur.x] = spacer
				}
			}
			t.wrapLine()
			t.cur.x = t.left
			if t.cur.x+w > t.right+1 {
				return // a wide character in a one-column region
			}
			edge = t.right
		}
	}
	x := t.cur.x
	var row []cell
	if t.insert || t.altActive {
		row = t.row(t.cur.y)
	} else {
		t.touch(t.cur.y)
		row = ensureRowPrefix(&t.screen()[t.cur.y], x+w, t.cols)
	}
	if t.insert && x+w <= edge {
		copy(row[x+w:edge+1], row[x:edge+1-w])
	}
	pen := t.cur.pen
	// Overwriting part of a wide character blanks its other half.
	if row[x].R == wideTail && x > 0 {
		row[x-1] = cell{}
	}
	if end := x + w; end < len(row) && row[end].R == wideTail {
		row[end] = cell{}
	}
	pen.R = r
	row[x] = pen
	if w == 2 {
		pen.R = wideTail
		row[x+1] = pen
	}
	if t.insert {
		t.fixWide(row) // the shift may have pushed half a wide character off
	}
	if x+w > edge {
		t.cur.x = edge
		t.cur.wrapPending = true
	} else {
		t.cur.x = x + w
	}
}

// wrapLine moves the cursor to the next line for autowrap and marks the
// row it left as continuing there.
func (t *Terminal) wrapLine() {
	content := t.semanticContent()
	defer t.semanticWrap(content)
	y := t.cur.y
	if !t.fullWidth() || t.cur.x != t.cols-1 {
		t.index() // soft wraps are only kept for full rows
		return
	}
	if t.top == t.bot {
		// A one-row screen: the row scrolls away as it wraps.
		t.setWrap(y, true)
	}
	if !t.index() {
		return
	}
	if t.cur.y != y {
		t.setWrap(y, true)
	} else if y > t.top {
		t.setWrap(y-1, true) // the region scrolled
	}
}

// combine adds a zero-width character (a combining mark, joiner or
// variation selector) to the character before the cursor.
func (t *Terminal) combine(r rune) {
	x, y := t.cur.x, t.cur.y
	if !t.cur.wrapPending && !(x == t.cols-1 && !t.autowrap) {
		x--
	}
	row := t.screen()[y]
	if x < 0 || row == nil {
		return
	}
	if rowCell(row, x).R == wideTail && x > 0 {
		x--
	}
	if x >= len(row) {
		if x == t.cols-1 && !t.autowrap && x > 0 && rowCell(row, x-1).R > 0 {
			x--
		} else {
			return
		}
	}
	c := &row[x]
	if c.R <= 0 {
		if x == t.cols-1 && !t.autowrap && x > 0 && row[x-1].R > 0 {
			c = &row[x-1]
		} else {
			return
		}
	}
	s := t.cellString(*c)
	if t.clusterTail(*c).count >= maxClusterRunes {
		return
	}
	if r == 0xfe0e || r == 0xfe0f {
		base := firstRune(s)
		// Native mode-off filtering uses the tailored grapheme class;
		// emoji modifier bases are separate from Extended_Pictographic.
		if !runeInRanges(base, emojiPictographs[:]) || runeInRanges(base, emojiModifierBases[:]) {
			return
		}
	}
	var modes clusterModes
	if c.R >= clusterBase {
		modes = t.x.clusters[c.R-clusterBase].joinModes
	}
	c.R = t.internAppend(s, r, uint8(t.runeWidth(c.R)), modes)
	t.touch(y)
}

// printASCII prints a run of printable ASCII. The common case (no
// charset mapping, insert mode or wide characters in the way) writes
// cells directly; anything else goes through print.
func (t *Terminal) printASCII(p []byte) {
	for len(p) > 0 {
		if t.cur.g[t.cur.gl] != charsetASCII || t.cur.ss != 0 || t.insert || t.cur.wrapPending || !t.autowrap {
			t.print(rune(p[0]), 1)
			p = p[1:]
			continue
		}
		x := t.cur.x
		edge := t.cols - 1
		if x <= t.right {
			edge = t.right
		}
		n := min(len(p), edge+1-x)
		scr := t.screen()
		row := scr[t.cur.y]
		// Hidden capacity may still contain an erased wide character.
		// Check the live prefix before exposing cells the run overwrites.
		if rowCell(row, x).R == wideTail || rowCell(row, x+n).R == wideTail {
			t.print(rune(p[0]), 1)
			p = p[1:]
			continue
		}
		if t.altActive {
			row = t.row(t.cur.y)
		} else {
			t.touch(t.cur.y)
			if len(row) < x+n {
				row = growASCIIRow(row, x, x+n, t.cols)
				scr[t.cur.y] = row
			}
		}
		row = row[x : x+n]
		if asciiVector && n >= 16 {
			fillASCII8(row, p[:n], t.cur.pen)
		} else {
			fg, bg, attrs, link := t.cur.pen.Fg, t.cur.pen.Bg, t.cur.pen.A, t.cur.pen.E
			for i, b := range p[:n] {
				c := &row[i]
				c.R, c.Fg, c.Bg, c.A, c.E = rune(b), fg, bg, attrs, link
			}
		}
		t.lastChar = rune(p[n-1])
		if x+n > edge {
			t.cur.x = edge
			t.cur.wrapPending = true
		} else {
			t.cur.x = x + n
		}
		p = p[n:]
	}
}

// index moves the cursor down a line, scrolling at the bottom margin. It
// reports whether the cursor moved or the region scrolled.
func (t *Terminal) index() bool {
	defer t.semanticIndex()
	if t.cur.y == t.bot {
		if !t.inMargins(t.cur.x) {
			return false
		}
		t.scrollUp(t.top, t.bot, 1)
		return true
	}
	if t.cur.y < t.rows-1 {
		t.cur.y++
		return true
	}
	return false
}

func (t *Terminal) reverseIndex() {
	if t.cur.y == t.top {
		if t.inMargins(t.cur.x) {
			t.scrollDown(t.top, t.bot, 1)
		}
	} else if t.cur.y > 0 {
		t.cur.y--
	}
}

// scrollUp moves rows top..bot up by n. Rows leaving the top of the main
// screen go to scrollback when the region starts at the first row.
func (t *Terminal) scrollUp(top, bot, n int) {
	n = min(n, bot-top+1)
	if n <= 0 {
		return
	}
	if !t.fullWidth() {
		t.scrollRect(top, bot, n)
		return
	}
	scr := t.screen()
	if top == 0 && !t.altActive {
		w := t.mainWrap
		for i, row := range scr[:n] {
			t.pushScrollback(row, rowFlagsAt(w, i), rowAt(scr, i+1))
		}
		open := t.sbOpen
		t.rotate(top, bot, n)
		t.sbOpen = open
		return
	}
	t.rotate(top, bot, n)
}

// rotate moves rows top..bot up by n rows (down by -n), 0 < |n| <=
// bot-top+1. The rows that leave at one end come back blank at the other,
// so programs that scroll, insert, delete and erase lines all the time
// do not allocate a row each time.
func (t *Terminal) rotate(top, bot, n int) {
	t.touchRows(top, bot+1)
	scr := t.screen()
	up := n > 0
	n = max(n, -n)
	if n == 1 {
		if up {
			row := scr[top]
			copy(scr[top:bot], scr[top+1:bot+1])
			scr[bot] = t.blank(row)
		} else {
			row := scr[bot]
			copy(scr[top+1:bot+1], scr[top:bot])
			scr[top] = t.blank(row)
		}
	} else {
		// Keep the outgoing rows for reuse when the batch is small.
		var spare [8][]cell
		recycled := spare[:0]
		if n <= len(spare) {
			if up {
				recycled = append(recycled, scr[top:top+n]...)
			} else {
				recycled = append(recycled, scr[bot-n+1:bot+1]...)
			}
		}
		in := scr[top : top+n] // where blank rows come in
		if up {
			copy(scr[top:], scr[top+n:bot+1])
			in = scr[bot-n+1 : bot+1]
		} else {
			copy(scr[top+n:bot+1], scr[top:bot+1-n])
		}
		for i := range in {
			if i < len(recycled) && recycled[i] != nil {
				in[i] = t.blank(recycled[i])
			} else {
				in[i] = t.blankRow()
			}
		}
	}
	// Soft wraps move with their rows; a row whose next row changed no
	// longer continues onto it.
	if w := t.wraps(); w != nil {
		if up {
			copy(w[top:], w[top+n:bot+1])
			clear(w[max(bot-n+1, top) : bot+1])
			if bot-n >= top {
				w[bot-n] &^= rowWrap
			}
		} else {
			copy(w[top+n:bot+1], w[top:bot+1-n])
			clear(w[top : top+n])
			w[bot] &^= rowWrap
		}
		if top > 0 {
			w[top-1] &^= rowWrap
		}
	}
	if top == 0 && !t.altActive {
		t.sbOpen = false
	}
}

func (t *Terminal) scrollDown(top, bot, n int) {
	n = min(n, bot-top+1)
	if n <= 0 {
		return
	}
	if !t.fullWidth() {
		t.scrollRect(top, bot, -n)
		return
	}
	t.rotate(top, bot, -n)
}

// scrollRect moves the cells between the left and right margins of rows
// top..bot up by n rows (down by -n), 0 < |n| <= bot-top+1, blanking the
// rows that come in. Nothing goes to the scrollback.
func (t *Terminal) scrollRect(top, bot, n int) {
	t.touchRows(max(top-1, 0), bot+1)
	scr := t.screen()
	l, r := t.left, t.right+1
	move := func(dst, src int) {
		if scr[src] == nil && scr[dst] == nil {
			return
		}
		d := t.row(dst)
		clear(d[l:r])
		if s := scr[src]; l < len(s) {
			copy(d[l:r], s[l:min(r, len(s))])
		}
	}
	erase := func(y int) {
		if scr[y] == nil && t.cur.pen.Bg == 0 {
			return
		}
		row := t.row(y)
		for x := l; x < r; x++ {
			row[x] = cell{Bg: t.cur.pen.Bg}
		}
	}
	if n > 0 {
		for y := top; y+n <= bot; y++ {
			move(y, y+n)
		}
		for y := bot - n + 1; y <= bot; y++ {
			erase(y)
		}
	} else {
		n = -n
		for y := bot; y-n >= top; y-- {
			move(y, y-n)
		}
		for y := top; y < top+n; y++ {
			erase(y)
		}
	}
	for y := max(top-1, 0); y <= bot; y++ {
		if scr[y] != nil {
			t.fixWide(scr[y])
		}
		t.setWrap(y, false)
	}
}

func (t *Terminal) insertLines(n int) {
	if t.cur.y < t.top || t.cur.y > t.bot || !t.inMargins(t.cur.x) {
		return
	}
	t.scrollDown(t.cur.y, t.bot, n)
	t.cur.x, t.cur.wrapPending = t.left, false
}

func (t *Terminal) deleteLines(n int) {
	if t.cur.y < t.top || t.cur.y > t.bot || !t.inMargins(t.cur.x) {
		return
	}
	if n = min(n, t.bot-t.cur.y+1); n > 0 {
		if t.fullWidth() {
			t.rotate(t.cur.y, t.bot, n)
		} else {
			t.scrollRect(t.cur.y, t.bot, n)
		}
	}
	t.cur.x, t.cur.wrapPending = t.left, false
}

func (t *Terminal) insertCells(n int) {
	t.cur.wrapPending = false
	if t.inMargins(t.cur.x) {
		t.shiftCells(min(n, t.right+1-t.cur.x))
	}
}

func (t *Terminal) deleteCells(n int) {
	t.cur.wrapPending = false
	if t.inMargins(t.cur.x) {
		t.shiftCells(-min(n, t.right+1-t.cur.x))
		t.setWrap(t.cur.y, false) // as ghostty: the row no longer wraps
	}
}

// inMargins reports whether column x is within the left and right margins.
func (t *Terminal) inMargins(x int) bool { return x >= t.left && x <= t.right }

// fullWidth reports whether the left and right margins span the screen.
func (t *Terminal) fullWidth() bool { return t.left == 0 && t.right == t.cols-1 }

// carriageReturn moves the cursor to the left margin, or to the first
// column from left of it.
func (t *Terminal) carriageReturn() {
	t.cur.wrapPending = false
	if t.cur.origin || t.cur.x >= t.left {
		t.cur.x = t.left
	} else {
		t.cur.x = 0
	}
}

// relPos returns the cursor position as CUP takes it (relative to the
// margins in origin mode), 0-based.
func (t *Terminal) relPos() (int, int) {
	if t.cur.origin {
		return t.cur.x - t.left, t.cur.y - t.top
	}
	return t.cur.x, t.cur.y
}

// shiftCells inserts (n > 0) or deletes (n < 0) blank cells at the cursor,
// shifting the rest of the line.
func (t *Terminal) shiftCells(n int) {
	t.cur.wrapPending = false
	if n == 0 || t.screen()[t.cur.y] == nil && t.cur.pen.Bg == 0 {
		return
	}
	row, x := t.row(t.cur.y), t.cur.x
	blank := cell{Bg: t.cur.pen.Bg}
	// Starting on the right half of a wide character splits it: the
	// left half goes too.
	if x > 0 && row[x].R == wideTail {
		row[x-1], row[x] = blank, blank
	}
	end := t.right + 1
	if n > 0 {
		copy(row[x+n:end], row[x:end])
		for i := x; i < x+n; i++ {
			row[i] = blank
		}
	} else {
		copy(row[x:end], row[x-n:end])
		for i := end + n; i < end; i++ {
			row[i] = blank
		}
	}
	t.fixWide(row)
}

// fixWide blanks halves of wide characters that lost their other half.
func (t *Terminal) fixWide(row []cell) {
	for x := range row {
		c := &row[x]
		switch {
		case c.R == wideTail:
			if x == 0 || row[x-1].R <= 0 || t.runeWidth(row[x-1].R) != 2 {
				c.R = 0
			}
		case c.R > 0x7f && t.runeWidth(c.R) == 2:
			if x+1 == len(row) || row[x+1].R != wideTail {
				c.R = 0
			}
		}
	}
}

// eraseDisplay is ED, or DECSED when selective.
func (t *Terminal) eraseDisplay(mode int, selective bool) {
	t.cur.wrapPending = false
	keep := selective || t.protISO // leave protected characters
	switch mode {
	case 0:
		t.erase(t.cur.y, t.cur.x, t.cols, keep)
		t.eraseFull(t.cur.y+1, t.rows, keep)
	case 1:
		t.eraseFull(0, t.cur.y, keep)
		t.erase(t.cur.y, 0, t.cur.x+1, keep)
	case 2:
		if !t.altActive && screenAtPrompt(t.main, t.mainWrap) {
			t.scrollClear()
		}
		t.eraseFull(0, t.rows, keep)
	case 3:
		if !selective {
			t.ClearScrollback()
		}
	case 22:
		t.scrollClear()
	}
}

// scrollClear moves written rows into history, regardless of scrolling
// margins or protection. Trailing empty rows stay on screen. New rows have
// the default background; the cursor keeps its pen and input modes.
func (t *Terminal) scrollClear() {
	scr := t.screen()
	n := t.rows
	for n > 0 {
		written := false
		for _, c := range scr[n-1] {
			if c.R != 0 || c.Bg != 0 {
				written = true
				break
			}
		}
		if written {
			break
		}
		n--
	}
	if n == 0 {
		return
	}
	if !t.altActive {
		for y, row := range scr[:n] {
			t.pushScrollback(row, rowFlagsAt(t.mainWrap, y), rowAt(scr, y+1))
		}
	}
	// rotate normally erases with the pen's background. Scroll-and-clear
	// adds fresh rows instead, without changing subsequent printed colors.
	bg := t.cur.pen.Bg
	t.cur.pen.Bg = 0
	t.rotate(0, t.rows-1, n)
	t.cur.pen.Bg = bg
	if t.cur.y < n {
		t.cur.x, t.cur.y = 0, 0
	} else {
		t.cur.y -= n
	}
}

// eraseLine is EL, or DECSEL when selective.
func (t *Terminal) eraseLine(mode int, selective bool) {
	t.cur.wrapPending = false
	keep := selective || t.protISO
	switch mode {
	case 0:
		t.erase(t.cur.y, t.cur.x, t.cols, keep)
	case 1:
		t.erase(t.cur.y, 0, t.cur.x+1, keep)
	case 2:
		// EL clears cells, not the shell's row prompt annotation.
		t.erase(t.cur.y, 0, t.cols, keep)
	}
}

// eraseFull erases rows [from, to); with keep, protected characters stay.
func (t *Terminal) eraseFull(from, to int, keep bool) {
	if !keep {
		t.eraseRows(from, to)
		return
	}
	for y := from; y < to; y++ {
		t.erase(y, 0, t.cols, true)
	}
}

// erase erases cells [from, to) of row y; with keep, protected
// characters stay.
func (t *Terminal) erase(y, from, to int, keep bool) {
	row := t.screen()[y]
	if !keep || row == nil {
		t.fillCells(y, from, to)
		return
	}
	for x := max(from, 0); x < min(to, t.cols); {
		if rowCell(row, x).A&AttrProtected != 0 {
			x++
			continue
		}
		end := x + 1
		for end < min(to, t.cols) && rowCell(row, end).A&AttrProtected == 0 {
			end++
		}
		t.fillCells(y, x, end)
		x = end
	}
}

// protect sets whether characters printed from now on are protected,
// by DECSCA or (iso) SPA/EPA.
func (t *Terminal) protect(on, iso bool) {
	if on {
		t.cur.pen.A |= AttrProtected
		t.protISO = iso
	} else {
		t.cur.pen.A &^= AttrProtected
	}
}

func (t *Terminal) setX(x int) {
	t.cur.wrapPending = false
	t.cur.x = min(max(x, 0), t.cols-1)
}

// moveTo positions the cursor, honoring origin mode.
func (t *Terminal) moveTo(x, y int) {
	t.cur.wrapPending = false
	minX, maxX, minY, maxY := 0, t.cols-1, 0, t.rows-1
	if t.cur.origin {
		x += t.left
		y += t.top
		minX, maxX, minY, maxY = t.left, t.right, t.top, t.bot
	}
	t.cur.x = min(max(x, minX), maxX)
	t.cur.y = min(max(y, minY), maxY)
}

// cursorLeft moves the cursor left n columns (BS, CUB). With reverse
// wraparound it goes on from the end of the previous row when this one
// continues it (45), or from any previous row and from the top margin to
// the bottom one (1045), as xterm does.
func (t *Terminal) cursorLeft(n int) {
	left := 0
	if t.cur.x >= t.left {
		left = t.left // stop at the left margin unless already left of it
	}
	if !t.autowrap || !t.revWrap && !t.revWrapExt {
		t.setX(max(t.cur.x-n, left))
		return
	}
	ext := t.revWrapExt
	if t.cur.wrapPending {
		// The pending wrap counts as one step.
		t.cur.wrapPending = false
		if n--; n == 0 {
			return
		}
	}
	if t.cur.x == left && !ext && t.cur.y <= t.top {
		t.cur.x, t.cur.y = left, t.top
		return
	}
	for {
		step := min(t.cur.x-left, n)
		t.cur.x -= step
		if n -= step; n == 0 {
			return
		}
		switch {
		case t.cur.y == t.top:
			if !ext {
				return
			}
			t.cur.x, t.cur.y = t.right, t.bot
		case t.cur.y == 0, !ext && !t.wrapped(t.cur.y):
			return
		default:
			t.cur.x, t.cur.y = t.right, t.cur.y-1
		}
		n--
	}
}

func (t *Terminal) cursorUp(n int) {
	t.cur.wrapPending = false
	lim := 0
	if t.cur.y >= t.top {
		lim = t.top
	}
	t.cur.y = max(t.cur.y-n, lim)
}

func (t *Terminal) cursorDown(n int) {
	t.cur.wrapPending = false
	lim := t.rows - 1
	if t.cur.y <= t.bot {
		lim = t.bot
	}
	t.cur.y = min(t.cur.y+n, lim)
}

// --- tab stops ---

func (t *Terminal) isTab(x int) bool {
	tabs := t.tabs()
	if tabs == nil {
		return x%8 == 0
	}
	return tabs[x/64]&(1<<(x%64)) != 0
}

// setTab sets or clears the tab stop at column x.
func (t *Terminal) setTab(x int, on bool) {
	if t.tabs() == nil {
		if t.isTab(x) == on {
			return
		}
		t.ext().tabs = defaultTabs(t.cols)
	}
	tabs := t.x.tabs
	if on {
		tabs[x/64] |= 1 << (x % 64)
	} else {
		tabs[x/64] &^= 1 << (x % 64)
	}
}

func defaultTabs(cols int) []uint64 {
	tabs := make([]uint64, (cols+63)/64)
	for x := 0; x < cols; x += 8 {
		tabs[x/64] |= 1 << (x % 64)
	}
	return tabs
}

// nextTab returns the next tab stop after x, or the last column.
func (t *Terminal) nextTab(x int) int {
	limit := t.cols - 1
	if x <= t.right {
		limit = t.right // tabs stop at the right margin
	}
	if x >= limit {
		return x
	}
	for x++; x < limit; x++ {
		if t.isTab(x) {
			return x
		}
	}
	return limit
}

// prevTab returns the tab stop before x, or the first column.
func (t *Terminal) prevTab(x int) int {
	for x--; x > 0; x-- {
		if t.isTab(x) {
			return x
		}
	}
	return 0
}

// Column changes restore the default tab stops, as libghostty does.
// Height-only resizes leave the current tab table alone.
func (t *Terminal) resizeTabs(oldCols int) {
	if oldCols != t.cols && t.x != nil {
		t.x.tabs = nil
	}
}

func (t *Terminal) sgr() {
	if t.nparams <= 1 {
		if t.nparams == 0 || t.sub == 0 && t.params[0] == 0 {
			if t.cur.pen.E != 0 {
				t.setUnderlineColor(0)
			}
			t.cur.pen = cell{A: t.cur.pen.A & AttrProtected, E: t.cur.pen.E}
			return
		}
		if t.sub == 0 && t.params[0] >= 30 && t.params[0] <= 37 {
			t.cur.pen.Fg = colorIndexed | Color(t.params[0]-30)
			return
		}
	}
	p := &t.cur.pen
	nparams := int(t.nparams)
	for i := 0; i < nparams; i++ {
		v := t.params[i]
		// Skip the sub-parameters of a parameter we do not interpret.
		next := i + 1
		for next < nparams && t.subParam(next) {
			next++
		}
		colon := t.subParam(i + 1)
		if colon && v != 4 && v != 38 && v != 48 && v != 58 {
			i = next - 1 // unknown colon groups do not apply their leading SGR
			continue
		}
		switch {
		case v == 0:
			if p.E != 0 {
				t.setUnderlineColor(0)
			}
			*p = cell{A: p.A & AttrProtected, E: p.E}
		case v == 1:
			p.A |= AttrBold
		case v == 2:
			p.A |= AttrFaint
		case v == 3:
			p.A |= AttrItalic
		case v == 4:
			if colon && next == i+1 {
				break
			}
			style := 1
			if next > i+1 {
				style = int(t.params[i+1])
			}
			p.A = p.A&^ulMask | Attr(min(style, 5))<<ulShift
		case v == 5 || v == 6:
			p.A |= AttrBlink
		case v == 7:
			p.A |= AttrInverse
		case v == 8:
			p.A |= AttrHidden
		case v == 9:
			p.A |= AttrStrikethrough
		case v == 21:
			p.A = p.A&^ulMask | 2<<ulShift
		case v == 22:
			p.A &^= AttrBold | AttrFaint
		case v == 23:
			p.A &^= AttrItalic
		case v == 24:
			p.A &^= ulMask
		case v == 25:
			p.A &^= AttrBlink
		case v == 27:
			p.A &^= AttrInverse
		case v == 28:
			p.A &^= AttrHidden
		case v == 29:
			p.A &^= AttrStrikethrough
		case v >= 30 && v <= 37:
			p.Fg = colorIndexed | Color(v-30)
		case v == 38, v == 48, v == 58:
			c, end, valid := t.extColor(i, next)
			if valid {
				switch v {
				case 38:
					p.Fg = c
				case 48:
					p.Bg = c
				case 58:
					t.setUnderlineColor(c)
				}
			}
			next = end
		case v == 39:
			p.Fg = 0
		case v >= 40 && v <= 47:
			p.Bg = colorIndexed | Color(v-40)
		case v == 49:
			p.Bg = 0
		case v == 53:
			p.A |= AttrOverline
		case v == 55:
			p.A &^= AttrOverline
		case v == 59:
			t.setUnderlineColor(0)
		case v >= 90 && v <= 97:
			p.Fg = colorIndexed | Color(v-90+8)
		case v >= 100 && v <= 107:
			p.Bg = colorIndexed | Color(v-100+8)
		}
		i = next - 1
	}
}

// extColor parses the color after an extended-color parameter at i, in
// either the colon form (38:5:n, 38:2::r:g:b, 38:2:r:g:b) or the legacy
// semicolon form (38;5;n, 38;2;r;g;b). It returns the color and the index
// of the next parameter to interpret and whether a color was complete.
// Invalid colors leave the old color in place; unconsumed parameters can
// still be independent SGRs. Colon groups require exactly three RGB
// channels, optionally preceded by a colorspace slot.
func (t *Terminal) extColor(i, next int) (Color, int, bool) {
	nparams := int(t.nparams)
	if i+1 >= nparams {
		return 0, i + 1, false
	}
	switch t.params[i+1] {
	case 5:
		if i+2 < nparams {
			return colorIndexed | Color(min(t.params[i+2], 0xffff)&0xff), i + 3, true
		}
	case 2:
		if i+4 < nparams {
			if !t.subParam(i + 1) { // legacy semicolon form
				return rgb(int(t.params[i+2]), int(t.params[i+3]), int(t.params[i+4])), i + 5, true
			}
			switch next - i {
			case 5:
				return rgb(int(t.params[i+2]), int(t.params[i+3]), int(t.params[i+4])), next, true
			case 6:
				return rgb(int(t.params[i+3]), int(t.params[i+4]), int(t.params[i+5])), next, true
			}
			return 0, max(next, i+2), false
		}
	}
	return 0, i + 1, false
}

func rgb(r, g, b int) Color {
	return colorRGB | Color(min(r, 0xffff)&0xff)<<16 | Color(min(g, 0xffff)&0xff)<<8 | Color(min(b, 0xffff)&0xff)
}

func (t *Terminal) oscDispatch() {
	if t.oscOver {
		return
	}
	s := t.osc
	switch {
	case len(s) >= 2 && (s[0] == '0' || s[0] == '2') && s[1] == ';':
		t.title = string(s[2:])
	case len(s) >= 2 && s[0] == '8' && s[1] == ';':
		// OSC 8 ; params ; uri: start a hyperlink, or end it (no uri).
		params, uri, ok := strings.Cut(string(s[2:]), ";")
		if !ok || uri == "" {
			t.setLink("")
		} else {
			t.setLink(params + ";" + uri)
		}
	case len(s) >= 5 && string(s[:4]) == "133;":
		t.semanticCommand(s[4:])
	default:
		t.oscColor(s)
	}
}

// internExtra shares uncommon attributes without enlarging ordinary cells.
// Recent pairs are reused. Collection drops unreachable pairs; at the
// 16-bit limit, a full search also reuses older pairs. Zero means the
// default pair, or that all 65535 entries are live and the new pair cannot
// be stored. Pen setters retain their current pair in the latter case.
func (t *Terminal) internExtra(v cellExtra) uint16 {
	if v == (cellExtra{}) {
		return 0
	}
	x := t.ext()
	for i := len(x.rare) - 1; i >= max(len(x.rare)-16, 0); i-- {
		if x.rare[i] == v {
			return uint16(i + 1)
		}
	}
	if len(x.rare) >= x.rareCap {
		t.compactExtras()
	}
	if len(x.rare) >= 0xffff {
		for i, old := range x.rare {
			if old == v {
				return uint16(i + 1)
			}
		}
		return 0
	}
	x.rare = append(x.rare, v)
	return uint16(len(x.rare))
}

func (t *Terminal) setPenExtra(v cellExtra) {
	if id := t.internExtra(v); id != 0 || v == (cellExtra{}) {
		t.cur.pen.E = id
	}
}

func (t *Terminal) setLink(target string) {
	v := t.extraAt(t.cur.pen.E)
	v.link = target
	t.setPenExtra(v)
}

func (t *Terminal) setUnderlineColor(c Color) {
	v := t.extraAt(t.cur.pen.E)
	v.underline = c
	t.setPenExtra(v)
}

// compactExtras drops the attribute pairs no cell or cursor refers to, renumbering
// the rest, and sets the next compaction point.
func (t *Terminal) compactExtras() {
	x := t.x
	if len(x.rare) > 0 {
		remap := make([]uint16, len(x.rare))
		var kept []cellExtra
		ref := func(l *uint16) {
			if *l == 0 {
				return
			}
			i := *l - 1
			if remap[i] == 0 {
				kept = append(kept, x.rare[i])
				remap[i] = uint16(len(kept))
			}
			*l = remap[i]
		}
		t.visitExtraRefs(ref)
		x.rare = kept
	}
	x.rareCap = min(0xffff, max(16, 2*len(x.rare)))
}

// visitExtraRefs covers the cells and pens which can still supply attributes.
// An invalid saved cursor cannot restore its pen; clear its old index so
// that it cannot become a dangling reference after its table is collected.
func (t *Terminal) visitExtraRefs(ref func(*uint16)) {
	for _, scr := range [][][]cell{t.main, t.alt} {
		for _, row := range scr {
			for i := range row {
				ref(&row[i].E)
			}
		}
	}
	ref(&t.cur.pen.E)
	if t.saved != nil {
		for _, saved := range [...]*savedCursor{&t.saved.screen[0], &t.saved.screen[1]} {
			if saved.valid {
				ref(&saved.pen.E)
			} else {
				saved.pen.E = 0
			}
		}
	}
}

func (t *Terminal) extraAt(id uint16) cellExtra {
	if id == 0 {
		return cellExtra{}
	}
	return t.x.rare[id-1]
}

// link returns a cell's hyperlink target ("params;uri"), or "".
func (t *Terminal) link(id uint16) string {
	return t.extraAt(id).link
}

// --- characters with combining marks ---

// cellString returns the characters a cell shows.
func (t *Terminal) cellString(c cell) string {
	if c.R >= clusterBase {
		return t.x.clusters[c.R-clusterBase].text
	}
	return string(c.R)
}

// runeWidth returns the width of a cell's character.
func (t *Terminal) runeWidth(r rune) int {
	if r >= clusterBase {
		return int(t.x.clusters[r-clusterBase].width)
	}
	return CodepointWidth(r)
}

// intern returns the cell rune for a character with combining marks.
// Recent entries are reused; when the table is full, entries no cell
// refers to any more are dropped.
func (t *Terminal) intern(s cluster) rune {
	x := t.ext()
	for i := len(x.clusters) - 1; i >= max(len(x.clusters)-16, 0); i-- {
		old := &x.clusters[i]
		if old.text == s.text && old.width == s.width && old.joinModes == s.joinModes {
			return clusterBase + rune(i)
		}
	}
	return t.storeCluster(s)
}

// internAppend checks recent clusters before allocating the appended text.
func (t *Terminal) internAppend(prefix string, r rune, width uint8, modes clusterModes) rune {
	x := t.ext()
	var storage [utf8.UTFMax]byte
	suffix := utf8.AppendRune(storage[:0], r)
	for i := len(x.clusters) - 1; i >= max(len(x.clusters)-16, 0); i-- {
		old := &x.clusters[i]
		if old.width == width && old.joinModes == modes && len(old.text) == len(prefix)+len(suffix) &&
			old.text[:len(prefix)] == prefix && old.text[len(prefix):] == string(suffix) {
			return clusterBase + rune(i)
		}
	}
	return t.storeCluster(cluster{text: prefix + string(r), width: width, joinModes: modes})
}

func (t *Terminal) storeCluster(s cluster) rune {
	s.tail = graphemeTailOf(s.text)
	x := t.ext()
	if len(x.clusters) >= x.clusterCap {
		t.compactClusters()
	}
	x.clusters = append(x.clusters, s)
	return clusterBase + rune(len(x.clusters)-1)
}

// compactClusters drops the clusters no cell refers to, renumbering the
// rest, and sets the next compaction point.
func (t *Terminal) compactClusters() {
	x := t.x
	if len(x.clusters) > 0 {
		remap := make([]rune, len(x.clusters))
		var kept []cluster
		ref := func(r *rune) {
			if *r < clusterBase {
				return
			}
			i := *r - clusterBase
			if remap[i] == 0 {
				kept = append(kept, x.clusters[i])
				remap[i] = clusterBase + rune(len(kept)-1)
			}
			*r = remap[i]
		}
		t.visitClusterRefs(ref)
		x.clusters = kept
	}
	x.clusterCap = max(64, 2*len(x.clusters))
}

// visitClusterRefs visits only stored cells. REP remembers the last
// independently printed scalar, rather than its subsequently attached marks.
func (t *Terminal) visitClusterRefs(ref func(*rune)) {
	for _, scr := range [][][]cell{t.main, t.alt} {
		for _, row := range scr {
			for i := range row {
				ref(&row[i].R)
			}
		}
	}
}

// --- scrollback ---

// pushScrollback serializes row onto the scrollback, evicting the oldest
// lines past the line and byte budgets. A row that continues on the next
// one (wrapped, with next the row below) is kept whole, and the next row
// is appended to its line.
func (t *Terminal) pushScrollback(row []cell, flags rowFlags, next []cell) {
	wrapped := flags&rowWrap != 0
	if t.sbMaxLines <= 0 {
		return
	}
	encoded := t.scratch[:0]
	if wrapped {
		// A spacer head at the end is left out: it is not text, and the
		// wide character after it recreates it on replay at this width.
		end := t.cols
		if rowCell(row, end-1).R == spacerHead {
			end--
		}
		encoded = t.appendRowHistory(encoded[:0], row, end, false, true)
	} else {
		encoded = t.appendRowHistory(encoded[:0], row, t.cols, true, true)
	}
	join := t.sbOpen && len(t.sb) > t.sbStart && len(t.sb[len(t.sb)-1])+len(encoded) <= maxLogicalLine
	if semantic := flags & rowSemanticMask; semantic != 0 || join && bytes.Contains(t.sb[len(t.sb)-1], []byte{0, 0, 'P'}) {
		if semantic != 0 && contentEnd(row) == 0 {
			// An annotated empty row is meaningful to native reflow.
			encoded = append(encoded, 0, 1, ' ')
		}
		n := len(encoded)
		encoded = append(encoded, 0, 0, 0, 0)
		copy(encoded[4:], encoded[:n])
		encoded[0], encoded[1], encoded[2], encoded[3] = 0, 0, 'P', byte(semantic>>1)
		join = join && len(t.sb[len(t.sb)-1])+len(encoded) <= maxLogicalLine
	}
	putBack := join && t.sbSpacer && !startsWide(row)
	if join && !wrapped && contentEnd(row) < t.cols {
		// Replayed, this row starts by wrapping, so something must be
		// printed, and the wrap may scroll in a row painted in the current
		// background: clear the rest of it.
		if len(encoded) == 0 {
			encoded = append(encoded, 0, 1, ' ')
		}
		encoded = append(encoded, "\x1b[K"...)
	}
	if putBack {
		// The line left out a spacer head, but this row no longer starts
		// with the wide character that made it: keep the column.
		encoded = append(encoded, 0)
		copy(encoded[1:], encoded)
		encoded[0] = ' '
	}
	if n := len(t.sb); join {
		line := append(t.sb[n-1], encoded...)
		if !wrapped && cap(line) > len(line)+len(line)/4 {
			line = append([]byte(nil), line...)
		}
		t.sbBytes += len(line) - len(t.sb[n-1])
		t.sb[n-1] = line
	} else {
		var reusable []byte
		if len(t.sb) > t.sbStart && (len(t.sb)-t.sbStart >= t.sbMaxLines || t.sbBytes+len(encoded)+24 > t.sbMaxBytes) {
			// This line will evict the oldest one. Reuse its storage so
			// steady scrolling does not allocate for every log line.
			reusable = t.sb[t.sbStart]
			t.sbBytes -= len(reusable) + 24
			t.sb[t.sbStart] = nil
			t.sbStart++
			if cap(reusable) > 2*max(len(encoded), 64) {
				reusable = nil // a short line must not pin an old long line
			}
		}
		line := append(reusable[:0], encoded...)
		t.sb = append(t.sb, line)
		t.sbBytes += len(line) + 24
	}
	t.sbOpen = wrapped
	t.sbSpacer = wrapped && rowCell(row, t.cols-1).R == spacerHead
	t.scratch = encoded
	if len(encoded) > 4096 {
		t.scratch = nil // do not keep a large buffer on an idle terminal
	}
	for n := len(t.sb) - t.sbStart; n > t.sbMaxLines || (t.sbBytes > t.sbMaxBytes && n > 1); n-- {
		t.sbBytes -= len(t.sb[t.sbStart]) + 24
		t.sb[t.sbStart] = nil
		t.sbStart++
	}
	if t.sbStart > 64 && t.sbStart*2 > len(t.sb) {
		n := copy(t.sb, t.sb[t.sbStart:])
		clear(t.sb[n:])
		t.sb = t.sb[:n]
		t.sbStart = 0
	}
}

func rowBlank(row []cell) bool {
	for _, c := range row {
		if !isZero(c) && !(c.R == ' ' && c.Fg == 0 && c.Bg == 0 && c.A == 0 && c.E == 0) {
			return false
		}
	}
	return true
}

// rowAt returns scr[i], or nil past the end.
func rowAt(scr [][]cell, i int) []cell {
	if i < len(scr) {
		return scr[i]
	}
	return nil
}

// wrapEnd returns how much of a row that continues on next is part of
// the line: all of it, but for a blank left at its end because the wide
// character starting next did not fit there.
func (t *Terminal) wrapEnd(row, next []cell) int {
	// Replayed, the wide character wraps and leaves a spacer head in its
	// own pen, so leave it out only if that recreates the original style.
	if last := rowCell(row, t.cols-1); last.R == spacerHead && startsWide(next) && styleOf(last) == styleOf(next[0]) {
		return t.cols - 1
	}
	return t.cols
}

// contentEnd returns the length of row without its trailing default
// blanks.
func contentEnd(row []cell) int {
	end := len(row)
	for end > 0 && isZeroAt(&row[end-1]) {
		end--
	}
	return end
}

// isZero reports whether c is a blank in the default style (a spacer
// head is a blank).
func isZero(c cell) bool { return isZeroAt(&c) }

// isZeroAt is isZero for a cell in place (field tests are about twice
// as fast as comparing whole cells in the trimming loops).
func isZeroAt(c *cell) bool {
	// A cell is two words: R and Fg, then Bg, A and E.
	w := *(*uint64)(unsafe.Pointer(c))
	return *(*uint64)(unsafe.Pointer(&c.Bg)) == 0 && (w == 0 || w == spacerWord)
}

var spacerWord = func() uint64 {
	c := cell{R: spacerHead}
	return *(*uint64)(unsafe.Pointer(&c))
}()

// startsWide reports whether row starts with a wide character.
func startsWide(row []cell) bool { return len(row) > 1 && row[1].R == wideTail }

// zeroRow reports whether row is blank in the default style (nil rows
// are).
func zeroRow(row []cell) bool {
	for i := range row {
		if !isZeroAt(&row[i]) {
			return false
		}
	}
	return true
}
