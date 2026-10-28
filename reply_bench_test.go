package phosphor

import (
	"fmt"
	"io"
	"testing"
)

// The PTY host normally handles replies even when a stream has no queries.
// Keep that dispatch cost visible at small and large PTY read sizes.
func BenchmarkWriteStreamReply(b *testing.B) {
	for _, stream := range streams(b) {
		for _, chunk := range []int{1024, 8192} {
			b.Run(fmt.Sprintf("%s/chunk-%d", stream.name, chunk), func(b *testing.B) {
				term := New(120, 40, WithReply(io.Discard))
				b.SetBytes(int64(len(stream.data)))
				b.ReportAllocs()
				for b.Loop() {
					for p := stream.data; len(p) > 0; {
						n := min(len(p), chunk)
						term.Write(p[:n])
						p = p[n:]
					}
				}
			})
		}
	}
}
