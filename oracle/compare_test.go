package oracle

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/2018kguo/phosphor"
	"github.com/2018kguo/phosphor/internal/benchdata"
	cvt "github.com/charmbracelet/x/vt"
	"github.com/hinshun/vt10x"
)

// The emulators compared, behind one constructor: each returns a writer
// for a cols x rows terminal.
var emulators = []struct {
	name string
	new  func(cols, rows int) io.Writer
}{
	{"phosphor", func(c, r int) io.Writer { return phosphor.New(c, r) }},
	{"charmbracelet-x-vt", func(c, r int) io.Writer {
		e := cvt.NewEmulator(c, r)
		go io.Copy(io.Discard, e) // drain replies, or a query blocks Write
		return e
	}},
	{"vt10x", func(c, r int) io.Writer { return vt10x.New(vt10x.WithSize(c, r)) }},
}

func compareStreams(b *testing.B) []struct {
	name string
	data []byte
} {
	read := func(name string) []byte {
		d, err := os.ReadFile(filepath.Join("..", "testdata", name))
		if err != nil {
			b.Fatal(err)
		}
		return d
	}
	return []struct {
		name string
		data []byte
	}{
		{"vim", read("vim.raw")},
		{"less", read("less.raw")},
		{"claude", benchdata.Claude(1<<20, 120, 40)},
		{"logs", benchdata.Logs(1 << 20)},
		{"progress", benchdata.Progress(1 << 20)},
	}
}

// BenchmarkCompareWrite feeds the same program output, in 8 KiB writes as
// a PTY reader would, to each emulator.
func BenchmarkCompareWrite(b *testing.B) {
	for _, s := range compareStreams(b) {
		for _, e := range emulators {
			b.Run(s.name+"/"+e.name, func(b *testing.B) {
				term := e.new(120, 40)
				b.SetBytes(int64(len(s.data)))
				b.ReportAllocs()
				for b.Loop() {
					for p := s.data; len(p) > 0; {
						n := min(len(p), 8192)
						term.Write(p[:n])
						p = p[n:]
					}
				}
			})
		}
	}
}

// BenchmarkCompareIdle reports the heap one 80x24 terminal holds after
// showing a shell prompt.
func BenchmarkCompareIdle(b *testing.B) {
	for _, e := range emulators {
		b.Run(e.name, func(b *testing.B) {
			const n = 200
			var per float64
			for b.Loop() {
				terms := make([]io.Writer, n)
				h0 := heapInUse()
				for i := range terms {
					terms[i] = e.new(80, 24)
					terms[i].Write([]byte("user@host:~$ "))
				}
				per = float64(heapInUse()-h0) / n
				runtime.KeepAlive(terms)
			}
			b.ReportMetric(per, "heap-B/term")
		})
	}
}

func heapInUse() uint64 {
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.HeapInuse
}
