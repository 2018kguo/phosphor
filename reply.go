package phosphor

import (
	"bytes"
	"encoding/hex"
	"strconv"
	"strings"
)

// The identity the terminal reports. DA1 claims a VT220 with ANSI color
// (as ghostty does): level 2 is what the cursor reports below follow
// (DECXCPR without a page number, DECSCL 62).
const (
	replyDA1       = "\x1b[?62;22c"
	replyDA2       = "\x1b[>1;10;0c"
	replyDA3       = "\x1bP!|00000000\x1b\\"
	replyXTVERSION = "\x1bP>|phosphor\x1b\\"
)

// sendReply writes b to the reply writer and keeps b's storage for the
// next reply.
func (t *Terminal) sendReply(b []byte) {
	if len(b) > 0 {
		t.cfg.reply.Write(b)
	}
	t.scratch = b[:0]
}

// csiQuery answers the CSI sequence ending in c if it is a query, and
// reports whether it was. Only called with a reply writer.
func (t *Terminal) csiQuery(c byte) bool {
	b := t.scratch[:0]
	switch {
	case c == 'c' && t.inter == 0 && (t.marker == 0 || t.marker == '>' || t.marker == '='): // DA
		switch {
		case t.param(0, 0) != 0: // not a request
		case t.marker == 0:
			b = append(b, replyDA1...)
		case t.marker == '>':
			b = append(b, replyDA2...)
		default:
			b = append(b, replyDA3...)
		}
	case c == 'n' && t.inter == 0 && (t.marker == 0 || t.marker == '?'):
		b = t.appendDSR(b)
	case c == 'p' && t.inter == '$' && (t.marker == 0 || t.marker == '?'): // DECRQM
		mode := 0
		if t.nparams > 0 {
			mode = t.params[0]
		}
		b = append(b, "\x1b["...)
		var state int
		if t.marker == '?' {
			b = append(b, '?')
			state = t.decModeState(mode)
		} else {
			state = t.ansiModeState(mode)
		}
		b = strconv.AppendInt(b, int64(mode), 10)
		b = append(b, ';')
		b = strconv.AppendInt(b, int64(state), 10)
		b = append(b, "$y"...)
	case c == 'q' && t.marker == '>' && t.inter == 0: // XTVERSION
		if t.param(0, 0) != 0 {
			return true
		}
		b = append(b, replyXTVERSION...)
	case c == 'u' && t.marker == '?' && t.inter == 0: // kitty keyboard flags
		flags := 0
		if n := len(t.kitty); n > 0 {
			flags = int(t.kitty[n-1])
		}
		b = append(b, "\x1b[?"...)
		b = strconv.AppendInt(b, int64(flags), 10)
		b = append(b, 'u')
	case c == 't' && t.marker == 0 && t.inter == 0 && t.param(0, 0) == 18: // XTWINOPS: text area size
		b = append(b, "\x1b[8;"...)
		b = strconv.AppendInt(b, int64(t.rows), 10)
		b = append(b, ';')
		b = strconv.AppendInt(b, int64(t.cols), 10)
		b = append(b, 't')
	default:
		return false
	}
	t.sendReply(b)
	return true
}

// appendDSR appends the answer to a device status report request (CSI n
// or CSI ? n), if it is one this terminal answers.
func (t *Terminal) appendDSR(b []byte) []byte {
	appendPos := func(b []byte) []byte {
		x, y := t.relPos()
		b = strconv.AppendInt(b, int64(y+1), 10)
		b = append(b, ';')
		b = strconv.AppendInt(b, int64(x+1), 10)
		return append(b, 'R')
	}
	if t.marker == 0 {
		switch t.param(0, 0) {
		case 5: // status: OK
			return append(b, "\x1b[0n"...)
		case 6: // cursor position (CPR)
			return appendPos(append(b, "\x1b["...))
		}
		return b
	}
	switch t.param(0, 0) {
	case 6: // DECXCPR: no page number at VT220 level
		return appendPos(append(b, "\x1b[?"...))
	case 15: // printer: none
		return append(b, "\x1b[?13n"...)
	case 26: // keyboard: North American, ready, LK201
		return append(b, "\x1b[?27;1;0;0n"...)
	case 996: // color scheme: 1 dark, 2 light, from the background
		bg := t.Palette().Background
		if 299*int(bg.R)+587*int(bg.G)+114*int(bg.B) < 128000 {
			return append(b, "\x1b[?997;1n"...)
		}
		return append(b, "\x1b[?997;2n"...)
	}
	return b
}

// decModeState returns the DECRQM state of DEC private mode mode: 1 set,
// 2 reset, 0 not recognized.
func (t *Terminal) decModeState(mode int) int {
	on, ok := t.modeValue(mode)
	if !ok {
		switch mode {
		case 47, 1047, 1049:
			on = t.altActive
		case 66: // DECNKM, the keypad mode DECKPAM sets
			on = t.appKeypad
		case 2026:
			on = t.sync
		default:
			return 0
		}
	}
	if on {
		return 1
	}
	return 2
}

// ansiModeState returns the DECRQM state of ANSI mode mode.
func (t *Terminal) ansiModeState(mode int) int {
	var on bool
	switch mode {
	case 4:
		on = t.insert
	case 20:
		on = t.newline
	default:
		return 0
	}
	if on {
		return 1
	}
	return 2
}

// isDCSQuery reports whether a DCS string starting with p (two bytes) is
// a query this terminal answers: DECRQSS ($q) or XTGETTCAP (+q).
func isDCSQuery(p []byte) bool { return (p[0] == '$' || p[0] == '+') && p[1] == 'q' }

// dcsDispatch answers the DCS string in t.osc, if it is a query.
func (t *Terminal) dcsDispatch() {
	p := t.osc
	if t.cfg == nil || t.cfg.reply == nil || len(p) < 2 || !isDCSQuery(p) {
		return
	}
	b := t.scratch[:0]
	if p[0] == '$' {
		b = t.appendDECRQSS(b, string(p[2:]))
	} else {
		b = t.appendXTGETTCAP(b, string(p[2:]))
	}
	t.sendReply(b)
}

// appendDECRQSS appends the answer to DECRQSS for the setting named q.
func (t *Terminal) appendDECRQSS(b []byte, q string) []byte {
	ok := "\x1bP1$r"
	appendPair := func(b []byte, a, z int, final string) []byte {
		b = append(b, ok...)
		b = strconv.AppendInt(b, int64(a), 10)
		b = append(b, ';')
		b = strconv.AppendInt(b, int64(z), 10)
		return append(b, final...)
	}
	switch q {
	case "m": // SGR, as "0;1;31m"
		b = append(b, ok...)
		n := len(b)
		b = appendSGR(b, t.cur.pen)
		b = append(b[:n], b[n+2:]...) // drop the CSI
	case "r":
		b = appendPair(b, t.top+1, t.bot+1, "r")
	case "s":
		b = appendPair(b, t.left+1, t.right+1, "s")
	case " q":
		b = append(b, ok...)
		b = strconv.AppendInt(b, int64(max(t.cursorStyle, 1)), 10)
		b = append(b, " q"...)
	case "\"q":
		b = append(b, ok...)
		if t.cur.pen.A&AttrProtected != 0 {
			b = append(b, '1')
		} else {
			b = append(b, '0')
		}
		b = append(b, "\"q"...)
	case "\"p":
		b = append(b, ok...)
		b = append(b, "62;1\"p"...)
	default:
		return append(b, "\x1bP0$r\x1b\\"...)
	}
	return append(b, "\x1b\\"...)
}

// appendXTGETTCAP answers XTGETTCAP for the hex-encoded terminfo
// capabilities in q (separated by ';'), one reply each.
func (t *Terminal) appendXTGETTCAP(b []byte, q string) []byte {
	for _, name := range strings.Split(q, ";") {
		raw, err := hex.DecodeString(name)
		var val string
		switch string(raw) {
		case "Co", "colors":
			val = "256"
		}
		if err != nil || val == "" {
			b = append(b, "\x1bP0+r"...)
			b = append(b, name...)
		} else {
			b = append(b, "\x1bP1+r"...)
			b = append(b, name...)
			b = append(b, '=')
			b = hex.AppendEncode(b, []byte(val))
		}
		b = append(b, "\x1b\\"...)
	}
	return b
}

// conf returns t.cfg, allocating it.
func (t *Terminal) conf() *config {
	if t.cfg == nil {
		t.cfg = &config{}
	}
	return t.cfg
}

// oscColor handles the OSC strings that set, query and reset colors: 4
// (palette), 10, 11, 12 (foreground, background, cursor), 104, 110, 111
// and 112 (reset). Others are ignored.
func (t *Terminal) oscColor(s []byte) {
	n, r, _ := bytes.Cut(s, []byte{';'})
	var num, rest string
	switch string(n) { // no allocation unless it is one of these
	case "4", "10", "11", "12", "104", "110", "111", "112":
		num, rest = string(n), string(r)
	default:
		return
	}
	switch num {
	case "4":
		for rest != "" {
			var idx, spec string
			idx, rest, _ = strings.Cut(rest, ";")
			spec, rest, _ = strings.Cut(rest, ";")
			n, err := strconv.Atoi(idx)
			if err != nil || n < 0 || n > 255 {
				return
			}
			t.colorOp(uint16(n), spec, "4;"+idx+";")
		}
	case "10", "11", "12":
		// OSC 10 ; a ; b sets (or queries) 10 to a and 11 to b.
		slot := slotFg + uint16(num[1]-'0')
		for _, spec := range strings.Split(rest, ";") {
			if slot > slotCursor {
				break
			}
			t.colorOp(slot, spec, strconv.Itoa(int(slot-slotFg+10))+";")
			slot++
		}
	case "104":
		if rest == "" {
			t.resetColors(func(i uint16) bool { return i < 256 })
			return
		}
		for _, idx := range strings.Split(rest, ";") {
			if n, err := strconv.Atoi(idx); err == nil && n >= 0 && n <= 255 {
				t.resetColors(func(i uint16) bool { return i == uint16(n) })
			}
		}
	case "110", "111", "112":
		slot := slotFg + uint16(num[2]-'0')
		t.resetColors(func(i uint16) bool { return i == slot })
	}
}

// colorOp sets palette slot to spec, or with spec "?" reports it, the
// reply starting "OSC " + prefix.
func (t *Terminal) colorOp(slot uint16, spec, prefix string) {
	if spec == "?" {
		if t.cfg == nil || t.cfg.reply == nil {
			return
		}
		p := t.Palette()
		c := *p.slot(slot)
		b := append(t.scratch[:0], "\x1b]"...)
		b = append(b, prefix...)
		b = append(b, "rgb:"...)
		for i, v := range [3]uint8{c.R, c.G, c.B} {
			if i > 0 {
				b = append(b, '/')
			}
			b = hex.AppendEncode(b, []byte{v, v})
		}
		if t.oscBel {
			b = append(b, '\a')
		} else {
			b = append(b, "\x1b\\"...)
		}
		t.sendReply(b)
		return
	}
	c, ok := parseColor(spec)
	if !ok {
		return
	}
	cfg := t.conf()
	t.touchAll()
	for i := range cfg.colors {
		if cfg.colors[i].i == slot {
			cfg.colors[i].c = c
			return
		}
	}
	cfg.colors = append(cfg.colors, colorSet{slot, c})
}

// resetColors drops the program's changes to the slots match selects.
func (t *Terminal) resetColors(match func(uint16) bool) {
	if t.cfg == nil || len(t.cfg.colors) == 0 {
		return
	}
	kept := t.cfg.colors[:0]
	for _, s := range t.cfg.colors {
		if !match(s.i) {
			kept = append(kept, s)
		}
	}
	t.cfg.colors = kept
	t.touchAll()
}

// parseColor parses an X11 color specification in the forms programs
// send: rgb:R/G/B with 1-4 hex digits per component, and #RGB with 1-4
// digits per component. Color names are not supported.
func parseColor(s string) (RGB, bool) {
	var parts [3]string
	if rest, ok := strings.CutPrefix(s, "rgb:"); ok {
		p := strings.Split(rest, "/")
		if len(p) != 3 {
			return RGB{}, false
		}
		copy(parts[:], p)
		var c [3]uint8
		for i, h := range parts {
			v, err := strconv.ParseUint(h, 16, 16)
			if err != nil || len(h) == 0 || len(h) > 4 {
				return RGB{}, false
			}
			maxv := uint64(1)<<(4*len(h)) - 1
			c[i] = uint8((v*255 + maxv/2) / maxv)
		}
		return RGB{c[0], c[1], c[2]}, true
	}
	if rest, ok := strings.CutPrefix(s, "#"); ok && len(rest)%3 == 0 && len(rest) > 0 && len(rest) <= 12 {
		n := len(rest) / 3
		var c [3]uint8
		for i := range c {
			h := rest[i*n : (i+1)*n]
			v, err := strconv.ParseUint(h, 16, 16)
			if err != nil {
				return RGB{}, false
			}
			// The digits are the high bits of the component.
			c[i] = uint8(v << (16 - 4*n) >> 8)
		}
		return RGB{c[0], c[1], c[2]}, true
	}
	return RGB{}, false
}
