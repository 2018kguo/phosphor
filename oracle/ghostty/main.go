//go:build ghostty

// Compare phosphor with a pinned, separately built libghostty-vt.
package main

/*
#include "bridge.h"
*/
import "C"

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/2018kguo/phosphor"
	"github.com/2018kguo/phosphor/internal/benchdata"
)

type View struct {
	Cols      int      `json:"cols"`
	Rows      int      `json:"rows"`
	Lines     []string `json:"lines"`
	X         int      `json:"cursor_x"`
	Y         int      `json:"cursor_y"`
	Alt       bool     `json:"alt_screen"`
	Visible   bool     `json:"visible"`
	Pending   bool     `json:"pending_wrap"`
	Bracketed bool     `json:"bracketed"`
	AppCursor bool     `json:"application_cursor"`
	Origin    bool     `json:"origin"`
	Autowrap  bool     `json:"autowrap"`
	Insert    bool     `json:"insert"`
}
type Engine interface {
	Write([]byte)
	Resize(int, int)
	View() View
	Repaint(bool) []byte
	Snapshot() []byte
	Close()
}
type phos struct{ t *phosphor.Terminal }

func (p *phos) Write(b []byte)        { p.t.Write(b) }
func (p *phos) Resize(c, r int)       { p.t.Resize(c, r) }
func (p *phos) Close()                { p.t = nil }
func (p *phos) Repaint(h bool) []byte { return p.t.Repaint(h) }
func (p *phos) Snapshot() []byte      { return p.t.Snapshot() }
func (p *phos) View() View {
	c, r := p.t.Size()
	cur := p.t.Cursor()
	m := p.t.Modes()
	return View{c, r, p.t.Lines(), cur.X, cur.Y, m.AltScreen, cur.Visible, cur.PendingWrap, m.BracketedPaste, m.ApplicationCursorKeys, m.Origin, m.Autowrap, m.Insert}
}

type ghost struct{ t C.GhosttyTerminal }

func must(code C.int) {
	if code != 0 {
		panic(fmt.Sprintf("Ghostty result %d", int(code)))
	}
}
func (g *ghost) Write(b []byte) {
	if len(b) > 0 {
		C.ghostty_terminal_vt_write(g.t, (*C.uint8_t)(unsafe.Pointer(&b[0])), C.size_t(len(b)))
	}
}
func (g *ghost) Resize(c, r int) {
	must(C.int(C.ghostty_terminal_resize(g.t, C.uint16_t(c), C.uint16_t(r), 8, 16)))
}
func (g *ghost) Close() { C.ghostty_terminal_free(g.t); g.t = nil }
func take(p *C.uint8_t, n C.size_t) []byte {
	b := C.GoBytes(unsafe.Pointer(p), C.int(n))
	C.ghostty_free(nil, p, n)
	return b
}
func (g *ghost) format(vt, h bool) []byte {
	var p *C.uint8_t
	var n C.size_t
	v, history := 0, 0
	if vt {
		v = 1
	}
	if h {
		history = 1
	}
	must(C.exp_format(g.t, C.int(v), C.int(history), &p, &n))
	return take(p, n)
}
func (g *ghost) Repaint(h bool) []byte { return g.format(true, h) }
func (g *ghost) Snapshot() []byte {
	var p *C.uint8_t
	var n C.size_t
	must(C.int(C.ghostty_snapshot_encode_alloc(g.t, nil, &p, &n)))
	return take(p, n)
}
func (g *ghost) continuation() []byte {
	var p *C.uint8_t
	var n C.size_t
	must(C.int(C.ghostty_terminal_continuation_alloc(g.t, nil, &p, &n)))
	return take(p, n)
}
func stats(t Engine) map[string]any {
	if p, ok := t.(*phos); ok {
		return map[string]any{"history_rows": p.t.ScrollbackLen(), "history_unit": "logical-lines"}
	}
	g := t.(*ghost)
	var rows C.size_t
	must(C.int(C.ghostty_terminal_get(g.t, C.GHOSTTY_TERMINAL_DATA_SCROLLBACK_ROWS, unsafe.Pointer(&rows))))
	var m C.GhosttyTerminalMemoryUsage
	m.size = C.size_t(C.sizeof_GhosttyTerminalMemoryUsage)
	must(C.int(C.ghostty_terminal_get(g.t, C.GHOSTTY_TERMINAL_DATA_MEMORY_USAGE, unsafe.Pointer(&m))))
	return map[string]any{"history_rows": int(rows), "history_unit": "physical-rows", "native_page_resident_bytes": uint64(m.primary_resident_bytes + m.alternate_resident_bytes), "native_page_virtual_bytes": uint64(m.primary_virtual_bytes + m.alternate_virtual_bytes), "native_pages": uint64(m.primary_pages + m.alternate_pages), "compression_supported": bool(m.compression_supported)}
}
func (g *ghost) View() View {
	var i C.ExpInfo
	must(C.exp_info(g.t, &i))
	lines := strings.Split(strings.ReplaceAll(string(g.format(false, false)), "\r\n", "\n"), "\n")
	for n := range lines {
		lines[n] = strings.TrimRight(lines[n], " ")
	}
	if len(lines) > int(i.rows) {
		lines = lines[:int(i.rows)]
	}
	for len(lines) < int(i.rows) {
		lines = append(lines, "")
	}
	return View{int(i.cols), int(i.rows), lines, int(i.x), int(i.y), bool(i.alt), bool(i.visible), bool(i.pending_wrap), bool(i.bracketed), bool(i.application_cursor), bool(i.origin), bool(i.autowrap), bool(i.insert)}
}
func newEngine(name string, c, r int) Engine {
	if name == "phosphor" {
		return &phos{phosphor.New(c, r)}
	}
	if name != "ghostty" {
		panic(name)
	}
	t := C.exp_new(C.int(c), C.int(r))
	if t == nil {
		panic("ghostty new")
	}
	return &ghost{t}
}
func restore(name string, b []byte, c, r int) Engine {
	if name == "phosphor" {
		var t phosphor.Terminal
		if err := t.RestoreSnapshot(b, phosphor.SnapshotLimits{}); err != nil {
			panic(err)
		}
		return &phos{&t}
	}
	var t C.GhosttyTerminal
	must(C.exp_restore((*C.uint8_t)(unsafe.Pointer(&b[0])), C.size_t(len(b)), &t))
	return &ghost{t}
}
func emit(v any) {
	if record, ok := v.(map[string]any); ok {
		if label := os.Getenv("COMPARE_LABEL"); label != "" {
			record["label"] = label
		}
	}
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	fmt.Println(string(b))
}
func same(a, b View) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return bytes.Equal(aJSON, bJSON)
}

type Stream struct {
	Name       string
	Data       []byte
	Cols, Rows int
}

func streams() []Stream {
	out := []Stream{{"claude-synthetic", benchdata.Claude(1<<20, 120, 40), 120, 40}, {"logs", benchdata.Logs(1 << 20), 120, 40}, {"progress", benchdata.Progress(1 << 20), 120, 40}}
	for _, f := range []string{"vim", "less", "rec-nvim", "rec-nano", "rec-man"} {
		b, e := os.ReadFile(os.Getenv("PHOSPHOR_DIR") + "/testdata/" + f + ".raw")
		if e != nil {
			panic(e)
		}
		out = append(out, Stream{f + "-recording", b, 120, 40})
	}
	return out
}
func feed(t Engine, b []byte, chunk int) {
	for len(b) > 0 {
		n := min(len(b), chunk)
		t.Write(b[:n])
		b = b[n:]
	}
}
func processCPU() time.Duration {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		panic(err)
	}
	return time.Duration(usage.Utime.Nano() + usage.Stime.Nano())
}
func throughput(duration time.Duration) {
	rounds := 5
	if raw := os.Getenv("COMPARE_ROUNDS"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			panic("COMPARE_ROUNDS must be positive")
		}
		rounds = n
	}
	for _, s := range streams() {
		if selected := os.Getenv("COMPARE_STREAM"); selected != "" && s.Name != selected {
			continue
		}
		for _, chunk := range []int{1024, 4096, 8192} {
			if selected := os.Getenv("COMPARE_CHUNK"); selected != "" && fmt.Sprint(chunk) != selected {
				continue
			}
			for round := 0; round < rounds; round++ {
				order := []string{"phosphor", "ghostty"}
				if round%2 == 1 {
					slices.Reverse(order)
				}
				for _, name := range order {
					if selected := os.Getenv("COMPARE_ENGINE"); selected != "" && name != selected {
						continue
					}
					t := newEngine(name, s.Cols, s.Rows)
					feed(t, s.Data, chunk)
					runtime.GC()
					cpuStart := processCPU()
					start := time.Now()
					n := 0
					for time.Since(start) < duration {
						feed(t, s.Data, chunk)
						n++
					}
					elapsed := time.Since(start)
					cpu := processCPU() - cpuStart
					emit(map[string]any{"kind": "throughput", "engine": name, "stream": s.Name, "chunk": chunk, "round": round, "iterations": n, "bytes_per_iteration": len(s.Data), "seconds": elapsed.Seconds(), "mib_per_s": float64(n*len(s.Data)) / elapsed.Seconds() / (1 << 20), "cpu_seconds": cpu.Seconds(), "mib_per_cpu_s": float64(n*len(s.Data)) / cpu.Seconds() / (1 << 20)})
					t.Close()
				}
			}
		}
	}
}
func stateCosts(duration time.Duration) {
	for _, name := range []string{"phosphor", "ghostty"} {
		if selected := os.Getenv("COMPARE_ENGINE"); selected != "" && name != selected {
			continue
		}
		t := newEngine(name, 120, 40)
		feed(t, benchdata.Claude(4<<20, 120, 40), 1024)
		for _, op := range []string{"view", "repaint-screen", "repaint-history", "snapshot", "restore"} {
			snapshot := t.Snapshot()
			size := 0
			n := 0
			runtime.GC()
			cpuStart := processCPU()
			start := time.Now()
			for time.Since(start) < duration {
				switch op {
				case "view":
					_ = t.View()
				case "repaint-screen":
					size = len(t.Repaint(false))
				case "repaint-history":
					size = len(t.Repaint(true))
				case "snapshot":
					size = len(t.Snapshot())
				case "restore":
					r := restore(name, snapshot, 120, 40)
					r.Close()
					size = len(snapshot)
				}
				n++
			}
			elapsed := time.Since(start)
			cpu := processCPU() - cpuStart
			emit(map[string]any{"kind": "state-cost", "engine": name, "op": op, "iterations": n, "ns_per_op": float64(elapsed.Nanoseconds()) / float64(n), "cpu_ns_per_op": float64(cpu.Nanoseconds()) / float64(n), "output_bytes": size, "retention": stats(t)})
		}
		t.Close()
	}
}

type Step struct {
	Write  string                    `json:"write"`
	Hex    string                    `json:"write_hex"`
	Resize *struct{ Cols, Rows int } `json:"resize"`
}
type Expected struct {
	Lines []string `json:"lines"`
	X     *int     `json:"cursor_x"`
	Y     *int     `json:"cursor_y"`
	Alt   *bool    `json:"alt_screen"`
}
type Probe struct {
	Name     string   `json:"name"`
	Cols     int      `json:"cols"`
	Rows     int      `json:"rows"`
	Steps    []Step   `json:"steps"`
	Expected Expected `json:"expected"`
	Reason   string   `json:"reason"`
}

func apply(t Engine, steps []Step) {
	for _, s := range steps {
		if s.Resize != nil {
			t.Resize(s.Resize.Cols, s.Resize.Rows)
		} else if s.Hex != "" {
			b, e := hex.DecodeString(s.Hex)
			if e != nil {
				panic(e)
			}
			t.Write(b)
		} else {
			t.Write([]byte(s.Write))
		}
	}
}
func checkProbes(path string) {
	raw, e := os.ReadFile(path)
	if e != nil {
		panic(e)
	}
	var cases []Probe
	if e = json.Unmarshal(raw, &cases); e != nil {
		panic(e)
	}
	for _, c := range cases {
		for _, name := range []string{"phosphor", "ghostty"} {
			t := newEngine(name, c.Cols, c.Rows)
			apply(t, c.Steps)
			v := t.View()
			want := c.Expected
			ok := true
			if want.Lines != nil {
				ok = len(v.Lines) >= len(want.Lines) && slices.Equal(v.Lines[:len(want.Lines)], want.Lines)
			}
			if want.X != nil {
				ok = ok && v.X == *want.X
			}
			if want.Y != nil {
				ok = ok && v.Y == *want.Y
			}
			if want.Alt != nil {
				ok = ok && v.Alt == *want.Alt
			}
			emit(map[string]any{"kind": "probe", "engine": name, "name": c.Name, "pass": ok, "got": v, "expected": want})
			t.Close()
		}
	}
}
func replayChecks() {
	inputs := streams()
	for i := range inputs {
		if strings.Contains(inputs[i].Name, "recording") {
			inputs[i].Cols, inputs[i].Rows = 100, 30
		}
	}
	for _, s := range inputs {
		a := newEngine("phosphor", s.Cols, s.Rows)
		g := newEngine("ghostty", s.Cols, s.Rows)
		feed(a, s.Data, 1024)
		feed(g, s.Data, 1024)
		emit(map[string]any{"kind": "stream-agreement", "stream": s.Name, "pass": same(a.View(), g.View()), "phosphor": a.View(), "ghostty": g.View()})
		a.Close()
		g.Close()
	}
	for _, name := range []string{"phosphor", "ghostty"} {
		for _, s := range inputs {
			for _, fraction := range []int{13, 37, 71, 100} {
				cut := len(s.Data) * fraction / 100
				t := newEngine(name, s.Cols, s.Rows)
				feed(t, s.Data[:cut], 1024)
				snapshot := t.Snapshot()
				r := restore(name, snapshot, s.Cols, s.Rows)
				equalAtRestore := same(t.View(), r.View())
				feed(t, s.Data[cut:], 1024)
				feed(r, s.Data[cut:], 1024)
				emit(map[string]any{"kind": "snapshot-continuation", "engine": name, "stream": s.Name, "percent": fraction, "restored_view_equal": equalAtRestore, "continued_view_equal": same(t.View(), r.View()), "bytes": len(snapshot)})
				t.Close()
				r.Close()
			}
		}
	}
	edge := []struct {
		Name          string
		Before, After []byte
	}{{"mid-csi", []byte("\x1b[3"), []byte("1mX")}, {"mid-osc", []byte("\x1b]0;hello"), []byte(" world\x07X")}, {"mid-utf8", []byte{0xf0, 0x9f}, []byte{0x98, 0x80, 'X'}}, {"inactive-main-screen", []byte("MAIN\x1b[?1049h\x1b[HALT"), []byte("\x1b[?1049l!")}, {"saved-cursor", []byte("\x1b[2;3H\x1b7\x1b[1;1HA"), []byte("\x1b8B")}, {"pending-wrap", []byte("123456789012"), []byte("X")}, {"origin-and-margins", []byte("\x1b[2;4r\x1b[?6h\x1b[1;1HO"), []byte("\x1b[1;1HX")}, {"scroll-clear-outside-origin", []byte("A\x1b[2;4r\x1b[?69h\x1b[3;8s\x1b[?6hB\x1b[22J"), []byte("X")}}
	for _, c := range edge {
		for _, name := range []string{"phosphor", "ghostty"} {
			for _, kind := range []string{"snapshot", "ansi-attach", "ansi-attach-with-continuation"} {
				if kind == "ansi-attach-with-continuation" && name == "phosphor" {
					continue
				}
				t := newEngine(name, 12, 5)
				t.Write(c.Before)
				var r Engine
				var b []byte
				if kind == "snapshot" {
					b = t.Snapshot()
					r = restore(name, b, 12, 5)
				} else {
					b = t.Repaint(true)
					if kind == "ansi-attach-with-continuation" {
						b = append(b, t.(*ghost).continuation()...)
					}
					r = newEngine(name, 12, 5)
					r.Write(b)
				}
				at := same(t.View(), r.View())
				t.Write(c.After)
				r.Write(c.After)
				emit(map[string]any{"kind": kind + "-edge", "engine": name, "name": c.Name, "restored_view_equal": at, "continued_view_equal": same(t.View(), r.View()), "expected": t.View(), "got": r.View(), "bytes": len(b)})
				t.Close()
				r.Close()
			}
		}
	}
}
func memory(name, workload string, n int) {
	fill := benchdata.Logs(512 << 10)
	if workload == "equal-100-history" {
		fill = []byte(strings.Repeat("\x1b[32mINFO\x1b[0m request id=12345 path=/api/v1/items status=200 took=3ms\r\n", 123))
	}
	if workload == "cleared" || workload == "cleared-compact" || workload == "full-window" || workload == "full-window-compact" {
		var grid strings.Builder
		for y := 1; y <= 24; y++ {
			fmt.Fprintf(&grid, "\x1b[%d;1H\x1b[32m%s", y, strings.Repeat("x", 80))
		}
		if strings.HasPrefix(workload, "cleared") {
			grid.WriteString("\x1b[0m\x1b[2J\x1b[Huser@host:~$ ")
		}
		fill = []byte(grid.String())
	}
	if workload == "rich-cleared" || workload == "rich-cleared-compact" {
		var grid strings.Builder
		for i := 0; i < 48; i++ {
			// Keep all targets and clusters live until the erase. Staying on
			// one row excludes history and row count as causes of the result.
			fmt.Fprintf(&grid, "\x1b[1;%dH\x1b]8;;https://example.invalid/%s/%d\x1b\\e%c\x1b]8;;\x1b\\", i+1, strings.Repeat("x", 2048), i, rune(0x300+i))
		}
		grid.WriteString("\x1b[0m\x1b[2J\x1b[Huser@host:~$ ")
		fill = []byte(grid.String())
	}
	terms := make([]Engine, n)
	// Warm shared initialization before taking the baseline. Each invocation is a fresh process.
	warm := newEngine(name, 80, 24)
	warm.Write([]byte("user@host:~$ "))
	warm.Close()
	runtime.GC()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	rss0 := uint64(C.exp_rss())
	for i := range terms {
		t := newEngine(name, 80, 24)
		if workload == "blank" {
		} else if workload == "prompt" || workload == "prompt-compact" {
			t.Write([]byte("user@host:~$ "))
		} else if workload == "colored-prompt" || workload == "colored-prompt-compact" {
			t.Write([]byte("\x1b[32muser@host\x1b[0m:~$ "))
		} else if workload == "saved-cursor" {
			t.Write([]byte("user@host:~$ \x1b7"))
		} else {
			feed(t, fill, 1024)
			if workload == "compressed" && name == "ghostty" {
				var result C.GhosttyTerminalCompressionResult
				must(C.int(C.ghostty_terminal_compress(t.(*ghost).t, C.GHOSTTY_TERMINAL_COMPRESSION_MODE_FULL, &result)))
			}
		}
		if strings.HasSuffix(workload, "-compact") {
			if p, ok := t.(*phos); ok {
				if c, ok := any(p.t).(interface{ Compact() }); ok {
					c.Compact()
				}
			} else {
				var result C.GhosttyTerminalCompressionResult
				must(C.int(C.ghostty_terminal_compress(t.(*ghost).t, C.GHOSTTY_TERMINAL_COMPRESSION_MODE_FULL, &result)))
			}
		}
		terms[i] = t
	}
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&after)
	rss1 := uint64(C.exp_rss())
	compactionApplied := false
	if strings.HasSuffix(workload, "-compact") {
		if p, ok := terms[0].(*phos); ok {
			_, compactionApplied = any(p.t).(interface{ Compact() })
		} else {
			compactionApplied = true
		}
	}
	emit(map[string]any{"kind": "memory", "engine": name, "workload": workload, "compaction_applied": compactionApplied, "n": n, "rss_before": rss0, "rss_after": rss1, "rss_delta_per_terminal": float64(int64(rss1)-int64(rss0)) / float64(n), "go_heap_delta_per_terminal": float64(int64(after.HeapAlloc)-int64(before.HeapAlloc)) / float64(n), "last_view": terms[n-1].View(), "retention": stats(terms[n-1])})
	runtime.KeepAlive(fill)
	runtime.KeepAlive(terms)
	for _, t := range terms {
		t.Close()
	}
}
func main() {
	mode := flag.String("mode", "check", "")
	engine := flag.String("engine", "phosphor", "")
	work := flag.String("workload", "prompt", "")
	n := flag.Int("n", 1000, "")
	duration := flag.Duration("duration", 150*time.Millisecond, "")
	probes := flag.String("probes", "", "")
	traces := flag.String("traces", "ghostty/traces.json", "bounded paired trace fixtures")
	traceChunk := flag.Int("trace-chunk", 0, "split trace Write actions into chunks; zero uses whole action")
	flag.Parse()
	switch *mode {
	case "native-info":
		emit(nativeBuildInfo())
	case "trace":
		if !traceChecks(*traces, *traceChunk) {
			os.Exit(1)
		}
	case "widths":
		scalarWidths(false)
	case "width-table":
		scalarWidths(true)
	case "grapheme-properties":
		nativeGraphemeProperties()
	case "cursor-replies":
		cursorReplies()
	case "layout":
		t := reflect.TypeOf(phosphor.Terminal{})
		fields := make([]map[string]any, 0, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			fields = append(fields, map[string]any{"field": f.Name, "bytes": f.Type.Size(), "offset": f.Offset})
		}
		emit(map[string]any{"kind": "layout", "bytes": t.Size(), "fields": fields})
	case "check":
		checkProbes(*probes)
		replayChecks()
	case "throughput":
		throughput(*duration)
	case "state":
		stateCosts(*duration)
	case "memory":
		memory(*engine, *work, *n)
	default:
		panic(*mode)
	}
}
