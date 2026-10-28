//go:build ghostty

package main

/*
#include "bridge.h"
*/
import "C"

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"unsafe"

	"github.com/2018kguo/phosphor"
)

// This is a closed projection of the headless text model. Differences are
// retained, including blanks, pending wrap, physical cursor and typed colors.
type modelCell struct {
	Text           string `json:"text"`
	Width          int    `json:"width"`
	Fg, Bg         uint32
	Attrs          uint16 `json:"attrs"`
	Underline      uint8  `json:"underline"`
	UnderlineColor uint32 `json:"underline_color"`
	URI            string `json:"uri"`
}
type modelCursor struct {
	X, Y, Shape                 int
	Visible, Blink, PendingWrap bool
}
type modelState struct {
	Cols, Rows     int
	Cursor         modelCursor
	Alt            bool
	Modes          [24]bool
	Mouse          uint16
	MouseEncoding  uint16
	Kitty          uint16
	Title          string
	Screen         [][]modelCell
	Wrapped        []bool
	Prompt         []int
	History        [][]modelCell
	HistoryWrapped []bool
	Replies        string
}
type tracedEngine struct {
	Engine
	replies       bytes.Buffer
	native        *C.ExpReplies
	replyBase     string
	sharedPalette bool
}

func newTracedEngine(name string, cols, rows int) *tracedEngine {
	t := &tracedEngine{}
	if name == "phosphor" {
		t.Engine = &phos{phosphor.New(cols, rows, phosphor.WithReply(&t.replies))}
	} else {
		t.Engine = newEngine(name, cols, rows)
		t.native = C.exp_replies_new(t.Engine.(*ghost).t)
		if t.native == nil {
			t.Engine.Close()
			panic("native reply sink")
		}
	}
	return t
}
func (t *tracedEngine) Close() { t.Engine.Close(); C.exp_replies_free(t.native) }

// A fixture can supply the same host palette to both engines before input.
// This tests color reset behavior without treating different host defaults
// as a protocol failure. Captured cells and reply bytes remain unchanged.
func (g *ghost) sharedHostPalette() {
	p := phosphor.DefaultPalette()
	rgb := func(c phosphor.RGB) C.GhosttyColorRgb {
		return C.GhosttyColorRgb{r: C.uint8_t(c.R), g: C.uint8_t(c.G), b: C.uint8_t(c.B)}
	}
	fg, bg, cursor := rgb(p.Foreground), rgb(p.Background), rgb(p.Cursor)
	var colors [256]C.GhosttyColorRgb
	for i, color := range p.Colors {
		colors[i] = rgb(color)
	}
	must(C.int(C.ghostty_terminal_set(g.t, C.GHOSTTY_TERMINAL_OPT_COLOR_FOREGROUND, unsafe.Pointer(&fg))))
	must(C.int(C.ghostty_terminal_set(g.t, C.GHOSTTY_TERMINAL_OPT_COLOR_BACKGROUND, unsafe.Pointer(&bg))))
	must(C.int(C.ghostty_terminal_set(g.t, C.GHOSTTY_TERMINAL_OPT_COLOR_CURSOR, unsafe.Pointer(&cursor))))
	must(C.int(C.ghostty_terminal_set(g.t, C.GHOSTTY_TERMINAL_OPT_COLOR_PALETTE, unsafe.Pointer(&colors[0]))))
}

func (g *ghost) mouseReport(action int, pressed bool, tracking, format int) ([]byte, error) {
	var out [64]byte
	var n C.size_t
	if rc := C.exp_mouse_report(g.t, C.int(action), C.bool(pressed), C.int(tracking), C.int(format), (*C.uint8_t)(unsafe.Pointer(&out[0])), C.size_t(len(out)), &n); rc != 0 {
		return nil, fmt.Errorf("native mouse encode: %d", int(rc))
	}
	return out[:int(n)], nil
}
func phosCell(c phosphor.Cell) modelCell {
	return modelCell{Text: c.Text(), Width: c.Width, Fg: uint32(c.Style.Fg), Bg: uint32(c.Style.Bg), Attrs: uint16(c.Style.Attrs), Underline: uint8(c.Style.Underline), UnderlineColor: uint32(c.Style.UnderlineColor), URI: c.Link.URI}
}
func (t *tracedEngine) state() (modelState, error) {
	if p, ok := t.Engine.(*phos); ok {
		cols, rows := p.t.Size()
		cur := p.t.Cursor()
		m := p.t.Modes()
		shape := int(cur.Shape)
		if shape == 0 {
			shape = 1
		} // host default is a block in both constructors
		s := modelState{Cols: cols, Rows: rows, Cursor: modelCursor{cur.X, cur.Y, shape, cur.Visible, cur.Blink, cur.PendingWrap}, Alt: m.AltScreen, Kitty: m.KittyKeyboard, Title: p.t.Title(), Replies: t.replies.String(), Mouse: uint16(m.Mouse), MouseEncoding: uint16(m.MouseEncoding)}
		s.Modes = [24]bool{m.ApplicationCursorKeys, m.ApplicationKeypad, m.BracketedPaste, m.FocusEvents, m.Autowrap, m.Insert, m.Newline, m.Origin, m.ReverseVideo, m.SyncUpdate,
			m.MouseTrackingModes&phosphor.MouseTrackingX10 != 0, m.MouseTrackingModes&phosphor.MouseTrackingNormal != 0,
			m.MouseTrackingModes&phosphor.MouseTrackingButtonEvent != 0, m.MouseTrackingModes&phosphor.MouseTrackingAnyEvent != 0,
			m.MouseEncodingModes&phosphor.MouseEncodingModeUTF8 != 0, m.MouseEncodingModes&phosphor.MouseEncodingModeSGR != 0,
			m.MouseEncodingModes&phosphor.MouseEncodingModeURXVT != 0, m.MouseEncodingModes&phosphor.MouseEncodingModeSGRPixels != 0,
			m.Columns132, m.AllowColumnSwitching, m.SmoothScrolling,
			m.AltScreenModes&phosphor.AltScreenMode47 != 0, m.AltScreenModes&phosphor.AltScreenMode1047 != 0, m.AltScreenModes&phosphor.AltScreenMode1049 != 0}
		for y := 0; y < rows; y++ {
			row := make([]modelCell, cols)
			for x := range row {
				row[x] = phosCell(p.t.Cell(x, y))
			}
			s.Screen = append(s.Screen, row)
			s.Wrapped = append(s.Wrapped, p.t.Wrapped(y))
			s.Prompt = append(s.Prompt, int(p.t.Prompt(y)))
		}
		if !m.AltScreen {
			for i := 0; i < p.t.ScrollbackLen(); i++ {
				for j, row := range p.t.ScrollbackCells(i) {
					cells := make([]modelCell, len(row))
					for x, c := range row {
						cells[x] = phosCell(c)
					}
					s.History = append(s.History, cells)
					s.HistoryWrapped = append(s.HistoryWrapped, j > 0)
				}
			}
		}
		s.Replies = t.replyBase + s.Replies
		return s, nil
	}
	var raw C.ExpState
	if rc := C.exp_capture(t.Engine.(*ghost).t, &raw); rc != 0 {
		return modelState{}, fmt.Errorf("native capture: %d", int(rc))
	}
	defer C.exp_capture_free(&raw)
	i := raw.info
	s := modelState{Cols: int(i.cols), Rows: int(i.rows), Cursor: modelCursor{int(i.x), int(i.y), int(raw.cursor_shape), bool(i.visible), bool(raw.cursor_blink), bool(i.pending_wrap)}, Alt: bool(i.alt), Kitty: uint16(raw.kitty), Mouse: uint16(raw.mouse), MouseEncoding: uint16(raw.mouse_encoding)}
	if raw.title.len > 0 {
		s.Title = string(C.GoBytes(unsafe.Pointer(raw.title.ptr), C.int(raw.title.len)))
	}
	for n, v := range raw.modes {
		s.Modes[n] = bool(v)
	}
	cells := unsafe.Slice(raw.cells, int(raw.total_rows)*s.Cols)
	wraps := unsafe.Slice(raw.wraps, int(raw.total_rows))
	prompts := unsafe.Slice(raw.prompts, int(raw.total_rows))
	for y := 0; y < int(raw.total_rows); y++ {
		row := make([]modelCell, s.Cols)
		for x := range row {
			c := &cells[y*s.Cols+x]
			rs := make([]rune, int(c.text_len))
			for n := range rs {
				rs[n] = rune(c.text[n])
			}
			row[x] = modelCell{Text: string(rs), Width: int(c.width), Fg: uint32(c.fg), Bg: uint32(c.bg), Attrs: uint16(c.attrs), Underline: uint8(c.underline), UnderlineColor: uint32(c.underline_color), URI: string(C.GoBytes(unsafe.Pointer(&c.uri[0]), C.int(c.uri_len)))}
		}
		if y < int(raw.history_rows) {
			s.History = append(s.History, row)
			s.HistoryWrapped = append(s.HistoryWrapped, wraps[y] != 0)
		} else {
			s.Screen = append(s.Screen, row)
			s.Wrapped = append(s.Wrapped, wraps[y] != 0)
			s.Prompt = append(s.Prompt, int(prompts[y]))
		}
	}
	if t.native != nil {
		if t.native.error != 0 {
			return s, fmt.Errorf("native reply sink: %d", t.native.error)
		}
		s.Replies = t.replyBase + string(C.GoBytes(unsafe.Pointer(t.native.data), C.int(t.native.len)))
	}
	return s, nil
}

type traceAction struct {
	Kind string `json:"kind"`
	Data string `json:"data,omitempty"`
	Hex  string `json:"hex,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}
type traceCase struct {
	Name          string        `json:"name"`
	Feature       string        `json:"feature"`
	Scope         string        `json:"scope"` // parity, identity, deliberate, or limit
	Source        string        `json:"source"`
	Cols          int           `json:"cols"`
	Rows          int           `json:"rows"`
	Actions       []traceAction `json:"actions"`
	SharedPalette bool          `json:"shared_palette,omitempty"`
}

func decodeTraces(r io.Reader) ([]traceCase, error) {
	input, err := io.ReadAll(io.LimitReader(r, (2<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(input) > 2<<20 {
		return nil, fmt.Errorf("trace file exceeds 2 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(input))
	d.DisallowUnknownFields()
	var cases []traceCase
	if err := d.Decode(&cases); err != nil {
		return nil, err
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("trailing JSON")
	}
	if len(cases) == 0 || len(cases) > 512 {
		return nil, fmt.Errorf("trace count must be 1..512")
	}
	seen := map[string]bool{}
	for _, c := range cases {
		if c.Name == "" || seen[c.Name] || c.Feature == "" || c.Source == "" {
			return nil, fmt.Errorf("missing metadata or duplicate name: %q", c.Name)
		}
		seen[c.Name] = true
		if c.Scope != "parity" && c.Scope != "identity" && c.Scope != "deliberate" && c.Scope != "limit" {
			return nil, fmt.Errorf("%s: invalid scope", c.Name)
		}
		if !traceDimensions(c.Cols, c.Rows) || len(c.Actions) == 0 || len(c.Actions) > 256 {
			return nil, fmt.Errorf("%s: invalid geometry/action count", c.Name)
		}
		bytes := 0
		for _, a := range c.Actions {
			switch a.Kind {
			case "write":
				if a.Cols != 0 || a.Rows != 0 || (a.Data != "" && a.Hex != "") {
					return nil, fmt.Errorf("%s: invalid write", c.Name)
				}
				if a.Hex != "" {
					b, err := hex.DecodeString(a.Hex)
					if err != nil {
						return nil, err
					}
					bytes += len(b)
				} else {
					bytes += len(a.Data)
				}
			case "resize":
				if !traceDimensions(a.Cols, a.Rows) || a.Data != "" || a.Hex != "" {
					return nil, fmt.Errorf("%s: invalid resize", c.Name)
				}
			case "compact", "snapshot", "repaint":
				if a.Cols != 0 || a.Rows != 0 || a.Data != "" || a.Hex != "" {
					return nil, fmt.Errorf("%s: unexpected action fields", c.Name)
				}
			default:
				return nil, fmt.Errorf("%s: unknown action %q", c.Name, a.Kind)
			}
		}
		if bytes > 1<<20 {
			return nil, fmt.Errorf("%s: input exceeds 1 MiB", c.Name)
		}
	}
	return cases, nil
}
func traceDimensions(c, r int) bool { return c >= 1 && c <= 256 && r >= 1 && r <= 128 && c*r <= 32768 }

const pinnedGhosttyCommit = "46e9dc9a927b2e7e7cba79e6902c176127dbdf2c"

func nativeBuildInfo() map[string]any {
	var version, build C.GhosttyString
	must(C.int(C.ghostty_build_info(C.GHOSTTY_BUILD_INFO_VERSION_STRING, unsafe.Pointer(&version))))
	must(C.int(C.ghostty_build_info(C.GHOSTTY_BUILD_INFO_VERSION_BUILD, unsafe.Pointer(&build))))
	return map[string]any{"version": string(C.GoBytes(unsafe.Pointer(version.ptr), C.int(version.len))), "build": string(C.GoBytes(unsafe.Pointer(build.ptr), C.int(build.len)))}
}

type modelDifference struct {
	Field             string `json:"field"`
	Phosphor, Ghostty any
}

func differences(a, b modelState) []modelDifference {
	av, bv := reflect.ValueOf(a), reflect.ValueOf(b)
	typ := av.Type()
	var out []modelDifference
	for n := 0; n < av.NumField(); n++ {
		if !reflect.DeepEqual(av.Field(n).Interface(), bv.Field(n).Interface()) {
			// Preserve exact differing cells while keeping bounded reports readable.
			name := typ.Field(n).Name
			if name == "Screen" || name == "History" {
				x, y := av.Field(n).Interface().([][]modelCell), bv.Field(n).Interface().([][]modelCell)
				if len(x) != len(y) {
					out = append(out, modelDifference{name + ".rows", len(x), len(y)})
					continue
				}
				for row := range x {
					if len(x[row]) != len(y[row]) {
						out = append(out, modelDifference{fmt.Sprintf("%s[%d].cols", name, row), len(x[row]), len(y[row])})
						continue
					}
					for col := range x[row] {
						if !reflect.DeepEqual(x[row][col], y[row][col]) {
							out = append(out, modelDifference{fmt.Sprintf("%s[%d][%d]", name, row, col), x[row][col], y[row][col]})
							if len(out) >= 12 {
								return out
							}
						}
					}
				}
				continue
			}
			out = append(out, modelDifference{name, av.Field(n).Interface(), bv.Field(n).Interface()})
		}
	}
	return out
}
func (t *tracedEngine) replace(data []byte, cols, rows int, snapshot bool) {
	old, e := t.state()
	if e != nil {
		panic(e)
	}
	name := "ghostty"
	if _, ok := t.Engine.(*phos); ok {
		name = "phosphor"
	}
	var engine Engine
	if name == "phosphor" {
		p := phosphor.New(cols, rows, phosphor.WithReply(&t.replies))
		engine = &phos{p}
		t.replies.Reset()
		if snapshot {
			if err := p.RestoreSnapshot(data, phosphor.SnapshotLimits{}); err != nil {
				panic(err)
			}
		} else {
			engine.Write(data)
		}
		t.replies.Reset()
	} else {
		if snapshot {
			engine = restore(name, data, cols, rows)
		} else {
			engine = newEngine(name, cols, rows)
			if t.sharedPalette {
				engine.(*ghost).sharedHostPalette()
			}
			engine.Write(data)
		}
	}
	t.Engine.Close()
	C.exp_replies_free(t.native)
	t.native = nil
	t.Engine = engine
	t.replyBase = old.Replies
	if name == "ghostty" {
		t.native = C.exp_replies_new(engine.(*ghost).t)
		if t.native == nil {
			panic("restored native reply sink")
		}
	}
}
func traceWrite(t *tracedEngine, data []byte, chunk int) {
	if chunk <= 0 {
		t.Write(data)
		return
	}
	for len(data) > 0 {
		n := min(chunk, len(data))
		t.Write(data[:n])
		data = data[n:]
	}
}
func traceChecks(path string, chunk int) bool {
	if chunk < 0 || chunk > 1<<20 {
		panic("trace chunk must be 0..1048576")
	}
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	cases, err := decodeTraces(f)
	if err != nil {
		panic(err)
	}
	ok := true
	totals := map[string][2]int{}
	directTotals, phosphorTotals, ghosttyTotals := map[string][2]int{}, map[string][2]int{}, map[string][2]int{}
	tally := func(totals map[string][2]int, scope string, pass bool) {
		counts := totals[scope]
		counts[1]++
		if pass {
			counts[0]++
		}
		totals[scope] = counts
	}
	for _, c := range cases {
		a, b := newTracedEngine("phosphor", c.Cols, c.Rows), newTracedEngine("ghostty", c.Cols, c.Rows)
		controlA, controlB := newTracedEngine("phosphor", c.Cols, c.Rows), newTracedEngine("ghostty", c.Cols, c.Rows)
		if c.SharedPalette {
			for _, t := range []*tracedEngine{a, b, controlA, controlB} {
				t.sharedPalette = true
				if g, ok := t.Engine.(*ghost); ok {
					g.sharedHostPalette()
				}
			}
		}
		for n, action := range c.Actions {
			var local []modelDifference
			switch action.Kind {
			case "write":
				data := []byte(action.Data)
				if action.Hex != "" {
					data, _ = hex.DecodeString(action.Hex)
				}
				for _, t := range []*tracedEngine{a, b, controlA, controlB} {
					traceWrite(t, data, chunk)
				}
			case "resize":
				for _, t := range []*tracedEngine{a, b, controlA, controlB} {
					t.Resize(action.Cols, action.Rows)
				}
			case "compact":
				before, e := a.state()
				if e != nil {
					panic(e)
				}
				a.Engine.(*phos).t.Compact()
				after, e := a.state()
				if e != nil {
					panic(e)
				}
				local = differences(before, after)
				for i := range local {
					local[i].Field = "phosphor.compact." + local[i].Field
				}
			case "snapshot", "repaint":
				for _, t := range []*tracedEngine{a, b} {
					before, e := t.state()
					if e != nil {
						panic(e)
					}
					var data []byte
					if action.Kind == "snapshot" {
						data = t.Snapshot()
					} else {
						data = t.Repaint(true)
					}
					t.replace(data, before.Cols, before.Rows, action.Kind == "snapshot")
					after, e := t.state()
					if e != nil {
						panic(e)
					}
					engine := "ghostty"
					if t == a {
						engine = "phosphor"
					}
					for _, diff := range differences(before, after) {
						diff.Field = engine + "." + action.Kind + "." + diff.Field
						local = append(local, diff)
					}
				}
			}
			x, e := a.state()
			if e != nil {
				panic(e)
			}
			y, e := b.state()
			if e != nil {
				panic(e)
			}
			diff := append(local, differences(x, y)...)
			phosphorPass, ghosttyPass := true, true
			for _, d := range local {
				if strings.HasPrefix(d.Field, "ghostty.") {
					ghosttyPass = false
				} else {
					phosphorPass = false
				}
			}
			var controls [2]modelState
			for i, t := range []*tracedEngine{controlA, controlB} {
				control, e := t.state()
				if e != nil {
					panic(e)
				}
				controls[i] = control
				actual := x
				engine := "phosphor"
				if i == 1 {
					actual = y
					engine = "ghostty"
				}
				for _, d := range differences(control, actual) {
					if i == 0 {
						phosphorPass = false
					} else {
						ghosttyPass = false
					}
					d.Field = engine + ".continuation." + d.Field
					diff = append(diff, d)
				}
			}
			directDiff := differences(controls[0], controls[1])
			directPass := len(directDiff) == 0
			pass := len(diff) == 0
			if c.Scope == "parity" && !pass {
				ok = false
			}
			tally(totals, c.Scope, pass)
			tally(directTotals, c.Scope, directPass)
			tally(phosphorTotals, c.Scope, phosphorPass)
			tally(ghosttyTotals, c.Scope, ghosttyPass)
			emit(map[string]any{"kind": "trace", "case": c.Name, "feature": c.Feature, "scope": c.Scope, "source": c.Source, "chunk": chunk, "step": n, "action": action, "pass": pass, "differences": diff,
				"direct_pass": directPass, "direct_differences": directDiff, "phosphor_continuation_pass": phosphorPass, "ghostty_continuation_pass": ghosttyPass,
				"phosphor_replies_hex": hex.EncodeToString([]byte(x.Replies)), "ghostty_replies_hex": hex.EncodeToString([]byte(y.Replies))})
		}
		a.Close()
		b.Close()
		controlA.Close()
		controlB.Close()
	}
	emit(map[string]any{"kind": "trace_summary", "native_build": nativeBuildInfo(), "chunk": chunk, "cases": len(cases), "steps_by_scope_pass_total": totals,
		"direct_steps_by_scope_pass_total": directTotals, "phosphor_continuation_steps_by_scope_pass_total": phosphorTotals, "ghostty_continuation_steps_by_scope_pass_total": ghosttyTotals,
		"parity_pass": ok, "coverage": "bounded named traces; not a terminal compliance percentage"})
	return ok
}
func scalarWidths(table bool) {
	widths := make([]byte, 0x110000)
	C.exp_widths((*C.uint8_t)(unsafe.Pointer(&widths[0])), C.size_t(len(widths)))
	start := -1
	var pw, nw int
	flush := func(end int) {
		if start >= 0 {
			kind := "scalar_width_difference"
			if table {
				kind = "native_scalar_width_range"
			}
			emit(map[string]any{"kind": kind, "first": start, "last": end, "phosphor_width": pw, "ghostty_width": nw})
			start = -1
		}
	}
	count := 0
	for r := 0; r < len(widths); r++ {
		if r >= 0xd800 && r <= 0xdfff {
			flush(r - 1)
			continue
		}
		p, n := phosphor.CodepointWidth(rune(r)), int(widths[r])
		if table {
			p = 1
		}
		if p == n {
			flush(r - 1)
			continue
		}
		count++
		if start >= 0 && (p != pw || n != nw) {
			flush(r - 1)
		}
		if start < 0 {
			start, pw, nw = r, p, n
		}
	}
	flush(0x10ffff)
	if table {
		build := nativeBuildInfo()
		// libghostty currently hardcodes the library version without a commit.
		// Verify the explicitly supplied source checkout and record the library's
		// self-reported version separately; do not invent runtime provenance.
		source := os.Getenv("GHOSTTY_SOURCE")
		if source == "" {
			panic("width-table requires GHOSTTY_SOURCE pointing at the linked source checkout")
		}
		commit, err := exec.Command("git", "-C", source, "rev-parse", "HEAD").Output()
		if err != nil {
			panic(err)
		}
		if strings.TrimSpace(string(commit)) != pinnedGhosttyCommit {
			panic("width-table source does not match pinned Ghostty commit")
		}
		if err := exec.Command("git", "-C", source, "diff", "--quiet", "HEAD", "--", "src/terminal", "build.zig", "build.zig.zon").Run(); err != nil {
			panic("width-table source has modified terminal/build code")
		}
		emit(map[string]any{"kind": "native_scalar_width_summary", "native_build": build, "ghostty_commit": pinnedGhosttyCommit, "unicode_version": "17.0.0", "unicode_scalars": 0x110000 - 0x800, "non_default_scalars": count, "default_width": 1, "first": 0, "last": 0x10ffff, "excluded_first": 0xd800, "excluded_last": 0xdfff, "mode": "native-codepoint-width", "basis": "batched ghostty_unicode_codepoint_width; no grapheme folding"})
	} else {
		emit(map[string]any{"kind": "scalar_width_summary", "unicode_scalars": 0x110000 - 0x800, "different": count, "basis": "phosphor.CodepointWidth vs native Ghostty scalar API; excludes surrogates; does not test terminal grapheme behavior"})
	}
}

// Prepend U+0600 joins the following printable scalar. Keeping consumed count
// distinguishes width-neutral continuation from a control-induced boundary.
func nativeGraphemeProperties() {
	values := make([]byte, 0x110000)
	C.exp_grapheme_properties((*C.uint8_t)(unsafe.Pointer(&values[0])), C.size_t(len(values)))
	start := -1
	var current byte
	neutral := 0
	flush := func(last int) {
		if start >= 0 {
			emit(map[string]any{"kind": "native_grapheme_property_range", "first": start, "last": last, "consumed": int(current >> 4), "width": int(current & 15)})
			start = -1
		}
	}
	for r, value := range values {
		if r >= 0xd800 && r <= 0xdfff {
			flush(r - 1)
			continue
		}
		if value == 0x21 {
			neutral++
		}
		if start >= 0 && value != current {
			flush(r - 1)
		}
		if start < 0 {
			start, current = r, value
		}
	}
	flush(0x10ffff)
	emit(map[string]any{"kind": "native_grapheme_property_summary", "ghostty_commit": pinnedGhosttyCommit, "native_build": nativeBuildInfo(), "unicode_version": "17.0.0", "prefix_scalar": 0x600, "unicode_scalars": 0x110000 - 0x800, "first": 0, "last": 0x10ffff, "excluded_first": 0xd800, "excluded_last": 0xdfff, "neutral_join_scalars": neutral, "mode": "native-grapheme-width-prepend-prefix", "basis": "batched ghostty_unicode_grapheme_width([U+0600, scalar]); ranges retain consumed count and folded width; controls unspecified per native API"})
}

// Keep raw replies independently runnable while evolving the wider trace model.
func cursorReplies() {
	inputs := []string{"", "\x1b[?12h", "\x1b[?12l", "\x1b[1 q", "\x1b[2 q", "\x1b[3 q", "\x1b[4 q", "\x1b[5 q", "\x1b[6 q", "\x1b[7 q", "\x1b[65536 q", "\x1b[5 q\x1b[!p", "\x1b[5 q\x1bc", "\x1b[?12h\x1b[?12s\x1b[?12l\x1b[?12r"}
	for _, input := range inputs {
		for _, engine := range []string{"phosphor", "ghostty"} {
			t := newTracedEngine(engine, 12, 4)
			t.Write([]byte(input + "\x1bP$q q\x1b\\\x1b[?12$p"))
			var replies string
			if t.native != nil {
				replies = string(C.GoBytes(unsafe.Pointer(t.native.data), C.int(t.native.len)))
			} else {
				replies = t.replies.String()
			}
			emit(map[string]any{"kind": "cursor_reply", "engine": engine, "input_hex": hex.EncodeToString([]byte(input)), "replies_hex": hex.EncodeToString([]byte(replies)), "replies": strings.ReplaceAll(replies, "\x1b", "<ESC>")})
			t.Close()
		}
	}
}
