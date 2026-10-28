//go:build ghostty

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/2018kguo/phosphor"
)

func TestNativeMouseBitsAndEffectiveSelection(t *testing.T) {
	p := newTracedEngine("phosphor", 12, 3)
	g := newTracedEngine("ghostty", 12, 3)
	defer p.Close()
	defer g.Close()
	for _, input := range []string{
		"\x1b[?1000h\x1b[?9h", // last-set wins, even when the earlier mode is more capable
		"\x1b[?1003l",         // resetting an inactive bit clears the effective mode
		"\x1b[?1002;1003;1006;1015;1005;1016h",
		"\x1b[?1006l", // the remaining encoding bits do not choose a fallback
		"\x1b[?9;1000;1002;1003;1005;1006;1015;1016s",
		"\x1b[?9;1000;1002;1003;1005;1006;1015;1016l",
		"\x1b[?1003;1000;9;1002;1016;1005;1006;1015r",
		"\x1b[?9$p\x1b[?1000$p\x1b[?1002$p\x1b[?1003$p\x1b[?1005$p\x1b[?1006$p\x1b[?1015$p\x1b[?1016$p",
		"\x1b[!p",
		"\x1bc",
	} {
		p.Write([]byte(input))
		g.Write([]byte(input))
		ps, gs := traceState(t, p), traceState(t, g)
		if diff := differences(ps, gs); len(diff) > 0 {
			t.Fatalf("after %q: %+v", input, diff)
		}
		t.Logf("input=%q bits=%v tracking=%d encoding=%d replies=%q", input, gs.Modes[10:], gs.Mouse, gs.MouseEncoding, gs.Replies)
	}
}

func TestNativeEffectiveMouseProbeHasDistinctOutputs(t *testing.T) {
	e := newTracedEngine("ghostty", 12, 3)
	defer e.Close()
	g := e.Engine.(*ghost)
	seen := map[string]bool{}
	for format := 0; format < 5; format++ {
		report, err := g.mouseReport(0, false, 2, format)
		if err != nil {
			t.Fatal(err)
		}
		if seen[string(report)] {
			t.Fatalf("format %d cannot be distinguished: %x", format, report)
		}
		seen[string(report)] = true
		t.Logf("native format %d: %x", format, report)
	}
	for tracking, wantPattern := range []uint8{0, 1, 3, 7, 15} {
		var pattern uint8
		for n := 0; n < 4; n++ {
			action := min(n, 2)
			report, err := g.mouseReport(action, n == 2, tracking, 2)
			if err != nil {
				t.Fatal(err)
			}
			if len(report) != 0 {
				pattern |= 1 << n
			}
		}
		if pattern != wantPattern {
			t.Fatalf("native tracking %d has pattern %d, want %d", tracking, pattern, wantPattern)
		}
	}
}

func TestNativeGraphemeCapacityIs64SuffixCodepoints(t *testing.T) {
	for _, suffix := range []string{"\u0301", "\U000E0100"} {
		for _, count := range []int{15, 16, 63, 64, 65, 80} {
			e := newTracedEngine("ghostty", 8, 3)
			e.Write([]byte("A" + strings.Repeat(suffix, count)))
			s := traceState(t, e)
			e.Close()
			want := "A" + strings.Repeat(suffix, min(count, 64))
			if got := s.Screen[0][0].Text; got != want {
				t.Fatalf("suffix=%U count=%d got %d bytes want %d", []rune(suffix)[0], count, len(got), len(want))
			}
			if s.Cursor.X != 1 || s.Cursor.Y != 0 || s.Screen[0][1].Text != "" {
				t.Fatal("excess suffix codepoints moved the cursor or became standalone cells", s.Cursor)
			}
			t.Logf("suffix=%U count=%d retained=%d codepoints %d bytes", []rune(suffix)[0], count, min(count, 64)+1, len(want))
		}
	}
}

func traceState(t *testing.T, engine *tracedEngine) modelState {
	t.Helper()
	s, err := engine.state()
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestNativeCapturePreservesCellSemantics(t *testing.T) {
	for _, name := range []string{"phosphor", "ghostty"} {
		t.Run(name, func(t *testing.T) {
			e := newTracedEngine(name, 8, 3)
			defer e.Close()
			e.Write([]byte("\x1b[1;2;3;5;7;8;9;53;4:3;38;5;123;48;2;1;2;3m\x1b]8;;https://example.com\x1b\\界"))
			s := traceState(t, e)
			c := s.Screen[0][0]
			if c.Text != "界" || c.Width != 2 || c.Fg != uint32(phosphor.IndexedColor(123)) || c.Bg != uint32(phosphor.RGBColor(1, 2, 3)) || c.Attrs != 255 || c.Underline != 3 || c.URI != "https://example.com" {
				t.Fatalf("head=%+v", c)
			}
			if tail := s.Screen[0][1]; tail.Width != 0 || tail.Text != "" {
				t.Fatalf("tail=%+v", tail)
			}
			if blank := s.Screen[0][2]; blank.Width != 1 || blank.Text != "" || blank.Fg != 0 {
				t.Fatalf("blank=%+v", blank)
			}
		})
	}
}
func TestNativeCapturePendingWrapAndExactCursor(t *testing.T) {
	for _, name := range []string{"phosphor", "ghostty"} {
		t.Run(name, func(t *testing.T) {
			e := newTracedEngine(name, 4, 3)
			defer e.Close()
			e.Write([]byte("abcd"))
			s := traceState(t, e)
			if s.Cursor.X != 3 || s.Cursor.Y != 0 || !s.Cursor.PendingWrap {
				t.Fatal(s.Cursor)
			}
			e.Write([]byte("E"))
			s = traceState(t, e)
			if !s.Wrapped[1] || s.Cursor.X != 1 || s.Cursor.Y != 1 {
				t.Fatalf("wrap=%v cursor=%+v", s.Wrapped, s.Cursor)
			}
		})
	}
}
func TestSnapshotReplyBinding(t *testing.T) {
	for _, name := range []string{"phosphor", "ghostty"} {
		t.Run(name, func(t *testing.T) {
			e := newTracedEngine(name, 8, 3)
			defer e.Close()
			e.Write([]byte("AB\x1b[6n"))
			before := traceState(t, e)
			e.replace(e.Snapshot(), 8, 3, true)
			if diff := differences(before, traceState(t, e)); len(diff) > 0 {
				t.Fatal(diff)
			}
			e.Write([]byte("\x1b[6n"))
			after := traceState(t, e)
			if after.Replies != before.Replies+before.Replies {
				t.Fatalf("reply binding lost/duplicated: %q", after.Replies)
			}
		})
	}
}

func TestNativePendingWrapWithoutAutowrap(t *testing.T) {
	for _, test := range []struct{ name, input string }{
		{"grapheme-widening", "\x1b[?2027h\x1b[?7lA❤️"},
		{"grapheme-saved-pending", "\x1b[?2027h\x1b[?7lA❤️\x1b7\x1b[HX\x1b8"},
		{"ordinary-saved-pending", "abc\x1b7\x1b[?7l\x1b[HX\x1b8"},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := newTracedEngine("ghostty", 3, 3)
			defer e.Close()
			e.Write([]byte(test.input))
			s := traceState(t, e)
			t.Logf("cursor=%+v autowrap=%v", s.Cursor, s.Modes[4])
			if s.Cursor.X != 2 || s.Cursor.Y != 0 || !s.Cursor.PendingWrap || s.Modes[4] {
				t.Fatal("native pending-wrap/DECAWM expectation changed")
			}
		})
	}
}

func TestNativeREPMapsInputScalarUsingCurrentCharset(t *testing.T) {
	for _, test := range []struct{ input, want string }{{"\x1b(0q\x1b(B\x1b[2b", "─qq"}, {"q\x1b(0\x1b[2b", "q──"}} {
		e := newTracedEngine("ghostty", 8, 3)
		e.Write([]byte(test.input))
		s := traceState(t, e)
		e.Close()
		var text string
		for _, c := range s.Screen[0] {
			text += c.Text
		}
		t.Logf("input=%x native=%q", test.input, text)
		if text != test.want {
			t.Fatalf("want %q got %q", test.want, text)
		}
	}
}

func TestNativeModifierRequiresImmediateBase(t *testing.T) {
	for _, prefix := range []string{
		"👋\x1b[?2027h", "👋\u200b\x1b[?2027h", "👋\u0301\x1b[?2027h",
		"👋\ufe0f\x1b[?2027h", "👋\u200d\x1b[?2027h", "👋\u0600\x1b[?2027h", "👋\x00\x1b[?2027h",
		"✍\ufe0f\x1b[?2027h", "\x1b[?2027h✍\ufe0f", "\x1b[?2027h👋\u0301", "\x1b[?2027h👋🏿",
	} {
		t.Run(fmt.Sprintf("prefix-%x", prefix), func(t *testing.T) {
			phos := newTracedEngine("phosphor", 12, 3)
			defer phos.Close()
			native := newTracedEngine("ghostty", 12, 3)
			defer native.Close()
			for _, part := range []string{prefix, "🏿X"} {
				phos.Write([]byte(part))
				native.Write([]byte(part))
				p, n := traceState(t, phos), traceState(t, native)
				if diff := differences(p, n); len(diff) != 0 {
					t.Fatalf("native modifier context: %+v", diff)
				}
			}
		})
	}
}

func TestNativeTabsResetOnlyWhenColumnsChange(t *testing.T) {
	e := newTracedEngine("ghostty", 20, 3)
	defer e.Close()
	e.Write([]byte("\x1b[3g\t"))
	if s := traceState(t, e); s.Cursor.X != 19 {
		t.Fatal(s.Cursor)
	}
	e.Write([]byte("\x1b[5G\x1bH\r\t"))
	if s := traceState(t, e); s.Cursor.X != 4 {
		t.Fatal(s.Cursor)
	}
	e.Resize(20, 4)
	e.Write([]byte("\r\t"))
	if s := traceState(t, e); s.Cursor.X != 4 {
		t.Fatal(s.Cursor)
	}
	e.Resize(30, 4)
	e.Write([]byte("\r\t"))
	if s := traceState(t, e); s.Cursor.X != 8 {
		t.Fatal(s.Cursor)
	}
	e.Resize(10, 4)
	e.Write([]byte("\r\t"))
	if s := traceState(t, e); s.Cursor.X != 8 {
		t.Fatal(s.Cursor)
	}
	t.Log("clear all => HT19; custom tab4 => HT4; height-only resize preserves4; width grow30/shrink10 restore defaults => HT8")
}

func TestNativeOSC2CommitsOnEscapeBeforeST(t *testing.T) {
	for _, suffix := range []string{"\\", "[31mX", "X"} {
		t.Run(fmt.Sprintf("suffix-%x", suffix), func(t *testing.T) {
			e := newTracedEngine("ghostty", 8, 3)
			defer e.Close()
			e.Write([]byte("\x1b]2;old\a\x1b]2;new\x1b"))
			s := traceState(t, e)
			if s.Title != "new" {
				t.Fatalf("title before ST=%q", s.Title)
			}
			e.Write([]byte(suffix))
			s = traceState(t, e)
			var text string
			for _, c := range s.Screen[0] {
				text += c.Text
			}
			t.Logf("suffix=%x title=%q text=%q", suffix, s.Title, text)
			if s.Title != "new" {
				t.Fatal(s.Title)
			}
		})
	}
}

func TestNativeCPRClampsOutsideOrigin(t *testing.T) {
	for _, input := range []string{
		"A\x1b[2;3r\x1b[?6hB\x1b[22J\x1b[6n",
		"A\x1b[2;3r\x1b[?69h\x1b[3;6s\x1b[?6hB\x1b[22J\x1b[6n",
	} {
		e := newTracedEngine("ghostty", 8, 4)
		e.Write([]byte(input))
		s := traceState(t, e)
		e.Close()
		t.Logf("input=%x cursor=%+v reply=%q", input, s.Cursor, s.Replies)
		if s.Cursor.X != 0 || s.Cursor.Y != 0 || s.Replies != "\x1b[1;1R" {
			t.Fatal("native origin CPR expectation changed")
		}
	}
}

func TestNativeHistoryDistinguishesHolesAndSpaces(t *testing.T) {
	for _, test := range []struct {
		name, input, leading string
		bg                   uint32
	}{
		{"cursor-hole", "\x1b[3GA", "", 0},
		{"literal-spaces", "  A", " ", 0},
		{"background-hole", "\x1b[41m\x1b[2J\x1b[3GA\x1b[0m", "", uint32(phosphor.IndexedColor(1))},
		{"hyperlink-hole", "\x1b]8;;https://example.com\x1b\\\x1b[3GA\x1b]8;;\x1b\\", "", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := newTracedEngine("ghostty", 8, 2)
			defer e.Close()
			e.Write([]byte(test.input + "\r\nB\r\nC"))
			s := traceState(t, e)
			if len(s.History) != 1 {
				t.Fatalf("history rows=%d", len(s.History))
			}
			a, b, c := s.History[0][0], s.History[0][1], s.History[0][2]
			t.Logf("leading=%+v,%+v text=%+v", a, b, c)
			if a.Text != test.leading || b.Text != test.leading || a.Bg != test.bg || b.Bg != test.bg || a.URI != "" || b.URI != "" || c.Text != "A" {
				t.Fatal("native history holes/space expectation changed")
			}
		})
	}
}

func TestNativeStringCancellation(t *testing.T) {
	for _, prefix := range []string{"\x1b]2;cancelled", "\x1b]8;;https://cancelled.example", "\x1bP$qm", "\x1bP1;2zignored", "\x1bXignored", "\x1b^ignored", "\x1b_ignored"} {
		for _, cancel := range []byte{0x18, 0x1a} {
			t.Run(fmt.Sprintf("prefix-%x-cancel-%x", prefix, cancel), func(t *testing.T) {
				e := newTracedEngine("ghostty", 12, 3)
				defer e.Close()
				e.Write([]byte("\x1b]2;before\a" + prefix + string(cancel) + "X"))
				s := traceState(t, e)
				var text string
				for _, c := range s.Screen[0] {
					text += c.Text
				}
				t.Logf("title=%q text=%q replies=%q link=%q", s.Title, text, s.Replies, s.Screen[0][0].URI)
				wantReply := ""
				if prefix == "\x1bP$qm" {
					wantReply = "\x1bP1$r0m\x1b\\"
				}
				if s.Title != "before" || text != "X" || s.Replies != wantReply || s.Screen[0][0].URI != "" {
					t.Fatal("native cancellation contract differs")
				}
			})
		}
	}
}
func TestTraceDecodingRejectsMalformedInput(t *testing.T) {
	valid := traceCase{Name: "test", Feature: "cells", Scope: "parity", Source: "test", Cols: 8, Rows: 3, Actions: []traceAction{{Kind: "write", Data: "x"}}}
	for _, test := range []struct {
		name   string
		mutate func(*traceCase)
	}{
		{"zero-cols", func(c *traceCase) { c.Cols = 0 }},
		{"oversized-rows", func(c *traceCase) { c.Rows = 129 }},
		{"scope", func(c *traceCase) { c.Scope = "skip" }},
		{"action", func(c *traceCase) { c.Actions = []traceAction{{Kind: "clean"}} }},
		{"mixed-write", func(c *traceCase) { c.Actions = []traceAction{{Kind: "write", Data: "x", Hex: "42"}} }},
		{"invalid-hex", func(c *traceCase) { c.Actions = []traceAction{{Kind: "write", Hex: "0"}} }},
		{"resize-payload", func(c *traceCase) { c.Actions = []traceAction{{Kind: "resize", Cols: 2, Rows: 2, Data: "x"}} }},
		{"checkpoint-payload", func(c *traceCase) { c.Actions = []traceAction{{Kind: "snapshot", Data: "x"}} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := valid
			test.mutate(&c)
			b, _ := json.Marshal([]traceCase{c})
			if _, err := decodeTraces(strings.NewReader(string(b))); err == nil {
				t.Fatal("accepted malformed case")
			}
		})
	}
	b, _ := json.Marshal([]traceCase{valid})
	for _, input := range []string{string(b) + " []", strings.Replace(string(b), `"name":`, `"extra":true,"name":`, 1), "[" + string(b[1:len(b)-1]) + "," + string(b[1:len(b)-1]) + "]", strings.Repeat(" ", (2<<20)+1) + string(b)} {
		if _, err := decodeTraces(strings.NewReader(input)); err == nil {
			t.Fatal("accepted unknown/duplicate/trailing/oversized input")
		}
	}
}
func TestTraceFixturesCoverTextModel(t *testing.T) {
	f, err := os.Open("traces.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cases, err := decodeTraces(f)
	if err != nil {
		t.Fatal(err)
	}
	features := map[string]bool{}
	actions := map[string]bool{}
	scopes := map[string]bool{}
	for _, c := range cases {
		features[c.Feature] = true
		scopes[c.Scope] = true
		for _, a := range c.Actions {
			actions[a.Kind] = true
		}
	}
	for _, f := range []string{"cells-wrap", "cell-style", "underline-color", "hyperlinks", "protection", "width-wrap", "graphemes", "graphemes-style", "cursor", "cursor-margins", "margins", "modes", "keyboard", "charsets", "rep", "editing", "tabs", "osc", "continuation", "screens", "history", "portable-repaint", "queries", "palette", "semantic-prompts", "columns"} {
		if !features[f] {
			t.Errorf("missing feature %s", f)
		}
	}
	for _, a := range []string{"write", "resize", "compact", "snapshot", "repaint"} {
		if !actions[a] {
			t.Errorf("missing action %s", a)
		}
	}
	for _, s := range []string{"parity", "identity", "deliberate"} {
		if !scopes[s] {
			t.Errorf("missing scope %s", s)
		}
	}
}
