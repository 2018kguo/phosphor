package phosphor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"unicode/utf8"
	"unsafe"
)

// SnapshotLimits bounds the storage a RestoreSnapshot call may install.
// Limits do not change the snapshot's scrollback policy or bound later output.
type SnapshotLimits struct {
	// MaxBytes bounds requested decoded storage, including cell rows, slice
	// tables and strings. Zero selects 64 MiB. Allocator overhead is excluded.
	MaxBytes int
	// MaxCells bounds columns * rows for each screen, including sparse blank
	// cells that may be allocated by later output. Zero selects 4 Mi cells.
	MaxCells int
}

var (
	ErrSnapshotMalformed = errors.New("malformed terminal snapshot")
	ErrSnapshotVersion   = errors.New("unsupported terminal snapshot version")
	ErrSnapshotLimit     = errors.New("terminal snapshot exceeds limit")
)

const snapshotMagic = "PHOSNP\x00\x00"
const snapshotHeader = 24
const snapshotVersion = 2

var snapshotCRC = crc32.MakeTable(crc32.Castagnoli)

// Snapshot returns an exact checkpoint, including scrollback and unfinished
// input. Use Repaint to draw the terminal into another terminal. Snapshots do
// not contain reply writers or bell callbacks.
//
// Like Write and Resize, snapshot methods require exclusive access.
func (t *Terminal) Snapshot() []byte {
	// Estimate from owned history and drawn row prefixes. Sparse terminals
	// need no capacity proportional to their full cell geometry.
	n := 128 + 2*t.rows + t.sbBytes + len(t.title) + 3*len(t.kitty)
	for _, screen := range [][][]cell{t.main, t.alt} {
		for _, row := range screen {
			end := len(row)
			for end > 0 && row[end-1] == (cell{}) {
				end--
			}
			if end > 0 {
				n += 2*end + 16
			}
		}
	}
	if t.alt != nil {
		n += 2 * t.rows
	}
	if t.saved != nil {
		n += 128
	}
	if x := t.x; x != nil {
		for _, c := range x.clusters {
			n += len(c.text) + 8
		}
		for _, s := range x.rare {
			n += len(s.link) + 9
		}
		n += 10*len(x.tabs) + 5*len(x.savedModes)
	}
	if c := t.cfg; c != nil {
		if c.palette != nil {
			n += 259 * 3
		}
		n += 6 * len(c.colors)
	}
	if t.dirty != nil {
		n += 10 * len(t.dirty.bits)
	}
	if t.state >= stOSC {
		n += len(t.osc) + 4
	} else if t.state == stCSI {
		n += 7*int(t.nparams) + 4
	}
	return t.AppendSnapshot(make([]byte, 0, n))
}

// AppendSnapshot appends an exact checkpoint to dst, reusing its storage.
func (t *Terminal) AppendSnapshot(dst []byte) []byte {
	start := len(dst)
	dst = append(dst, snapshotMagic...)
	dst = append(dst, make([]byte, snapshotHeader-len(snapshotMagic))...)
	e := snapshotEncoder{b: dst}
	e.u(uint64(t.cols))
	e.u(uint64(t.rows))
	var flags uint64
	for i, on := range []bool{t.altActive, t.lrmm, t.protISO, t.autowrap, t.revWrap,
		t.revWrapExt, t.insert, t.newline, t.cursorHidden, t.reverseVideo, t.appCursor,
		t.appKeypad, t.bracketed, t.focus, t.sync, t.grapheme, t.sbOpen, t.sbSpacer} {
		if on {
			flags |= 1 << uint(i)
		}
	}
	e.u(flags)
	for _, n := range []int{t.top, t.bot, t.left, t.right, t.sbMaxLines, t.sbMaxBytes} {
		e.u(uint64(n))
	}
	e.u(uint64(t.ss))
	e.u(uint64(t.mouse))
	e.u(uint64(t.mouseEnc))
	e.u(uint64(t.cursorStyle.shape))
	e.boolean(t.cursorStyle.blink)
	e.u(uint64(t.lastChar))
	e.blob([]byte(t.title))
	e.u(uint64(len(t.kitty)))
	for _, n := range t.kitty {
		e.u(uint64(n))
	}

	var x extra
	if t.x != nil {
		x = *t.x
	}
	e.u(uint64(len(x.clusters)))
	for _, c := range x.clusters {
		e.blob([]byte(c.text))
		e.u(uint64(c.width))
		e.u(c.joinModes[0])
		e.u(c.joinModes[1])
	}
	e.u(uint64(len(x.rare)))
	for _, s := range x.rare {
		e.blob([]byte(s.link))
		e.u(uint64(s.underline))
	}
	e.u(uint64(len(x.tabs)))
	for _, bits := range x.tabs {
		e.u(bits)
	}
	e.u(uint64(len(x.savedModes)))
	for _, s := range x.savedModes {
		e.u(uint64(s.mode))
		e.boolean(s.on)
	}
	e.cursor(t.cur)
	var saves [3]savedCursor
	if t.saved != nil {
		saves = [3]savedCursor{t.saved.screen[0], t.saved.screen[1], t.saved.alternate}
	}
	var valid uint64
	for i, c := range saves {
		if c.valid {
			valid |= 1 << uint(i)
		}
	}
	e.u(valid)
	for _, c := range saves {
		if c.valid {
			e.cursor(c.cursor)
		}
	}
	e.screen(t.main, t.mainWrap)
	e.boolean(t.alt != nil)
	if t.alt != nil {
		e.screen(t.alt, t.altWrap)
	}
	e.u(uint64(t.ScrollbackLen()))
	for _, line := range t.sb[t.sbStart:] {
		e.blob(line)
	}
	var cfg config
	if t.cfg != nil {
		cfg = *t.cfg
	}
	e.boolean(cfg.palette != nil)
	if cfg.palette != nil {
		for _, c := range cfg.palette.Colors {
			e.rgb(c)
		}
		e.rgb(cfg.palette.Foreground)
		e.rgb(cfg.palette.Background)
		e.rgb(cfg.palette.Cursor)
	}
	e.u(uint64(len(cfg.colors)))
	for _, c := range cfg.colors {
		e.u(uint64(c.i))
		e.rgb(c.c)
	}
	e.boolean(t.dirty != nil)
	if t.dirty != nil {
		for _, bits := range t.dirty.bits {
			e.u(bits)
		}
	}
	e.blob(t.appendContinuation(nil))
	binary.LittleEndian.PutUint16(e.b[start+8:], snapshotVersion)
	binary.LittleEndian.PutUint64(e.b[start+12:], uint64(len(e.b)-start-snapshotHeader))
	binary.LittleEndian.PutUint32(e.b[start+20:], crc32.Checksum(e.b[start+snapshotHeader:], snapshotCRC))
	return e.b
}

type snapshotEncoder struct{ b []byte }

func (e *snapshotEncoder) u(n uint64) { e.b = binary.AppendUvarint(e.b, n) }
func (e *snapshotEncoder) boolean(on bool) {
	if on {
		e.u(1)
	} else {
		e.u(0)
	}
}
func (e *snapshotEncoder) blob(b []byte) { e.u(uint64(len(b))); e.b = append(e.b, b...) }
func (e *snapshotEncoder) rgb(c RGB)     { e.b = append(e.b, c.R, c.G, c.B) }
func (e *snapshotEncoder) style(c cell) {
	e.u(uint64(c.Fg))
	e.u(uint64(c.Bg))
	e.u(uint64(c.A))
	e.u(uint64(c.E))
}
func (e *snapshotEncoder) cursor(c cursor) {
	e.u(uint64(c.x))
	e.u(uint64(c.y))
	e.style(c.pen)
	var flags uint64
	for i, on := range []bool{c.wrapPending, c.origin, c.g[0], c.g[1], c.g[2], c.g[3]} {
		if on {
			flags |= 1 << uint(i)
		}
	}
	e.u(flags)
	e.u(uint64(c.gl))
}
func (e *snapshotEncoder) screen(screen [][]cell, wraps []bool) {
	for y, row := range screen {
		e.boolean(wraps != nil && wraps[y])
		n := len(row)
		for n > 0 && row[n-1] == (cell{}) {
			n--
		}
		e.u(uint64(n))
		for i := 0; i < n; {
			end := i + 1
			style := styleOf(row[i])
			for end < n && styleOf(row[end]) == style {
				end++
			}
			e.u(uint64(end - i))
			e.style(style)
			for j := i; j < end; {
				e.u(uint64(int64(row[j].R) + 2))
				if row[j].R == 0 {
					last := j + 1
					for last < end && row[last].R == 0 {
						last++
					}
					e.u(uint64(last - j))
					j = last
				} else {
					j++
				}
			}
			i = end
		}
	}
}

// RestoreSnapshot replaces durable state and retains only the receiver's reply
// writer and bell callback. It also works on a zero Terminal. Constructor
// geometry, palette and scrollback choices are replaced by snapshot state.
//
// No callbacks run, no input storage is retained, and an error leaves the
// receiver unchanged. Negative limits are invalid. The decoded-storage bound
// excludes fixed parser-validation scratch state and a buffer of at most
// maxOSC+4 bytes.
func (t *Terminal) RestoreSnapshot(data []byte, limits SnapshotLimits) error {
	if limits.MaxBytes < 0 || limits.MaxCells < 0 {
		return fmt.Errorf("%w: negative restore limit", ErrSnapshotLimit)
	}
	if limits.MaxBytes == 0 {
		limits.MaxBytes = 64 << 20
	}
	if limits.MaxCells == 0 {
		limits.MaxCells = 4 << 20
	}
	if len(data) < snapshotHeader || string(data[:8]) != snapshotMagic {
		return fmt.Errorf("%w: header", ErrSnapshotMalformed)
	}
	version := binary.LittleEndian.Uint16(data[8:])
	if version != 1 && version != snapshotVersion {
		return ErrSnapshotVersion
	}
	if binary.LittleEndian.Uint16(data[10:]) != 0 || binary.LittleEndian.Uint64(data[12:]) != uint64(len(data)-snapshotHeader) {
		return fmt.Errorf("%w: envelope", ErrSnapshotMalformed)
	}
	payload := data[snapshotHeader:]
	if binary.LittleEndian.Uint32(data[20:]) != crc32.Checksum(payload, snapshotCRC) {
		return fmt.Errorf("%w: checksum", ErrSnapshotMalformed)
	}
	d := snapshotDecoder{b: payload, remaining: limits.MaxBytes, section: "terminal"}
	d.charge(1, int(unsafe.Sizeof(Terminal{})))
	var next Terminal
	next.cols, next.rows = d.integer(), d.integer()
	if next.cols < 1 || next.rows < 1 {
		d.fail(ErrSnapshotMalformed, "empty geometry")
	} else if next.rows > limits.MaxCells/next.cols {
		d.fail(ErrSnapshotLimit, "screen cells")
	}
	flags := d.number((1 << 18) - 1)
	for i, p := range []*bool{&next.altActive, &next.lrmm, &next.protISO, &next.autowrap, &next.revWrap,
		&next.revWrapExt, &next.insert, &next.newline, &next.cursorHidden, &next.reverseVideo, &next.appCursor,
		&next.appKeypad, &next.bracketed, &next.focus, &next.sync, &next.grapheme, &next.sbOpen, &next.sbSpacer} {
		*p = flags&(1<<uint(i)) != 0
	}
	next.top, next.bot, next.left, next.right = d.integer(), d.integer(), d.integer(), d.integer()
	next.sbMaxLines, next.sbMaxBytes = d.integer(), d.integer()
	if next.top > next.bot || next.bot >= next.rows || next.left > next.right || next.right >= next.cols ||
		!next.lrmm && (next.left != 0 || next.right != next.cols-1) || next.sbMaxBytes < 1 {
		d.fail(ErrSnapshotMalformed, "margins or scrollback policy")
	}
	next.ss = uint8(d.number(3))
	if next.ss == 1 {
		d.fail(ErrSnapshotMalformed, "single shift")
	}
	if version == 1 {
		tracking, encoding := int(d.number(1015)), int(d.number(1015))
		if tracking != 0 && mouseModeBit(tracking, mouseTrackingNumbers) == 0 ||
			encoding != 0 && (encoding == 1016 || mouseModeBit(encoding, mouseEncodingNumbers) == 0) {
			d.fail(ErrSnapshotMalformed, "legacy mouse mode")
		}
		// V1 remembered only the effective mode. Other enabled bits were
		// already lost by that release, so preserve the single known bit.
		next.mouse = setMouseMode(0, tracking, true, mouseTrackingNumbers)
		next.mouseEnc = setMouseMode(0, encoding, true, mouseEncodingNumbers)
	} else {
		next.mouse, next.mouseEnc = uint16(d.number(0x4f)), uint16(d.number(0x4f))
		if !validMouseModes(next.mouse) || !validMouseModes(next.mouseEnc) {
			d.fail(ErrSnapshotMalformed, "mouse modes")
		}
	}
	next.cursorStyle = cursorAppearance{shape: CursorShape(d.number(uint64(CursorBar))), blink: d.boolean()}
	next.lastChar = rune(d.number(utf8.MaxRune))
	if next.lastChar != 0 && !snapshotScalar(next.lastChar) {
		d.fail(ErrSnapshotMalformed, "REP character")
	}
	next.title = d.text(maxOSC)
	if !snapshotOSCText(next.title) {
		d.fail(ErrSnapshotMalformed, "title terminator")
	}
	n := d.count(16, int(unsafe.Sizeof(uint16(0))))
	if n > 0 {
		next.kitty = make([]uint16, n)
		for i := range next.kitty {
			next.kitty[i] = uint16(d.number(0xffff))
		}
	}
	d.section = "lookup tables"
	var x extra
	n = d.count(int(^uint32(0)>>1)-int(clusterBase)+1, int(unsafe.Sizeof(cluster{})))
	if n > 0 {
		x.clusters = make([]cluster, n)
		for i := range x.clusters {
			bound := maxCluster
			if version == 1 {
				bound = 32
			}
			c := cluster{text: d.text(bound), width: uint8(d.number(2))}
			if version == 1 {
				// Released v1 indexed mode bits by UTF-8 byte offset. Reject
				// non-boundaries before converting to codepoint indices.
				old := uint32(d.number(0xffffffff))
				index := 0
				for offset := range c.text {
					if offset != 0 && old&(1<<uint(offset)) != 0 {
						c.joinModes.set(index)
						old &^= 1 << uint(offset)
					}
					index++
				}
				if old != 0 {
					d.fail(ErrSnapshotMalformed, "cluster byte boundary")
				}
			} else {
				c.joinModes = clusterModes{d.u(), d.number(1)}
			}
			if !snapshotCluster(c) {
				d.fail(ErrSnapshotMalformed, "cluster")
			}
			x.clusters[i] = c
		}
		x.clusterCap = max(64, len(x.clusters)*2)
	}
	n = d.count(0xffff, int(unsafe.Sizeof(cellExtra{})))
	if n > 0 {
		x.rare = make([]cellExtra, n)
		for i := range x.rare {
			x.rare[i].link = d.text(maxOSC)
			if !snapshotOSCText(x.rare[i].link) {
				d.fail(ErrSnapshotMalformed, "hyperlink terminator")
			}
			if version >= 2 {
				x.rare[i].underline = Color(d.number(0xffffffff))
				if !snapshotColor(x.rare[i].underline) {
					d.fail(ErrSnapshotMalformed, "underline color")
				}
			}
		}
		x.rareCap = min(0xffff, max(16, len(x.rare)*2))
	}
	n = d.count((next.cols+63)/64, 8)
	if n > 0 {
		if n != (next.cols+63)/64 {
			d.fail(ErrSnapshotMalformed, "tab table length")
		}
		x.tabs = make([]uint64, n)
		for i := range x.tabs {
			x.tabs[i] = d.u()
		}
	}
	n = d.count(16, int(unsafe.Sizeof(savedMode{})))
	if n > 0 {
		x.savedModes = make([]savedMode, n)
		for i := range x.savedModes {
			s := savedMode{mode: d.integer(), on: d.boolean()}
			if _, ok := next.modeValue(s.mode); !ok {
				d.fail(ErrSnapshotMalformed, "saved mode")
			}
			for _, prior := range x.savedModes[:i] {
				if prior.mode == s.mode {
					d.fail(ErrSnapshotMalformed, "duplicate saved mode")
				}
			}
			x.savedModes[i] = s
		}
	}
	if x.clusters != nil || x.rare != nil || x.tabs != nil || x.savedModes != nil {
		if d.charge(1, int(unsafe.Sizeof(extra{}))) {
			next.x = new(extra)
			*next.x = x
		}
	}
	d.section = "cursors"
	next.cur = d.cursor(&next)
	valid := d.number(7)
	if valid != 0 && d.charge(1, int(unsafe.Sizeof(savedCursors{}))) {
		next.saved = &savedCursors{}
		for i, c := range []*savedCursor{&next.saved.screen[0], &next.saved.screen[1], &next.saved.alternate} {
			if valid&(1<<uint(i)) != 0 {
				*c = savedCursor{cursor: d.cursor(&next), valid: true}
			}
		}
	}
	d.section = "screens"
	next.main, next.mainWrap = d.screen(&next)
	if d.boolean() {
		next.alt, next.altWrap = d.screen(&next)
	}
	if next.altActive && next.alt == nil {
		d.fail(ErrSnapshotMalformed, "active alternate screen missing")
	}
	d.section = "scrollback"
	n = d.count(int(^uint(0)>>1), 24)
	if n > 0 {
		next.sb = make([][]byte, n)
		for i := range next.sb {
			b := d.blob(int(^uint(0) >> 1))
			if d.err == nil && !walkHistoryText(b, nil) {
				d.fail(ErrSnapshotMalformed, "history formatter syntax")
			}
			if d.err != nil {
				break
			}
			next.sb[i] = bytes.Clone(b)
			next.sbBytes += len(b) + 24
		}
	}
	d.section = "palette"
	var cfg config
	if d.boolean() && d.charge(1, int(unsafe.Sizeof(Palette{}))) {
		cfg.palette = &Palette{}
		for i := range cfg.palette.Colors {
			cfg.palette.Colors[i] = d.rgb()
		}
		cfg.palette.Foreground, cfg.palette.Background, cfg.palette.Cursor = d.rgb(), d.rgb(), d.rgb()
	}
	n = d.count(259, int(unsafe.Sizeof(colorSet{})))
	if n > 0 {
		cfg.colors = make([]colorSet, n)
		for i := range cfg.colors {
			c := colorSet{i: uint16(d.number(slotCursor)), c: d.rgb()}
			for _, prior := range cfg.colors[:i] {
				if prior.i == c.i {
					d.fail(ErrSnapshotMalformed, "duplicate override")
				}
			}
			cfg.colors[i] = c
		}
	}
	if t.cfg != nil {
		cfg.reply, cfg.bell = t.cfg.reply, t.cfg.bell
	}
	if cfg.palette != nil || cfg.colors != nil || cfg.reply != nil || cfg.bell != nil {
		d.charge(1, int(unsafe.Sizeof(config{})))
		cfg.sbLines, cfg.sbBytes = next.sbMaxLines, next.sbMaxBytes
	}
	d.section = "damage"
	if d.boolean() && d.charge(1, int(unsafe.Sizeof(dirtySet{}))) && d.charge((next.rows+63)/64, 8) {
		next.dirty = &dirtySet{bits: make([]uint64, (next.rows+63)/64)}
		for i := range next.dirty.bits {
			next.dirty.bits[i] = d.u()
		}
	}
	d.section = "parser continuation"
	prefix := d.blob(maxOSC + 4)
	if d.err == nil && !snapshotContinuation(prefix, &next) {
		d.fail(ErrSnapshotMalformed, "noncanonical or completed input")
	}
	if d.err != nil {
		return d.err
	}
	if len(d.b) != 0 {
		return fmt.Errorf("%w: trailing payload", ErrSnapshotMalformed)
	}
	if cfg.palette != nil || cfg.colors != nil || cfg.reply != nil || cfg.bell != nil {
		next.cfg = new(config)
		*next.cfg = cfg
	}
	*t = next
	return nil
}

type snapshotDecoder struct {
	b         []byte
	remaining int
	section   string
	err       error
}

func (d *snapshotDecoder) fail(kind error, reason string) {
	if d.err == nil {
		d.err = fmt.Errorf("%w: %s: %s", kind, d.section, reason)
	}
}
func (d *snapshotDecoder) u() uint64 {
	if d.err != nil {
		return 0
	}
	n, size := binary.Uvarint(d.b)
	if size <= 0 {
		d.fail(ErrSnapshotMalformed, "integer")
		return 0
	}
	d.b = d.b[size:]
	return n
}
func (d *snapshotDecoder) number(maximum uint64) uint64 {
	n := d.u()
	if n > maximum {
		d.fail(ErrSnapshotMalformed, "integer range")
		return 0
	}
	return n
}
func (d *snapshotDecoder) integer() int  { return int(d.number(uint64(^uint(0) >> 1))) }
func (d *snapshotDecoder) boolean() bool { return d.number(1) != 0 }
func (d *snapshotDecoder) charge(n, size int) bool {
	if d.err != nil {
		return false
	}
	if n < 0 || size < 1 || n > d.remaining/size {
		d.fail(ErrSnapshotLimit, "decoded storage")
		return false
	}
	d.remaining -= n * size
	return true
}
func (d *snapshotDecoder) count(maximum, size int) int {
	n := d.integer()
	if n > maximum {
		d.fail(ErrSnapshotMalformed, "count")
		return 0
	}
	if !d.charge(n, size) {
		return 0
	}
	return n
}
func (d *snapshotDecoder) take(n int) []byte {
	if d.err != nil {
		return nil
	}
	if n < 0 || n > len(d.b) {
		d.fail(ErrSnapshotMalformed, "truncated data")
		return nil
	}
	b := d.b[:n]
	d.b = d.b[n:]
	return b
}
func (d *snapshotDecoder) blob(maximum int) []byte {
	n := d.count(maximum, 1)
	return d.take(n)
}
func (d *snapshotDecoder) text(maximum int) string { return string(d.blob(maximum)) }
func (d *snapshotDecoder) rgb() RGB {
	b := d.take(3)
	if len(b) != 3 {
		return RGB{}
	}
	return RGB{b[0], b[1], b[2]}
}
func (d *snapshotDecoder) style(t *Terminal) cell {
	c := cell{Fg: Color(d.number(0xffffffff)), Bg: Color(d.number(0xffffffff)), A: Attr(d.number(0xffff)), E: uint16(d.number(0xffff))}
	if !snapshotColor(c.Fg) || !snapshotColor(c.Bg) || c.A & ^(AttrBold|AttrFaint|AttrItalic|AttrBlink|AttrInverse|AttrHidden|AttrStrikethrough|AttrOverline|AttrProtected|ulMask) != 0 || (c.A&ulMask)>>ulShift > Attr(UnderlineDashed) {
		d.fail(ErrSnapshotMalformed, "cell style")
	}
	if c.E != 0 && (t.x == nil || int(c.E) > len(t.x.rare)) {
		d.fail(ErrSnapshotMalformed, "cell attribute reference")
	}
	return c
}

func snapshotColor(c Color) bool {
	return c == 0 || c>>24 == 1 && c&0x00ffff00 == 0 || c>>24 == 2
}
func (d *snapshotDecoder) cursor(t *Terminal) cursor {
	c := cursor{x: d.integer(), y: d.integer(), pen: d.style(t)}
	flags := d.number(63)
	c.wrapPending, c.origin = flags&1 != 0, flags&2 != 0
	for i := range c.g {
		c.g[i] = flags&(1<<uint(i+2)) != 0
	}
	c.gl = uint8(d.number(3))
	if c.x >= t.cols || c.y >= t.rows {
		d.fail(ErrSnapshotMalformed, "cursor position")
	}
	return c
}
func (d *snapshotDecoder) screen(t *Terminal) ([][]cell, []bool) {
	if !d.charge(t.rows, int(unsafe.Sizeof([]cell{}))) {
		return nil, nil
	}
	screen := make([][]cell, t.rows)
	var wraps []bool
	for y := range screen {
		if d.err != nil {
			break
		}
		if d.boolean() {
			if y == t.rows-1 {
				d.fail(ErrSnapshotMalformed, "last row wraps")
			}
			if wraps == nil && d.charge(t.rows, 1) {
				wraps = make([]bool, t.rows)
			}
			if wraps != nil {
				wraps[y] = true
			}
		}
		n := d.integer()
		if n > t.cols {
			d.fail(ErrSnapshotMalformed, "row length")
		}
		if n == 0 || !d.charge(t.cols, int(unsafe.Sizeof(cell{}))) {
			continue
		}
		row := make([]cell, t.cols)
		screen[y] = row
		wantTail := false
		for i := 0; i < n && d.err == nil; {
			run := d.integer()
			if run < 1 || run > n-i {
				d.fail(ErrSnapshotMalformed, "style run")
				break
			}
			style := d.style(t)
			for end := i + run; i < end; {
				c := style
				c.R = rune(int64(d.number(uint64(^uint32(0)>>1)+2)) - 2)
				if c.R == 0 {
					zeros := d.integer()
					if zeros < 1 || zeros > end-i || wantTail {
						d.fail(ErrSnapshotMalformed, "blank cell run")
						break
					}
					if style != (cell{}) {
						for j := i; j < i+zeros; j++ {
							row[j] = style
						}
					}
					i += zeros
					continue
				}
				if c.R >= clusterBase {
					if t.x == nil || int(c.R-clusterBase) >= len(t.x.clusters) {
						d.fail(ErrSnapshotMalformed, "cluster reference")
					}
				} else if c.R != 0 && c.R != wideTail && c.R != spacerHead && !snapshotScalar(c.R) {
					d.fail(ErrSnapshotMalformed, "cell character")
				}
				if d.err == nil {
					if (c.R == wideTail) != wantTail {
						d.fail(ErrSnapshotMalformed, "wide cell geometry")
					}
					wantTail = c.R >= utf8.RuneSelf && t.runeWidth(c.R) == 2
				}
				row[i] = c
				i++
			}
		}
		if wantTail {
			d.fail(ErrSnapshotMalformed, "wide cell geometry")
		}
	}
	return screen, wraps
}

func snapshotScalar(r rune) bool {
	if r >= 0x20 && r < 0x7f {
		return true
	}
	return utf8.ValidRune(r) && r >= 0x20 && r != 0x7f && (r < 0x80 || r >= 0xa0) && CodepointWidth(r) > 0
}
func snapshotCluster(c cluster) bool {
	if len(c.text) == 0 || !utf8.ValidString(c.text) || c.width < 1 || c.width > 2 || !snapshotScalar(firstRune(c.text)) {
		return false
	}
	var starts clusterModes
	index := 0
	for _, r := range c.text {
		if index >= maxClusterRunes {
			return false
		}
		if r < 0x20 || r >= 0x7f && r < 0xa0 {
			return false
		}
		if index != 0 {
			starts.set(index)
		}
		index++
	}
	return c.joinModes[0] & ^starts[0] == 0 && c.joinModes[1] & ^starts[1] == 0
}

func snapshotOSCText(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 0x07, 0x18, 0x1a, 0x1b:
			return false
		}
	}
	return true
}

// Validate before feeding each byte: no complete command or scalar is ever
// run, even on the scratch terminal. Canonical re-encoding then rules out
// redundant prefixes and discarded controls that already took effect.
func snapshotContinuation(prefix []byte, target *Terminal) bool {
	if len(prefix) == 0 {
		return true
	}
	if len(prefix) > maxOSC+4 {
		return false
	}
	probe := Terminal{cols: 1, rows: 1}
	if len(prefix) >= 2 && prefix[0] == 0x1b && (prefix[1] == ']' || prefix[1] == 'P') {
		probe.osc = make([]byte, 0, len(prefix)-2)
	}
	for i, c := range prefix {
		allowed := false
		if probe.utf8n != 0 {
			var b [utf8.UTFMax]byte
			copy(b[:], probe.utf8buf[:probe.utf8n])
			b[probe.utf8n] = c
			allowed = c >= 0x80 && c <= 0xbf && !utf8.FullRune(b[:int(probe.utf8n)+1])
		} else {
			switch probe.state {
			case stGround:
				allowed = c == 0x1b || c >= 0xc2 && c <= 0xf4
			case stEsc:
				allowed = c >= 0x20 && c <= 0x2f || c == '[' || c == ']' || c == 'P' || c == 'X'
			case stEscInter:
				allowed = c >= 0x20 && c <= 0x2f
			case stCSI:
				allowed = c >= 0x20 && c <= 0x3f
			case stCSIIgnore:
				allowed = c == '?'
			case stOSC:
				allowed = c != 0x07 && c != 0x18 && c != 0x1a && c != 0x1b
			case stDCS, stStr:
				allowed = c != 0x18 && c != 0x1a && c != 0x1b
			}
		}
		if !allowed {
			return false
		}
		probe.Write(prefix[i : i+1])
	}
	var canonical [maxOSC + 4]byte
	if !bytes.Equal(prefix, probe.appendContinuation(canonical[:0])) {
		return false
	}
	target.params, target.sub, target.nparams = probe.params, probe.sub, probe.nparams
	target.state, target.paramSet, target.marker, target.inter = probe.state, probe.paramSet, probe.marker, probe.inter
	target.utf8buf, target.utf8n = probe.utf8buf, probe.utf8n
	target.osc, target.oscOver = probe.osc, probe.oscOver
	return true
}
