//go:build ghostty

package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/2018kguo/phosphor"
)

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
func TestNativeSnapshotReplyBinding(t *testing.T) {
	e := newTracedEngine("ghostty", 8, 3)
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
	for _, f := range []string{"cells-wrap", "cell-style", "underline-color", "hyperlinks", "protection", "width-wrap", "graphemes", "graphemes-style", "cursor", "cursor-margins", "margins", "modes", "keyboard", "charsets", "rep", "editing", "tabs", "osc", "continuation", "screens", "history", "portable-repaint"} {
		if !features[f] {
			t.Errorf("missing feature %s", f)
		}
	}
	for _, a := range []string{"write", "resize", "compact", "snapshot", "repaint"} {
		if !actions[a] {
			t.Errorf("missing action %s", a)
		}
	}
	for _, s := range []string{"parity", "identity", "deliberate", "limit"} {
		if !scopes[s] {
			t.Errorf("missing scope %s", s)
		}
	}
}
