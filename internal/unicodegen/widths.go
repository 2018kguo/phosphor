package main

import (
	"bytes"
	"encoding/json"
	"io"
)

// The native API enumerates every scalar. Its final coverage record prevents
// a clipped capture or another native revision from changing terminal width.
func nativeWidths(data []byte, commit, version string) (widths [0x110000]uint8) {
	for i := range widths {
		widths[i] = 1
	}
	type record struct {
		Kind          string `json:"kind"`
		First, Last   int
		Width         int    `json:"ghostty_width"`
		Commit        string `json:"ghostty_commit"`
		Version       string `json:"unicode_version"`
		Scalars       int    `json:"unicode_scalars"`
		Default       int    `json:"default_width"`
		ExcludedFirst int    `json:"excluded_first"`
		ExcludedLast  int    `json:"excluded_last"`
		NonDefault    int    `json:"non_default_scalars"`
	}
	complete, previous, nonDefault := false, -1, 0
	d := json.NewDecoder(bytes.NewReader(data))
	for {
		var row record
		err := d.Decode(&row)
		if err == io.EOF {
			break
		}
		must(err)
		if complete {
			panic("data follows native width summary")
		}
		switch row.Kind {
		case "native_scalar_width_range":
			if row.First < 0 || row.Last > 0x10ffff || row.First > row.Last || row.Width != 0 && row.Width != 2 || row.First <= 0xdfff && row.Last >= 0xd800 || previous >= row.First {
				panic("invalid or overlapping native scalar width range")
			}
			for cp := row.First; cp <= row.Last; cp++ {
				widths[cp] = uint8(row.Width)
			}
			previous = row.Last
			nonDefault += row.Last - row.First + 1
		case "native_scalar_width_summary":
			if row.Commit != commit || row.Version != version || row.Scalars != 1112064 || row.First != 0 || row.Last != 0x10ffff || row.Default != 1 || row.ExcludedFirst != 0xd800 || row.ExcludedLast != 0xdfff || row.NonDefault != nonDefault {
				panic("unexpected native Unicode width coverage")
			}
			complete = true
		default:
			panic("unexpected native Unicode width record: " + row.Kind)
		}
	}
	if !complete || previous < 0 {
		panic("incomplete native Unicode width stream")
	}
	return widths
}
