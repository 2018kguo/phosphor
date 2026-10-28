package phosphor

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestASCIIFillMatchesScalarAndKeepsCanaries(t *testing.T) {
	rng := rand.New(rand.NewPCG(91, 37))
	for n := 0; n < 258; n++ {
		for offset := 0; offset < 4; offset++ {
			dst := make([]cell, n+8)
			for i := range dst {
				dst[i] = cell{R: rune(rng.Uint32()), Fg: Color(rng.Uint32()), Bg: Color(rng.Uint32()), A: Attr(rng.Uint32()), E: uint16(rng.Uint32())}
			}
			want := slices.Clone(dst)
			input := make([]byte, n+8)
			for i := range input {
				input[i] = byte(rng.Uint32())
			}
			pen := cell{Fg: Color(rng.Uint32()), Bg: Color(rng.Uint32()), A: Attr(rng.Uint32()), E: uint16(rng.Uint32())}
			for i, b := range input[offset : offset+n] {
				c := pen
				c.R = rune(b)
				want[offset+i] = c
			}
			fillASCII8(dst[offset:offset+n], input[offset:offset+n], pen)
			if !slices.Equal(dst, want) {
				t.Fatalf("length %d offset %d: fields or canaries changed", n, offset)
			}
		}
	}
}
