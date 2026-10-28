package phosphor

import (
	"slices"
	"testing"
)

func TestDamageRangesPreserveOtherRows(t *testing.T) {
	const rows = 194
	term := New(2, rows)
	term.ClearDirty()
	for from := 0; from <= rows; from++ {
		for to := from; to <= rows; to++ {
			want := make([]uint64, len(term.dirty.bits))
			for i := range want {
				want[i] = 0xaaaaaaaaaaaaaaaa
			}
			copy(term.dirty.bits, want)
			for y := from; y < to; y++ {
				want[y>>6] |= 1 << (y & 63)
			}
			term.touchRows(from, to)
			if !slices.Equal(term.dirty.bits, want) {
				t.Fatalf("range [%d,%d) changed other rows: got %x want %x", from, to, term.dirty.bits, want)
			}
		}
	}
}

func BenchmarkWriteStreamDamage(b *testing.B) {
	for _, stream := range streams(b) {
		b.Run(stream.name, func(b *testing.B) {
			term := New(120, 40)
			term.ClearDirty()
			b.SetBytes(int64(len(stream.data)))
			b.ReportAllocs()
			for b.Loop() {
				for p := stream.data; len(p) > 0; {
					n := min(len(p), 8192)
					term.Write(p[:n])
					term.ClearDirty()
					p = p[n:]
				}
			}
		})
	}
}
