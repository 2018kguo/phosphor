package phosphor

import (
	"encoding/json"
	"io"
	"os"
	"testing"
	"unicode/utf8"
)

func TestCodepointWidthsMatchAuditedNativeStream(t *testing.T) {
	f, err := os.Open("internal/unicodegen/data/native-widths.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	next, checked, nonDefault, summaries := 0, 0, 0, 0
	check := func(first, last, want int) {
		t.Helper()
		for cp := first; cp <= last; cp++ {
			if got := CodepointWidth(rune(cp)); got != want {
				t.Fatalf("%U width %d, audited native width %d", cp, got, want)
			}
			checked++
		}
	}
	for {
		var row struct {
			Kind        string `json:"kind"`
			First, Last int
			Width       int    `json:"ghostty_width"`
			Scalars     int    `json:"unicode_scalars"`
			NonDefault  int    `json:"non_default_scalars"`
			Commit      string `json:"ghostty_commit"`
			Version     string `json:"unicode_version"`
		}
		if err := decoder.Decode(&row); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if summaries != 0 {
			t.Fatal("data follows native summary")
		}
		switch row.Kind {
		case "native_scalar_width_range":
			if row.First < next || row.Last < row.First || row.Last > utf8.MaxRune || row.Width != 0 && row.Width != 2 {
				t.Fatal("invalid native width range")
			}
			check(next, row.First-1, 1)
			check(row.First, row.Last, row.Width)
			next = row.Last + 1
			nonDefault += row.Last - row.First + 1
		case "native_scalar_width_summary":
			check(next, utf8.MaxRune, 1)
			if checked != 0x110000 || row.Scalars != 1112064 || row.NonDefault != nonDefault || row.Commit != "6ba144158577d033ccb6ed4b574e4013957c9161" || row.Version != "18.0.0" {
				t.Fatal("native scalar coverage changed")
			}
			summaries++
		default:
			t.Fatal("unknown native width record")
		}
	}
	if summaries != 1 {
		t.Fatal("missing native scalar coverage")
	}
	for _, cp := range []rune{-1, utf8.MaxRune + 1, clusterBase} {
		if CodepointWidth(cp) != 1 || emojiVariationBase(cp) || graphemeSimpleScalar(cp) {
			t.Fatalf("invalid scalar %U changed properties", cp)
		}
	}
}
