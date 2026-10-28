package phosphor

import (
	"os"
	"testing"
)

// Match the native comparison's recorded PTY streams and 1024-byte reads.
func BenchmarkWriteRecordedPTY(b *testing.B) {
	for _, name := range []string{"rec-nvim", "rec-nano", "rec-man"} {
		b.Run(name, func(b *testing.B) {
			data, err := os.ReadFile("testdata/" + name + ".raw")
			if err != nil {
				b.Fatal(err)
			}
			term := New(120, 40)
			write := func() {
				for i := 0; i < len(data); i += 1024 {
					term.Write(data[i:min(i+1024, len(data))])
				}
			}
			write()
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				write()
			}
		})
	}
}
