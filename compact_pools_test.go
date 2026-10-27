package phosphor

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func poolChurnInput() string {
	var b strings.Builder
	for i := range 64 {
		fmt.Fprintf(&b, "\x1b[%d;%dH\x1b]8;;https://unused/%d/%s\x1b\\a%c%c", 1+i/16, 1+4*(i%16), i, strings.Repeat("x", 512), rune(0x300+i%32), rune(0x300+i/32))
	}
	b.WriteString("\x1b]8;;\x1b\\\x1b[2J\x1b[H$ ")
	return b.String()
}

func assertPoolEquivalent(t *testing.T, original, compacted *Terminal) {
	t.Helper()
	assertCompactEquivalent(t, original, compacted)
	for _, screen := range []struct {
		name string
		a, b [][]cell
	}{{"main", original.main, compacted.main}, {"alternate", original.alt, compacted.alt}} {
		if len(screen.a) != len(screen.b) {
			t.Fatalf("%s screen height changed", screen.name)
		}
		for y := range screen.a {
			for x := range original.cols {
				a, b := Cell{Width: 1}, Cell{Width: 1}
				if screen.a[y] != nil {
					a = original.publicCell(screen.a[y], x)
				}
				if screen.b[y] != nil {
					b = compacted.publicCell(screen.b[y], x)
				}
				if a != b {
					t.Fatalf("%s cell %d,%d changed: %+v / %+v", screen.name, x, y, a, b)
				}
			}
		}
	}
}

func TestCompactReleasesDeadPools(t *testing.T) {
	input := []byte(poolChurnInput())
	original, compacted := New(80, 24), New(80, 24)
	original.Write(input)
	compacted.Write(input)
	if compacted.x == nil || len(compacted.x.links) == 0 || len(compacted.x.clusters) == 0 {
		t.Fatal("trace did not populate both pools")
	}
	compacted.ClearDirty()
	original.ClearDirty()
	compacted.Compact()
	if compacted.x != nil {
		t.Fatal("erased cells kept dead Unicode or hyperlink storage")
	}
	assertPoolEquivalent(t, original, compacted)
	for _, next := range []string{"\x1b[2b", "\x1b]8;;https://new\x1b\\é\x1b]8;;\x1b\\", "\r\nnext"} {
		original.Write([]byte(next))
		compacted.Write([]byte(next))
		assertPoolEquivalent(t, original, compacted)
	}
}

func TestCompactKeepsLivePoolReferences(t *testing.T) {
	prelude := "\x1b]8;;https://main-cell\x1b\\á\x1b]8;;https://main-save\x1b\\\x1b7" +
		"\x1b]8;;https://1049-save\x1b\\\x1b[?1049h\x1b[H\x1b]8;;https://alt-cell\x1b\\é" +
		"\x1b]8;;https://alt-save\x1b\\\x1b7"
	var churn strings.Builder
	for i := range 60 {
		fmt.Fprintf(&churn, "\x1b[4;1H\x1b]8;;https://unused/%d\x1b\\a%c", i, rune(0x300+i))
	}
	// The last cluster is erased from the grid but REP still refers to it.
	churn.WriteString("\x1b[4;1H\x1b[2K\x1b]8;;https://current-pen\x1b\\")
	original, compacted := New(12, 5), New(12, 5)
	for _, term := range []*Terminal{original, compacted} {
		term.Write([]byte(prelude + churn.String()))
		term.ClearDirty()
	}
	compacted.Compact()
	assertPoolEquivalent(t, original, compacted)
	if compacted.lastChar < clusterBase || compacted.cellString(cell{R: compacted.lastChar}) != original.cellString(cell{R: original.lastChar}) {
		t.Fatal("erased cluster needed by REP was collected")
	}
	for _, next := range []string{"\x1b[2b", "\x1b8X", "\x1b[?1049lY", "\x1b8Z", "\x1b[?47hkept\x1b[?47l"} {
		original.Write([]byte(next))
		compacted.Write([]byte(next))
		compacted.Compact()
		assertPoolEquivalent(t, original, compacted)
	}
	for _, term := range []*Terminal{original, compacted} {
		term.Resize(8, 8)
		term.Resize(14, 5)
	}
	compacted.Compact()
	assertPoolEquivalent(t, original, compacted)
}

func TestCompactKeepsSerializedHistory(t *testing.T) {
	var input strings.Builder
	for i := range 30 {
		fmt.Fprintf(&input, "\x1b]8;;https://history/%d\x1b\\line %d á\x1b]8;;\x1b\\\r\n", i, i)
	}
	input.WriteString("\x1b[2J\x1b[H$ ")
	original, compacted := New(40, 4), New(40, 4)
	original.Write([]byte(input.String()))
	compacted.Write([]byte(input.String()))
	if compacted.ScrollbackLen() == 0 {
		t.Fatal("trace did not populate history")
	}
	compacted.Compact()
	if compacted.x != nil {
		t.Fatal("self-contained history retained unused lookup tables")
	}
	assertPoolEquivalent(t, original, compacted)
	for _, term := range []*Terminal{original, compacted} {
		term.Resize(40, 12)
		term.Write([]byte("\x1b[?1049hnew á\x1b[?1049l"))
	}
	assertPoolEquivalent(t, original, compacted)
}

func TestCompactDoesNotRebuildLivePools(t *testing.T) {
	term := New(12, 4)
	term.Write([]byte("\x1b]8;;https://live\x1b\\á\x1b7"))
	term.Compact()
	cluster, link := &term.x.clusters[0], &term.x.links[0]
	if n := testing.AllocsPerRun(100, term.Compact); n != 0 {
		t.Fatalf("repeated compaction of unchanged live pools allocated %g times", n)
	}
	if cluster != &term.x.clusters[0] || link != &term.x.links[0] {
		t.Fatal("unchanged live pool storage was replaced")
	}
}

func TestCompactPoolsKeepsPendingInput(t *testing.T) {
	for _, c := range []struct{ prefix, suffix string }{
		{"\x1b]8;;https://partial", "\x1b\\X"},
		{"\x1b[38:2::10:20:", "30mX"},
		{"\x1bP$q", "m\x1b\\X"},
		{"\xf0\x9f", "\x98\x80X"},
	} {
		original, compacted := New(80, 24), New(80, 24)
		for _, term := range []*Terminal{original, compacted} {
			term.Write([]byte(poolChurnInput() + c.prefix))
		}
		compacted.Compact()
		if compacted.x != nil {
			t.Fatal("pending input kept dead lookup tables")
		}
		original.Write([]byte(c.suffix))
		compacted.Write([]byte(c.suffix))
		assertPoolEquivalent(t, original, compacted)
	}
}

func TestCompactPoolsKeepsTabsAndSavedModes(t *testing.T) {
	original, compacted := New(80, 24), New(80, 24)
	for _, term := range []*Terminal{original, compacted} {
		term.Write([]byte(poolChurnInput() + "\x1b[3g\x1b[1;5H\x1bH\x1b[?2004s\x1b[?2004h"))
	}
	compacted.Compact()
	if compacted.x == nil || compacted.x.tabs == nil || len(compacted.x.savedModes) == 0 {
		t.Fatal("other live extra state was collected with the lookup tables")
	}
	for _, term := range []*Terminal{original, compacted} {
		term.Write([]byte("\x1b[1;1H\tX\x1b[?2004r"))
	}
	assertPoolEquivalent(t, original, compacted)
	if compacted.Cell(4, 0).Rune != 'X' || compacted.Modes().BracketedPaste {
		t.Fatal("custom tab or saved mode no longer takes effect")
	}
}

func TestCompactDropsInvalidSavedLink(t *testing.T) {
	term := New(12, 4)
	term.Write([]byte("\x1b]8;;https://old\x1b\\\x1b[?1049h\x1b[?1049l\x1b]8;;\x1b\\$ "))
	term.Compact()
	if term.x != nil || term.saved != nil {
		t.Fatal("invalid 1049 slot retained cursor or hyperlink storage")
	}
	// A subsequent table must not revisit the invalid slot's old index.
	for i := range 40 {
		term.Write(fmt.Appendf(nil, "\x1b]8;;https://new/%d\x1b\\", i))
	}
	term.Write([]byte("\x1b[?1049h\x1b[?1049lX"))
	if got := term.Cell(2, 0).Link.URI; got != "https://new/39" {
		t.Fatalf("new hyperlink table restored %q", got)
	}
}

func TestCompactDiscardedSaveRestoresAsUnset(t *testing.T) {
	original, compacted := New(12, 4), New(12, 4)
	for _, term := range []*Terminal{original, compacted} {
		term.Write([]byte("\x1b[2;3H\x1b[31m\x1b[?1049hALT\x1b[?1049l"))
	}
	compacted.Compact()
	if compacted.saved != nil {
		t.Fatal("trace did not release its invalid cursor slots")
	}
	for _, next := range []string{"\x1b8X", "\x1b[2b", "\x1b[?47h\x1b[Hkept á\x1b[?47l", "\x1b[?47hX\x1b[?47l"} {
		original.Write([]byte(next))
		compacted.Write([]byte(next))
		compacted.Compact()
		assertPoolEquivalent(t, original, compacted)
	}
}

func TestCompactLargeLivePools(t *testing.T) {
	term := New(40, 12)
	for i := range 300 {
		term.Write(fmt.Appendf(nil, "\x1b[%d;%dH\x1b]8;;https://live/%d\x1b\\%c%c", 1+i/40, 1+i%40, i, 'a'+rune(i%26), rune(0x300+i/26)))
	}
	term.ClearDirty()
	before := term.Repaint(true)
	cluster, link := &term.x.clusters[0], &term.x.links[0]
	term.Compact()
	if !bytes.Equal(before, term.Repaint(true)) || term.Dirty() {
		t.Fatal("large liveness map changed the rendered state")
	}
	if cluster != &term.x.clusters[0] || link != &term.x.links[0] {
		t.Fatal("large unchanged pools were rebuilt")
	}
}
