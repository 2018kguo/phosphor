package phosphor

import (
	"bytes"
	"math/rand/v2"
	"testing"
)

func TestHistoryASCIICopyMatchesScalar(t *testing.T) {
	rng := rand.New(rand.NewPCG(13, 57))
	for trial := range 10000 {
		length := 1 + rng.IntN(256)
		row := make([]cell, length)
		style := cell{Fg: Color(rng.Uint32()), Bg: Color(rng.Uint32()), A: Attr(rng.Uint32()), E: uint16(rng.Uint32())}
		for i := range row {
			row[i] = style
			row[i].R = rune(1 + rng.IntN(127))
		}
		if boundary := trial % (length + 1); boundary < length {
			c := &row[boundary]
			switch trial % 6 {
			case 0:
				c.R = []rune{0, 128, '界', wideTail, spacerHead, clusterBase}[trial/6%6]
			case 1:
				c.Fg ^= 1 << uint(trial%32)
			case 2:
				c.Bg ^= 1 << uint(trial%32)
			case 3:
				c.A ^= 1 << uint(trial%16)
			case 4:
				c.E ^= 1 << uint(trial%16)
			case 5:
				c.R = rune(rng.Uint32())
			}
		}
		got, want := bytes.Repeat([]byte{0xcc}, length), bytes.Repeat([]byte{0xcc}, length)
		n := copyASCIICells(got, row)
		k := 0
		for k < len(row) && row[k].R > 0 && row[k].R < 128 && styleOf(row[k]) == styleOf(row[0]) {
			want[k] = byte(row[k].R)
			k++
		}
		if n != k || !bytes.Equal(got, want) {
			t.Fatalf("trial %d copied %d cells, want %d", trial, n, k)
		}
	}
}

func TestHistoryASCIICopyWordBoundaries(t *testing.T) {
	for _, fg := range []Color{0, 1, 0x80000000, 0xffffffff} {
		for _, r := range []rune{-1 << 31, wideTail, 0, 1, 127, 128, 1<<31 - 1} {
			for _, delta := range []Color{0, 1, 0xffffffff} {
				row := []cell{{R: 'A', Fg: fg}, {R: r, Fg: fg + delta}, {R: 'Z', Fg: fg}}
				want := 1
				if r > 0 && r < 128 && delta == 0 {
					want = 3
				}
				dst := []byte{0xcc, 0xcc, 0xcc}
				if n := copyASCIICells(dst, row); n != want || dst[0] != 'A' || want == 1 && dst[1] != 0xcc {
					t.Fatalf("foreground=%08x delta=%08x rune=%d: copied %d, want %d", fg, delta, r, n, want)
				}
			}
		}
	}
}
