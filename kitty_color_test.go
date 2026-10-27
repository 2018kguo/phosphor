package phosphor

import (
	"bytes"
	"strings"
	"testing"
)

func TestKittyColorsSharePalette(t *testing.T) {
	base := DefaultPalette()
	base.Colors[1], base.Foreground, base.Background, base.Cursor = RGB{1, 2, 3}, RGB{4, 5, 6}, RGB{7, 8, 9}, RGB{10, 11, 12}
	var replies bytes.Buffer
	term := New(8, 3, WithPalette(base), WithReply(&replies))
	term.ClearDirty()
	term.Write([]byte("\x1b]21;1=rgb:12/34/56;foreground=red;background=rgbi:0/0.5/1;cursor=#aabbcc\x1b\\"))
	p := term.Palette()
	if p.Colors[1] != (RGB{0x12, 0x34, 0x56}) || p.Foreground != (RGB{255, 0, 0}) || p.Background != (RGB{0, 127, 255}) || p.Cursor != (RGB{170, 187, 204}) || !term.Dirty() {
		t.Fatalf("palette changes missing: %v %v %v %v dirty=%v", p.Colors[1], p.Foreground, p.Background, p.Cursor, term.Dirty())
	}
	// Both adapters read and write the same entries, including query order
	// within a batch and canonical index formatting.
	term.Write([]byte("\x1b]4;1;rgb:01/02/03\a\x1b]21;001=?;1=blue;1=?;foreground=?;cursor=?\a"))
	if got, want := replies.String(), "\x1b]21;1=rgb:01/02/03;1=rgb:00/00/ff;foreground=rgb:ff/00/00;cursor=rgb:aa/bb/cc\a"; got != want {
		t.Fatalf("reply %q, want %q", got, want)
	}
	term.Write([]byte("\x1b]21;1=;foreground;background=;cursor\x1b\\"))
	if term.Palette() != base {
		t.Fatal("kitty reset did not restore base palette")
	}
	term.Write([]byte("\x1b]21;1=red;foreground=blue\x1b\\\x1b]104;1\a\x1b]110\a"))
	if term.Palette() != base {
		t.Fatal("xterm reset did not remove kitty changes")
	}
}

func TestKittyColorCheckpointAndBounds(t *testing.T) {
	input := []byte("\x1b]21;1=red;foreground=rgbi:1/0.5/0;cursor=#aabbcc\x1b\\\x1b]21;1=?;foreground=?;cursor=?\a")
	for cut := 0; cut <= len(input); cut++ {
		var aReply, bReply bytes.Buffer
		a := New(8, 3, WithReply(&aReply))
		a.Write(input[:cut])
		b := New(1, 1, WithReply(&bReply))
		if err := b.RestoreSnapshot(a.Snapshot(), SnapshotLimits{}); err != nil {
			t.Fatalf("cut %d: %v", cut, err)
		}
		aReply.Reset()
		a.Write(input[cut:])
		b.Write(input[cut:])
		b.Compact()
		assertSnapshotState(t, a, b)
		if !bytes.Equal(aReply.Bytes(), bReply.Bytes()) {
			t.Fatalf("cut %d: reply changed across snapshot", cut)
		}
	}
	term := New(8, 3)
	base := term.Palette()
	for _, body := range []string{
		"-1=red;256=red;1_0=red;foreground =red;1=bad-color;selection_foreground=red;cursor_text=?;unknown=?",
		"1=red;" + strings.Repeat("0=;", maxKittyColorRequests),
	} {
		term.Write([]byte("\x1b]21;" + body + "\x1b\\"))
		if term.Palette() != base {
			t.Fatalf("invalid or oversized batch changed colors: %q", body[:min(64, len(body))])
		}
	}
}

func TestNamedAndIntensityColors(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want RGB
		ok   bool
	}{
		{"\tFoReSt GrEeN ", RGB{34, 139, 34}, true}, {"LawnGreen", RGB{124, 252, 0}, true},
		{"rgbi:1/.5/-0", RGB{255, 127, 0}, true}, {"rgbi:2/-1/1e-1", RGB{255, 0, 25}, true},
		{"rgbi:1e999/-1e999/1e-999", RGB{255, 0, 0}, true},
		{"rgbi:NaN/0/0", RGB{}, false}, {"rgbi:Inf/0/0", RGB{}, false}, {"rgbi:0x1p-1/0/0", RGB{}, false},
		{"rgbi:0/0/0/0", RGB{}, false}, {"rgbi:/0/0", RGB{}, false}, {"rgbi:./0/0", RGB{}, false},
	} {
		if got, ok := parseColor(tc.in); got != tc.want || ok != tc.ok {
			t.Fatalf("parseColor(%q) = %v,%v, want %v,%v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
	for i, e := range x11ColorNames {
		if i > 0 && x11ColorNames[i-1].name >= e.name {
			t.Fatal("generated names are not sorted and unique")
		}
		if got, ok := parseColor(e.name); !ok || got != e.color {
			t.Fatalf("generated color %q failed lookup", e.name)
		}
	}
	if n := testing.AllocsPerRun(100, func() { _, _ = parseColor("FoReStGrEeN") }); n != 0 {
		t.Fatalf("named color lookup allocated %g times", n)
	}
}
