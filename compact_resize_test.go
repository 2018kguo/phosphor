package phosphor

import (
	"fmt"
	"strings"
	"testing"
)

func historyPoolInput(kind string) string {
	var b strings.Builder
	printCell := func(group string, i int) {
		fmt.Fprintf(&b, "\x1b[38;5;%dm", i%16)
		if kind != "clusters" {
			fmt.Fprintf(&b, "\x1b]8;;https://history/%s/%d\x1b\\", group, i)
		}
		if kind != "links" {
			fmt.Fprintf(&b, "a%c%c", rune(0x300+i%64), rune(0x300+i/64))
		} else {
			b.WriteByte('a')
		}
	}
	// Keep enough entries live to give the uncollected control a large
	// collection threshold. The first three rows later become history.
	for y := range 6 {
		fmt.Fprintf(&b, "\x1b[%d;1H", y+1)
		for x := range 100 {
			printCell("warm", 100*y+x)
		}
	}
	b.WriteString("\x1b]8;;\x1b\\\x1b[4;1H\x1b[J\x1b[6;1H\r\n\r\n\r\n")
	// Existing main rows will be rewrapped before history is imported.
	b.WriteString("\x1b[2;1H")
	for x := range 80 {
		printCell("main", 700+x)
	}
	b.WriteString("\x1b]8;;https://saved-main\x1b\\\x1b7\x1b]8;;\x1b\\")
	// A kept alternate screen and its saved pen are also collection roots.
	b.WriteString("\x1b[?47h\x1b[1;1H")
	printCell("alt", 800)
	b.WriteString("\x1b]8;;https://saved-alt\x1b\\\x1b7\x1b]8;;\x1b\\\x1b[?47l\x1b[6;1H$ ")
	return b.String()
}

func TestCompactResizeHistoryPools(t *testing.T) {
	for _, kind := range []string{"clusters", "links", "mixed"} {
		for _, size := range [][2]int{{100, 10}, {40, 18}} {
			for _, alt := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%dx%d/alt=%v", kind, size[0], size[1], alt), func(t *testing.T) {
					original, compacted := New(100, 6), New(100, 6)
					for _, term := range []*Terminal{original, compacted} {
						term.Write([]byte(historyPoolInput(kind)))
						if term.ScrollbackLen() != 3 {
							t.Fatalf("trace produced %d history lines, want three", term.ScrollbackLen())
						}
						if alt {
							term.Write([]byte("\x1b[?1049h\x1b[H\x1b]8;;https://active-alt\x1b\\ALT á\x1b7"))
						}
					}
					compacted.Compact()
					clusterCap, linkCap := compacted.x.clusterCap, compacted.x.linkCap
					original.Resize(size[0], size[1])
					compacted.Resize(size[0], size[1])
					if compacted.x.clusterCap != clusterCap || compacted.x.linkCap != linkCap {
						t.Fatal("resize did not restore collection thresholds")
					}
					assertPoolEquivalent(t, original, compacted)
					collectClusters := len(compacted.x.clusters) > 0 && len(compacted.x.clusters) >= clusterCap
					collectLinks := len(compacted.x.links) >= linkCap
					// Cross the restored collection thresholds only after all
					// imported and rewrapped rows are attached to the terminal.
					for i := range 100 {
						next := fmt.Appendf(nil, "\x1b[%d;%dH\x1b]8;;https://future/%d\x1b\\z%c%c", size[1]-1, size[0], i, rune(0x300+i%64), rune(0x300+i/64))
						original.Write(next)
						compacted.Write(next)
						if i == 0 && (collectClusters && compacted.x.clusterCap == clusterCap || collectLinks && compacted.x.linkCap == linkCap) {
							t.Fatal("ordinary write did not resume pool collection")
						}
					}
					assertPoolEquivalent(t, original, compacted)
					for _, next := range []string{"\x1b8X", "\x1b[?1049l", "\x1b8Y", "\x1b[?47h\x1b8Z\x1b[?47l", "\x1b[2b"} {
						original.Write([]byte(next))
						compacted.Write([]byte(next))
						assertPoolEquivalent(t, original, compacted)
					}
					for _, later := range [][2]int{{25, 22}, {100, 12}} {
						original.Resize(later[0], later[1])
						compacted.Resize(later[0], later[1])
						assertPoolEquivalent(t, original, compacted)
						compacted.Compact()
						assertPoolEquivalent(t, original, compacted)
					}
				})
			}
		}
	}
}

func TestResizePlainHistoryDoesNotAllocateExtra(t *testing.T) {
	term := New(12, 3)
	term.Write([]byte(strings.Repeat("plain\r\n", 8)))
	if term.x != nil {
		t.Fatal("plain history trace allocated extra state")
	}
	term.Compact()
	term.Resize(8, 10)
	if term.x != nil {
		t.Fatal("plain history import allocated Unicode or hyperlink storage")
	}
}
