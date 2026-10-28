package phosphor

import (
	"fmt"
	"io"
	"strings"
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
