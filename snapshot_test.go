package phosphor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func assertSnapshotState(t *testing.T, a, b *Terminal) {
	t.Helper()
	ac, ar := a.Size()
	bc, br := b.Size()
	if ac != bc || ar != br || a.Title() != b.Title() || a.Cursor() != b.Cursor() || a.Modes() != b.Modes() || a.Palette() != b.Palette() {
		t.Fatalf("terminal observations changed: size %dx%d/%dx%d cursor %+v/%+v modes %+v/%+v", ac, ar, bc, br, a.Cursor(), b.Cursor(), a.Modes(), b.Modes())
	}
	if !snapshotCursorsEqual(a, b, a.cur, b.cur) || a.lastChar != b.lastChar || a.top != b.top || a.bot != b.bot || a.left != b.left || a.right != b.right ||
		a.lrmm != b.lrmm || a.protISO != b.protISO || a.ss != b.ss || a.revWrap != b.revWrap || a.revWrapExt != b.revWrapExt ||
		a.grapheme != b.grapheme || a.sbOpen != b.sbOpen || a.sbSpacer != b.sbSpacer || a.sbMaxLines != b.sbMaxLines || a.sbMaxBytes != b.sbMaxBytes ||
		!slices.Equal(a.kitty, b.kitty) || !bytes.Equal(a.appendContinuation(nil), b.appendContinuation(nil)) {
		t.Fatal("future-output state changed")
	}
	for i := range 3 {
		var left, right savedCursor
		if a.saved != nil {
			left = [3]savedCursor{a.saved.screen[0], a.saved.screen[1], a.saved.alternate}[i]
		}
		if b.saved != nil {
			right = [3]savedCursor{b.saved.screen[0], b.saved.screen[1], b.saved.alternate}[i]
		}
		if left.valid != right.valid || left.valid && !snapshotCursorsEqual(a, b, left.cursor, right.cursor) {
			t.Fatalf("saved cursor %d changed", i)
		}
	}
	for _, s := range []struct {
		name   string
		a, b   [][]cell
		aw, bw []bool
	}{{"main", a.main, b.main, a.mainWrap, b.mainWrap}, {"alternate", a.alt, b.alt, a.altWrap, b.altWrap}} {
		if len(s.a) != len(s.b) {
			t.Fatalf("%s presence changed", s.name)
		}
		for y := range s.a {
			for x := range ac {
				left, right := Cell{Width: 1}, Cell{Width: 1}
				if s.a[y] != nil {
					left = a.publicCell(s.a[y], x)
				}
				if s.b[y] != nil {
					right = b.publicCell(s.b[y], x)
				}
				if left != right {
					t.Fatalf("%s cell %d,%d: %+v / %+v", s.name, x, y, left, right)
				}
			}
			if (s.aw != nil && s.aw[y]) != (s.bw != nil && s.bw[y]) {
				t.Fatalf("%s row %d wrap changed", s.name, y)
			}
		}
	}
	if a.ScrollbackLen() != b.ScrollbackLen() || a.sbBytes != b.sbBytes {
		t.Fatal("history length or byte accounting changed")
	}
	for i := range a.ScrollbackLen() {
		if !bytes.Equal(a.sb[a.sbStart+i], b.sb[b.sbStart+i]) || a.ScrollbackLine(i) != b.ScrollbackLine(i) {
			t.Fatalf("history line %d changed", i)
		}
	}
	for y := range ar {
		if a.RowDirty(y) != b.RowDirty(y) {
			t.Fatalf("row %d damage changed", y)
		}
	}
}

func snapshotCursorsEqual(a, b *Terminal, left, right cursor) bool {
	// Pool collection can renumber IDs at different times after restore.
	// Compare the referenced targets while keeping every cursor flag exact.
	leftExtra, rightExtra := a.extraAt(left.pen.E), b.extraAt(right.pen.E)
	left.pen.E, right.pen.E = 0, 0
	return left == right && leftExtra == rightExtra
}

func snapshotScene() string {
	return strings.Repeat("\x1b[32mhistory 漢 é\x1b[0m\r\n", 20) +
		"\x1b[3g\x1b[3G\x1bH\x1b[?69h\x1b[2;10s\x1b[2;4r\x1b[?6h" +
		"\x1b]8;id=main;https://main\x1b\\\x1b[31;4:3m\x1b(0\x1bn\x1b7" +
		"\x1b[?1;5;7;12;45;69;1045;2004s\x1b[?1;5;12;45;1045;2004h\x1b[3 q\x1b[>7u\x1b[>13u" +
		"\x1b[?1049h\x1b[?6l\x1b[H\x1b]8;id=alt;https://alt\x1b\\ALT á\x1b[35m\x1b7" +
		"\x1b]4;1;#123456\a\x1b]10;#234567\a\x1b]2;raw title\xff\a" +
		"\x1b[?2027h❤️\x1b[?2027l\x1b[?2004l\x1b[2;4r\x1b[?6h\x1b[?6l\x1b[1;1H\x1b[?6h\x1b[1A"
}

func TestSnapshotFutureState(t *testing.T) {
	var aReply, bReply bytes.Buffer
	aBells, bBells := 0, 0
	palette := DefaultPalette()
	palette.Colors[1] = RGB{11, 12, 13}
	a := New(12, 5, WithPalette(palette), WithScrollback(32), WithScrollbackBytes(4096), WithReply(&aReply), WithBell(func() { aBells++ }))
	a.Write([]byte(snapshotScene()))
	a.ClearDirty()
	a.Write([]byte("X"))
	b := New(1, 1, WithReply(&bReply), WithBell(func() { bBells++ }))
	if err := b.RestoreSnapshot(a.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	if bReply.Len() != 0 || bBells != 0 {
		t.Fatal("restore invoked runtime callbacks")
	}
	assertSnapshotState(t, a, b)
	aReply.Reset()
	aBells = 0
	for _, suffix := range []string{
		"\x1b[2b\x1b8X\x1b[?1049lY\x1b8Z\x1b[?47hkeep\x1b[?47l",
		"\x1b[?1;5;7;12;45;69;1045;2004r\tT\x1b[<1u\x1b[?u",
		"\x1b]104;1\a\x1b]110\a\x1b]4;1;?\a\x1b]10;?\a\x1b[6n\x1bP$q q\x1b\\\a",
		"\x1b[?2027h👩‍👩‍👧‍👦\x1b[?2027l é\r\nnext\x1b[3b",
	} {
		a.Write([]byte(suffix))
		b.Write([]byte(suffix))
		assertSnapshotState(t, a, b)
		if !bytes.Equal(aReply.Bytes(), bReply.Bytes()) || aBells != bBells {
			t.Fatalf("future callbacks differ: %q / %q, bells %d / %d", aReply.Bytes(), bReply.Bytes(), aBells, bBells)
		}
	}
	for _, size := range [][2]int{{2, 3}, {1, 4}, {20, 12}, {12, 5}} {
		a.Resize(size[0], size[1])
		b.Resize(size[0], size[1])
		b.Compact()
		assertSnapshotState(t, a, b)
		a.Write([]byte("\x1b[2b\r\n"))
		b.Write([]byte("\x1b[2b\r\n"))
		assertSnapshotState(t, a, b)
	}
}

func TestSnapshotContinuesAcrossPoolCollection(t *testing.T) {
	a := New(4, 2)
	writeLink := func(term *Terminal, i int) {
		term.Write(fmt.Appendf(nil, "\x1b[H\x1b]8;id=%d;https://example.com/%d\x1b\\X", i, i))
	}
	for i := range 30 {
		writeLink(a, i)
	}
	var b Terminal
	if err := b.RestoreSnapshot(a.Snapshot(), SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	for i := 30; i < 80; i++ {
		writeLink(a, i)
		writeLink(&b, i)
		assertSnapshotState(t, a, &b)
	}
}

func TestSnapshotContinuation(t *testing.T) {
	for _, stream := range []string{
		"abc\x1b[3\b\a1mX\x1b[?1?2004hY\x1b[38:2::10:20:30mZ",
		"\x1b[" + strings.Repeat("999999999999999999;", maxParams+1) + "31mX",
		"\x1b]2;title\x1b\\X\x1b]2;cancel\x18Y\x1b]8;id=a;https://example.com\x1b\\L",
		"\x1b]2;" + strings.Repeat("x", maxOSC+1) + "\aX",
		"\x1bP$qm\x1b\\X\x1bP+q544e\x1b\\Y\x1bPqignored\x1b\\Z\x1b_ignore\x1b\\",
		"😀\xf0\x9f\x1b[31m漢\x1b(0lqk\x1b(BX",
	} {
		for cut := 0; cut <= len(stream); cut++ {
			var aReply, bReply bytes.Buffer
			aBells, bBells := 0, 0
			a := New(12, 5, WithReply(&aReply), WithBell(func() { aBells++ }))
			a.Write([]byte(stream[:cut]))
			b := New(1, 1, WithReply(&bReply), WithBell(func() { bBells++ }))
			if err := b.RestoreSnapshot(a.Snapshot(), SnapshotLimits{}); err != nil {
				t.Fatalf("cut %d of %q: %v", cut, stream[:min(len(stream), 48)], err)
			}
			assertSnapshotState(t, a, b)
			if bReply.Len() != 0 || bBells != 0 {
				t.Fatal("restore repeated callbacks")
			}
			aReply.Reset()
			aBells = 0
			a.Write([]byte(stream[cut:]))
			b.Write([]byte(stream[cut:]))
			assertSnapshotState(t, a, b)
			if !bytes.Equal(aReply.Bytes(), bReply.Bytes()) || aBells != bBells {
				t.Fatalf("cut %d: callbacks differ", cut)
			}
		}
	}
}

func TestSnapshotNarrowedGrapheme(t *testing.T) {
	a := New(5, 2)
	a.Write([]byte("\x1b[?2027h😀︎\x1b[?2027l"))
	for _, width := range []int{1, 2, 5} {
		a.Resize(width, 2)
		var b Terminal
		if err := b.RestoreSnapshot(a.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatal(err)
		}
		assertSnapshotState(t, a, &b)
		for _, suffix := range []string{"\x1b[2b", "\x1b[?2027h️\x1b[?2027l"} {
			a.Write([]byte(suffix))
			b.Write([]byte(suffix))
			assertSnapshotState(t, a, &b)
		}
	}
}

func sealSnapshot(data []byte) []byte {
	binary.LittleEndian.PutUint64(data[12:], uint64(len(data)-snapshotHeader))
	binary.LittleEndian.PutUint32(data[20:], crc32.Checksum(data[snapshotHeader:], snapshotCRC))
	return data
}

func TestSnapshotMalformedAtomic(t *testing.T) {
	receiver := New(7, 3)
	receiver.Write([]byte("keep me\x1b[31m\x1b[?2004h\x1b[38:"))
	before := receiver.Snapshot()
	valid := New(4, 2).Snapshot()
	check := func(name string, data []byte, limits SnapshotLimits, want error) {
		t.Helper()
		err := receiver.RestoreSnapshot(data, limits)
		if !errors.Is(err, want) {
			t.Fatalf("%s: error %v, want %v", name, err, want)
		}
		if !bytes.Equal(before, receiver.Snapshot()) {
			t.Fatalf("%s: failed restore changed receiver", name)
		}
	}
	for cut := range len(valid) {
		check(fmt.Sprintf("truncation-%d", cut), valid[:cut], SnapshotLimits{}, ErrSnapshotMalformed)
		if cut >= snapshotHeader {
			check(fmt.Sprintf("payload-truncation-%d", cut), sealSnapshot(bytes.Clone(valid[:cut])), SnapshotLimits{}, ErrSnapshotMalformed)
		}
	}
	for i := range valid {
		bad := bytes.Clone(valid)
		bad[i] ^= 0x80
		want := ErrSnapshotMalformed
		if i == 8 || i == 9 {
			want = ErrSnapshotVersion
		}
		check(fmt.Sprintf("corruption-%d", i), bad, SnapshotLimits{}, want)
	}
	check("trailing", append(bytes.Clone(valid), 0), SnapshotLimits{}, ErrSnapshotMalformed)
	check("storage", valid, SnapshotLimits{MaxBytes: 1}, ErrSnapshotLimit)
	check("geometry", valid, SnapshotLimits{MaxCells: 7}, ErrSnapshotLimit)
	check("negative", valid, SnapshotLimits{MaxCells: -1}, ErrSnapshotLimit)
	for name, mutate := range map[string]func(*Terminal){
		"cursor":            func(t *Terminal) { t.cur.x = t.cols },
		"margin":            func(t *Terminal) { t.bot = t.rows },
		"REP":               func(t *Terminal) { t.lastChar = clusterBase },
		"alternate":         func(t *Terminal) { t.altActive = true },
		"tail":              func(t *Terminal) { t.row(0)[0].R = wideTail },
		"wide-head":         func(t *Terminal) { t.row(0)[0].R = '漢' },
		"cell-scalar":       func(t *Terminal) { t.row(0)[0].R = 0xd800 },
		"cluster-reference": func(t *Terminal) { t.row(0)[0].R = clusterBase },
		"link-reference":    func(t *Terminal) { t.cur.pen.E = 1 },
		"color":             func(t *Terminal) { t.cur.pen.Fg = Color(0x1000100) },
		"attribute":         func(t *Terminal) { t.cur.pen.A = 1 << 10 },
		"tabs":              func(t *Terminal) { t.ext().tabs = []uint64{0, 0} },
		"saved-mode":        func(t *Terminal) { t.ext().savedModes = []savedMode{{mode: 999}} },
		"duplicate-mode":    func(t *Terminal) { t.ext().savedModes = []savedMode{{mode: 7}, {mode: 7}} },
		"cluster-text":      func(t *Terminal) { t.ext().clusters = []cluster{{text: "\xff", width: 1}} },
		"cluster-control":   func(t *Terminal) { t.ext().clusters = []cluster{{text: "a\x1b[cXY", width: 1}} },
		"cluster-C1":        func(t *Terminal) { t.ext().clusters = []cluster{{text: "a\u009b", width: 1}} },
		"cluster-width":     func(t *Terminal) { t.ext().clusters = []cluster{{text: "á", width: 0}} },
		"cluster-join":      func(t *Terminal) { t.ext().clusters = []cluster{{text: "á", width: 1, joinModes: 1 << 2}} },
		"wrap":              func(t *Terminal) { t.mainWrap = []bool{false, true} },
		"palette-slot":      func(t *Terminal) { t.cfg = &config{colors: []colorSet{{i: 259}}} },
		"palette-duplicate": func(t *Terminal) { t.cfg = &config{colors: []colorSet{{i: 1}, {i: 1}}} },
		"title-command":     func(t *Terminal) { t.title = "title\x1b[c" },
		"link-command":      func(t *Terminal) { t.ext().rare = []cellExtra{{link: ";https://example.com\a"}} },
		"underline-color":   func(t *Terminal) { t.ext().rare = []cellExtra{{underline: Color(3 << 24)}} },
		"history-query":     func(t *Terminal) { t.sb = [][]byte{[]byte("\x1b[cX")}; t.sbBytes = 28 },
		"history-bell":      func(t *Terminal) { t.sb = [][]byte{[]byte("hello\a")}; t.sbBytes = 30 },
		"history-OSC":       func(t *Terminal) { t.sb = [][]byte{[]byte("\x1b]52;;clipboard\x1b\\")}; t.sbBytes = 42 },
	} {
		bad := New(4, 2)
		mutate(bad)
		check(name, bad.Snapshot(), SnapshotLimits{}, ErrSnapshotMalformed)
	}
	for _, prefix := range []string{"A", "\a", "\x1bc", "\x1b[2J", "\x1b]2;changed\a", "\x1b]2;changed\x1b", "\x1b]2;cancel\x18", "\x1bP$qm\x1b\\", "😀", "\xf0\x80"} {
		bad := bytes.Clone(valid[:len(valid)-1])
		bad = binary.AppendUvarint(bad, uint64(len(prefix)))
		bad = append(bad, prefix...)
		check("completed-prefix", sealSnapshot(bad), SnapshotLimits{}, ErrSnapshotMalformed)
	}
}

func TestHistoryFormatterGrammar(t *testing.T) {
	for _, line := range []string{
		"plain 漢 é",
		"\x1b[0;1;38;2;1;2;3;4:3mstyled\x1b[0m",
		"\x1b[1\"qprotected\x1b[0\"q\x1b[K",
		"\x1b[?2027h❤️\x1b[?2027l",
		"\x1b]8;id=a;https://example.com/\xff\x1b\\linked\x1b]8;;\x1b\\",
	} {
		var plain bytes.Buffer
		if !walkHistoryText([]byte(line), func(text []byte) { plain.Write(text) }) || !walkHistoryText([]byte(line), nil) {
			t.Fatalf("valid formatter output rejected: %q", line)
		}
		if bytes.ContainsAny(plain.Bytes(), "\x1b") {
			t.Fatalf("formatting leaked into text: %q", plain.Bytes())
		}
	}
	for _, line := range []string{
		"\a", "\r", "\n", "\x7f", "\u009b", "\xff", "\xe6\xbc", "\x1b", "\x1b[", "\x1b[0",
		"\x1b[c", "\x1b[6n", "\x1b[2J", "\x1b[?69h", "\x1b[2\"q", "\x1b[0K", "\x1bP$qm\x1b\\",
		"\x1b]52;;clipboard\x1b\\", "\x1b]8;;uri\a", "\x1b]8;;uri\x18", "\x1b]8;uri\x1b\\", "\x1b]8;;uri\x1b[c",
	} {
		if walkHistoryText([]byte(line), nil) {
			t.Fatalf("non-formatter command accepted: %q", line)
		}
	}
}

func TestSnapshotRecordings(t *testing.T) {
	paths, err := filepath.Glob("testdata/*.raw")
	if err != nil || len(paths) == 0 {
		t.Fatalf("recordings: %v", err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		a := New(100, 30)
		a.Write(data)
		var b Terminal
		if err := b.RestoreSnapshot(a.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		assertSnapshotState(t, a, &b)
	}
}

func TestSnapshotOwnsInputAndReusesAppend(t *testing.T) {
	a := New(12, 5)
	a.Write([]byte(snapshotScene() + "\x1b]2;partial"))
	data := a.Snapshot()
	var b Terminal
	if err := b.RestoreSnapshot(data, SnapshotLimits{}); err != nil {
		t.Fatal(err)
	}
	clear(data)
	assertSnapshotState(t, a, &b)
	a.Write([]byte(" title\aX"))
	b.Write([]byte(" title\aX"))
	assertSnapshotState(t, a, &b)
	buffer := make([]byte, 3, len(a.Snapshot())+3)
	copy(buffer, "pre")
	got := a.AppendSnapshot(buffer)
	if string(got[:3]) != "pre" || &got[0] != &buffer[0] || !bytes.Equal(got[3:], a.Snapshot()) {
		t.Fatal("AppendSnapshot did not preserve prefix and reuse storage")
	}
	if allocations := testing.AllocsPerRun(10, func() { a.AppendSnapshot(got[:3]) }); allocations != 0 {
		t.Fatalf("reusable append allocated %g times", allocations)
	}
}

func FuzzRestoreSnapshot(f *testing.F) {
	for _, in := range []string{"", "hello", "\x1b[?2027h❤️", snapshotScene()} {
		t := New(12, 5)
		t.Write([]byte(in))
		f.Add(t.Snapshot())
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64<<10 {
			t.Skip()
		}
		data = bytes.Clone(data)
		if len(data) >= snapshotHeader {
			sealSnapshot(data)
		}
		receiver := New(7, 3)
		receiver.Write([]byte("unchanged"))
		before := receiver.Snapshot()
		if err := receiver.RestoreSnapshot(data, SnapshotLimits{MaxBytes: 64 << 10, MaxCells: 4096}); err != nil {
			if !bytes.Equal(before, receiver.Snapshot()) {
				t.Fatal("failed restore changed receiver")
			}
			return
		}
		for y := range receiver.rows {
			for x := range receiver.cols {
				receiver.Cell(x, y)
			}
		}
		receiver.Repaint(true)
		receiver.Compact()
		receiver.Resize(8, 4)
		receiver.Write([]byte("\x1b8\x1b[2b\r\nnext\x1b[?1049hALT\x1b[?1049l"))
	})
}

func FuzzSnapshotContinuation(f *testing.F) {
	for _, in := range []string{
		"hello", "\x1b[38:2::10:20:30m漢\xf0\x9f", "\x1b[?2027h😀︎❤️👩‍👩‍👧‍👦",
		"\x1b[?2027h\x1b[?7lA❤️", "abc\x1b7\x1b[?7l\x1b[HX\x1b8",
		"\x1b]8;id=a;https://example.com\x1b\\á\x1b7\x1b[?1049hALT\x1b7",
		snapshotScene(),
	} {
		f.Add([]byte(in), uint8(12), uint8(5), uint16(len(in)/2))
	}
	f.Fuzz(func(t *testing.T, input []byte, width, height uint8, split uint16) {
		if len(input) > 4096 {
			t.Skip()
		}
		cut := int(split) % (len(input) + 1)
		var aReply, bReply bytes.Buffer
		aBells, bBells := 0, 0
		a := New(1+int(width%24), 1+int(height%8), WithScrollback(16), WithScrollbackBytes(8192), WithReply(&aReply), WithBell(func() { aBells++ }))
		a.Write(input[:cut])
		a.Resize(1+int(height%24), 1+int(width%8))
		if width&1 != 0 {
			a.Compact()
		}
		b := New(1, 1, WithReply(&bReply), WithBell(func() { bBells++ }))
		if err := b.RestoreSnapshot(a.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatalf("legitimate state rejected: %v", err)
		}
		assertSnapshotState(t, a, b)
		if bReply.Len() != 0 || bBells != 0 {
			t.Fatal("restore invoked callbacks")
		}
		aReply.Reset()
		aBells = 0
		for _, suffix := range [][]byte{input[cut:], []byte("\x1b[6n\x1b[?12$p\x1b]10;?\a\x1b8\x1b[2b\r\nnext\a")} {
			a.Write(suffix)
			b.Write(suffix)
			assertSnapshotState(t, a, b)
			if !bytes.Equal(aReply.Bytes(), bReply.Bytes()) || aBells != bBells {
				t.Fatal("future callbacks changed")
			}
		}
		a.Resize(1+int(width%16), 1+int(height%6))
		b.Resize(1+int(width%16), 1+int(height%6))
		a.Write([]byte("\x1b[?1049hALT\x1b[?1049l\x1b[2b"))
		b.Write([]byte("\x1b[?1049hALT\x1b[?1049l\x1b[2b"))
		assertSnapshotState(t, a, b)
	})
}

func BenchmarkSnapshot(b *testing.B) {
	for _, workload := range []struct {
		name     string
		terminal func() *Terminal
	}{
		{"prompt", func() *Terminal { t := New(80, 24); t.Write([]byte("user@host:~$ ")); return t }},
		{"sparse", func() *Terminal { t := New(200, 50); t.Write([]byte("\x1b[25;100Hsparse é\x1b[50;200H!")); return t }},
		{"full", fullTerm},
		{"rich-history", func() *Terminal {
			t := New(120, 40)
			for i := range 2000 {
				t.Write(fmt.Appendf(nil, "\x1b]8;id=%d;https://example.com/%d\x1b\\\x1b[38;2;%d;20;30mline %d 漢 á\x1b]8;;\x1b\\\x1b[0m\r\n", i, i, i%256, i))
			}
			return t
		}},
	} {
		b.Run(workload.name, func(b *testing.B) {
			t := workload.terminal()
			snapshot, repaint := t.Snapshot(), t.Repaint(true)
			b.Run("encode", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					t.Snapshot()
				}
				b.ReportMetric(float64(len(snapshot)), "snapshot-B")
			})
			b.Run("append", func(b *testing.B) {
				buffer := make([]byte, 0, len(snapshot))
				b.ReportAllocs()
				for b.Loop() {
					buffer = t.AppendSnapshot(buffer[:0])
				}
			})
			b.Run("restore", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					var restored Terminal
					if err := restored.RestoreSnapshot(snapshot, SnapshotLimits{}); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("repaint", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					t.Repaint(true)
				}
				b.ReportMetric(float64(len(repaint)), "repaint-B")
			})
			b.Run("repaint-append", func(b *testing.B) {
				buffer := make([]byte, 0, len(repaint))
				b.ReportAllocs()
				for b.Loop() {
					buffer = t.AppendRepaint(buffer[:0], true)
				}
			})
			b.Run("replay", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					New(t.cols, t.rows).Write(repaint)
				}
			})
		})
	}
}
