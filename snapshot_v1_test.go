package phosphor

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// These files were emitted by released v0.3.0, before the v2 writer existed.
// Expected observations were collected from that version after each suffix.
func TestReleasedSnapshotV1(t *testing.T) {
	type observation struct {
		Cells   [][]Cell
		History []string
		Cursor  Cursor
		Modes   Modes
		Palette Palette
		Title   []byte
		Reply   string
	}
	files, err := filepath.Glob("testdata/snapshot-v1/*.bin")
	if err != nil || len(files) != 6 {
		t.Fatalf("released fixture inventory: %v, %v", files, err)
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			read := func(path string) []byte {
				b, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				return b
			}
			stem := strings.TrimSuffix(file, ".bin")
			var want observation
			if err := json.Unmarshal(read(stem+".json"), &want); err != nil {
				t.Fatal(err)
			}
			var replies bytes.Buffer
			term := New(1, 1, WithReply(&replies))
			if err := term.RestoreSnapshot(read(file), SnapshotLimits{}); err != nil {
				t.Fatal(err)
			}
			term.Write(read(stem + ".suffix"))
			got := observation{Cursor: term.Cursor(), Modes: term.Modes(), Palette: term.Palette(), Title: []byte(term.Title()), Reply: replies.String()}
			cols, rows := term.Size()
			for y := range rows {
				row := make([]Cell, cols)
				for x := range cols {
					row[x] = term.Cell(x, y)
				}
				got.Cells = append(got.Cells, row)
			}
			for i := range term.ScrollbackLen() {
				got.History = append(got.History, term.ScrollbackLine(i))
			}
			for i := range reflect.TypeOf(got).NumField() {
				left, right := reflect.ValueOf(got).Field(i).Interface(), reflect.ValueOf(want).Field(i).Interface()
				if !reflect.DeepEqual(left, right) {
					t.Errorf("released v1 %s changed: got%+v want%+v", reflect.TypeOf(got).Field(i).Name, left, right)
				}
			}
		})
	}
}
