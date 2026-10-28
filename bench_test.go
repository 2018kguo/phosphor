package phosphor

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/2018kguo/phosphor/internal/benchdata"
)

func benchInput(b *testing.B) []byte {
	var all []byte
	// The original vim and less recordings only, so the benchmark input
	// stays the same as recordings are added.
	for _, p := range []string{"testdata/less.raw", "testdata/vim.raw"} {
		d, err := os.ReadFile(p)
		if err != nil {
			b.Fatal(err)
		}
		all = append(all, d...)
	}
	return all
}

// BenchmarkWrite feeds recorded vim and less output.
func BenchmarkWrite(b *testing.B) {
	in := benchInput(b)
	term := New(100, 30)
	b.SetBytes(int64(len(in)))
	b.ReportAllocs()
	for b.Loop() {
		term.Write(in)
	}
}

// BenchmarkWriteLines feeds plain scrolling log output, which exercises
// scrollback.
func BenchmarkWriteLines(b *testing.B) {
	var in []byte
	for range 1000 {
		in = append(in, "\x1b[32mINFO\x1b[0m request id=12345 path=/api/v1/items status=200 took=3ms\r\n"...)
	}
	term := New(120, 40)
	b.SetBytes(int64(len(in)))
	b.ReportAllocs()
	for b.Loop() {
		term.Write(in)
	}
}

func BenchmarkRepaint(b *testing.B) {
	term := New(100, 30)
	term.Write(benchInput(b))
	b.ReportAllocs()
	for b.Loop() {
		term.Repaint(true)
	}
}

// BenchmarkMemory reports the heap one terminal holds when idle at a
// prompt and when its screen and scrollback are full.
func BenchmarkMemory(b *testing.B) {
	full := func(t *Terminal) {
		for range 5000 {
			t.Write([]byte("\x1b[32mINFO\x1b[0m request id=12345 path=/api/v1/items status=200 took=3ms and some more text\r\n"))
		}
	}
	for _, c := range []struct {
		name string
		fill func(*Terminal)
	}{
		{"prompt-80x24", func(t *Terminal) { t.Write([]byte("user@host:~$ ")) }},
		{"full-80x24", full},
		{"full-200x50", full},
	} {
		b.Run(c.name, func(b *testing.B) {
			cols, rows := 80, 24
			if c.name == "full-200x50" {
				cols, rows = 200, 50
			}
			var per float64
			for b.Loop() {
				h0 := heapInUse()
				t := New(cols, rows)
				c.fill(t)
				per = float64(heapInUse() - h0)
				runtime.KeepAlive(t)
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

// streams are the realistic inputs: the recorded vim and less sessions,
// and generated agent-TUI, log and progress-bar output (~1 MiB each).
func streams(b *testing.B) []struct {
	name string
	data []byte
} {
	vim, err := os.ReadFile("testdata/vim.raw")
	if err != nil {
		b.Fatal(err)
	}
	less, err := os.ReadFile("testdata/less.raw")
	if err != nil {
		b.Fatal(err)
	}
	return []struct {
		name string
		data []byte
	}{
		{"vim", vim},
		{"less", less},
		{"claude", benchdata.Claude(1<<20, 120, 40)},
		{"logs", benchdata.Logs(1 << 20)},
		{"progress", benchdata.Progress(1 << 20)},
	}
}

// BenchmarkWriteStream feeds each stream in PTY-read-sized chunks, as the
// daemon does.
func BenchmarkWriteStream(b *testing.B) {
	for _, s := range streams(b) {
		b.Run(s.name, func(b *testing.B) {
			term := New(120, 40)
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

// fullTerm is a 120x40 terminal with full scrollback of agent output.
func fullTerm() *Terminal {
	term := New(120, 40)
	term.Write(benchdata.Claude(4<<20, 120, 40))
	return term
}

// BenchmarkSerialize covers what the API and checkpoints read from a
// terminal with a full scrollback: the repaint with and without
// scrollback, and the screen and scrollback as text.
func BenchmarkSerialize(b *testing.B) {
	term := fullTerm()
	b.Run("RepaintScreen", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			term.Repaint(false)
		}
	})
	b.Run("RepaintScrollback", func(b *testing.B) {
		b.SetBytes(int64(len(term.Repaint(true))))
		b.ReportAllocs()
		for b.Loop() {
			term.Repaint(true)
		}
	})
	b.Run("Lines", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			term.Lines()
		}
	})
	b.Run("ScrollbackLines", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			term.ScrollbackLines()
		}
	})
}

// BenchmarkReplay is restoring a screen: a fresh emulator fed a full
// checkpoint (scrollback plus screen).
func BenchmarkReplay(b *testing.B) {
	ckpt := fullTerm().Repaint(true)
	b.SetBytes(int64(len(ckpt)))
	b.ReportAllocs()
	for b.Loop() {
		New(120, 40).Write(ckpt)
	}
}

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

// Keep mode-on Unicode and hyperlinks visible beside the ASCII-heavy streams.
// The mixed and link bytes match the native oracle's audited fixtures.
func BenchmarkWriteUnicodeAndLinks(b *testing.B) {
	for _, fixture := range []struct{ name, line string }{
		{"graphemes", "日本語 e\u0301 👩‍💻 🇺🇸 👨‍👩‍👧‍👦 한글\r\n"},
		{"cjk", "日本語 中文 한글 테스트\r\n"},
		{"ascii", "plain ASCII text with grapheme mode enabled\r\n"},
		{"links", "\x1b]8;;https://example.invalid/build/123\x1b\\\x1b[32mcomplete build 123\x1b[0m\x1b]8;;\x1b\\\r\n"},
	} {
		data := []byte(strings.Repeat(fixture.line, (1<<20)/len(fixture.line)+1))
		if fixture.name != "links" {
			data = append([]byte("\x1b[?2027h"), data...)
		}
		for _, history := range []string{"budgeted", "none"} {
			for _, chunk := range []int{1024, 8192} {
				b.Run(fmt.Sprintf("%s/%s/chunk-%d", fixture.name, history, chunk), func(b *testing.B) {
					opts := []Option{WithReply(io.Discard)}
					if history == "none" {
						opts = append(opts, WithScrollback(0))
					}
					term := New(120, 40, opts...)
					b.SetBytes(int64(len(data)))
					b.ReportAllocs()
					for b.Loop() {
						for p := data; len(p) > 0; {
							n := min(len(p), chunk)
							term.Write(p[:n])
							p = p[n:]
						}
					}
				})
			}
		}
	}
}

// Distinct clusters exercise cache misses as well as repeated display text.
func BenchmarkWriteGraphemeVariety(b *testing.B) {
	for _, count := range []int{16, 1024} {
		var text strings.Builder
		for i := range count {
			fmt.Fprintf(&text, "%c\u0301 ", rune(0x4e00+i))
			if i%16 == 15 {
				text.WriteString("\r\n")
			}
		}
		line := text.String()
		data := []byte("\x1b[?2027h" + strings.Repeat(line, (1<<20)/len(line)+1))
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			term := New(120, 40, WithReply(io.Discard))
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				for p := data; len(p) > 0; {
					n := min(len(p), 1024)
					term.Write(p[:n])
					p = p[n:]
				}
			}
		})
	}
}

func richGraphemeHistory() *Terminal {
	term := New(20, 2)
	term.Write([]byte("\x1b[?2027h"))
	for i := range 8 {
		term.Write([]byte(fmt.Sprintf("\x1b]8;;https://example.com/%d/%s\x1b\\❤️", i, strings.Repeat("a", 3500))))
	}
	term.Write([]byte("\x1b]8;;\x1b\\\r\nX\r\nY"))
	return term
}

func TestRichGraphemeHistory(t *testing.T) {
	term := richGraphemeHistory()
	if term.ScrollbackLen() != 1 {
		t.Fatalf("history lines%d", term.ScrollbackLen())
	}
	rows := term.ScrollbackCells(0)
	for i := range 8 {
		c := rows[0][2*i]
		if c.Rune != '❤' || c.Combining != "️" || c.Width != 2 || !strings.Contains(c.Link.URI, fmt.Sprintf("/%d/", i)) {
			t.Fatalf("cell%d:%+v", i, c)
		}
	}
	if term.ScrollbackLine(0) != strings.Repeat("❤️", 8) {
		t.Fatalf("text%q", term.ScrollbackLine(0))
	}
}

func BenchmarkGraphemeRichHistoryDecode(b *testing.B) {
	term := richGraphemeHistory()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = term.ScrollbackCells(0)
	}
	b.ReportMetric(float64(term.sbBytes), "history-B")
}
