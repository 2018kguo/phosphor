//go:build ghostty

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestComparisonSampleMetadata(t *testing.T) {
	if os.Getenv("ORACLE_METADATA_TEST_CHILD") == "1" {
		emit(map[string]any{"kind": "test"})
		return
	}
	for _, tc := range []struct {
		repeat, position string
		valid            bool
	}{
		{"", "", true}, {"0", "0", true}, {"5", "2", true},
		{"-1", "0", false}, {"bad", "1", false}, {"0", "3", false}, {"0", "-1", false},
	} {
		t.Run(tc.repeat+"/"+tc.position, func(t *testing.T) {
			t.Setenv("ORACLE_METADATA_TEST_CHILD", "1")
			t.Setenv("COMPARE_OUTER_REPEAT", tc.repeat)
			t.Setenv("COMPARE_SAMPLE_ORDER", tc.position)
			cmd := exec.Command(os.Args[0], "-test.run=^TestComparisonSampleMetadata$")
			output, err := cmd.CombinedOutput()
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v exit=%v output=%s", tc.valid, err, output)
			}
			if !tc.valid {
				return
			}
			var record map[string]any
			if err := json.NewDecoder(strings.NewReader(string(output))).Decode(&record); err != nil {
				t.Fatal(err)
			}
			for _, field := range []struct{ key, raw string }{{"outer_repeat", tc.repeat}, {"sample_order", tc.position}} {
				value, present := record[field.key]
				if field.raw == "" {
					if present {
						t.Fatalf("unconfigured %s acquired a value", field.key)
					}
				} else {
					want, _ := strconv.Atoi(field.raw)
					if value != float64(want) {
						t.Fatalf("%s=%v, want%d", field.key, value, want)
					}
				}
			}
		})
	}
}

func TestABRunnerBalancesPositionsAndRecordsRepeats(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "fake-driver")
	// The fake driver emits only configuration: this test never measures a core.
	const script = `#!/bin/sh
printf '{"stream":"%s","label":"%s","outer_repeat":%s,"sample_order":%s}\n' "$COMPARE_STREAM" "$COMPARE_LABEL" "$COMPARE_OUTER_REPEAT" "$COMPARE_SAMPLE_ORDER"
`
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PHOSPHOR_DIR", dir)
	t.Setenv("COMPARE_REPEATS", "")
	output := filepath.Join(dir, "samples.jsonl")
	if result, err := exec.Command("bash", "ab.sh", binary, binary, output).CombinedOutput(); err != nil {
		t.Fatalf("runner: %v %s", err, result)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	type sample struct {
		Stream, Label string
		Repeat        int `json:"outer_repeat"`
		Order         int `json:"sample_order"`
	}
	positions := map[string]map[string][3]int{}
	permutations := map[string]map[int][]string{}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		var row sample
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		if row.Repeat < 0 || row.Repeat > 5 || row.Order < 0 || row.Order > 2 {
			t.Fatalf("invalid sample position: %+v", row)
		}
		if positions[row.Stream] == nil {
			positions[row.Stream] = map[string][3]int{}
			permutations[row.Stream] = map[int][]string{}
		}
		counts := positions[row.Stream][row.Label]
		counts[row.Order]++
		positions[row.Stream][row.Label] = counts
		if row.Order != len(permutations[row.Stream][row.Repeat]) {
			t.Fatal("sample positions do not preserve execution order")
		}
		permutations[row.Stream][row.Repeat] = append(permutations[row.Stream][row.Repeat], row.Label)
	}
	if len(lines) != 180 || len(positions) != 10 {
		t.Fatalf("runner emitted %d samples across %d streams", len(lines), len(positions))
	}
	for stream, labels := range positions {
		for _, label := range []string{"baseline", "phosphor", "ghostty"} {
			if labels[label] != ([3]int{2, 2, 2}) {
				t.Fatalf("%s/%s has unbalanced positions: %v", stream, label, labels[label])
			}
		}
		seen := map[string]bool{}
		for _, order := range permutations[stream] {
			seen[strings.Join(order, ",")] = true
		}
		if len(seen) != 6 {
			t.Fatalf("%s repeats an order before all six permutations", stream)
		}
	}
	t.Setenv("COMPARE_REPEATS", "2")
	if result, err := exec.Command("bash", "ab.sh", binary, binary, output).CombinedOutput(); err != nil {
		t.Fatalf("configured repeats: %v %s", err, result)
	}
	data, err = os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(strings.Split(strings.TrimSpace(string(data)), "\n")); got != 60 {
		t.Fatalf("two repeats emitted %d samples, want60", got)
	}
	for _, invalid := range []string{"0", "-1", "01", "not-a-number", "1000000000"} {
		t.Setenv("COMPARE_REPEATS", invalid)
		if err := exec.Command("bash", "ab.sh", binary, binary, output).Run(); err == nil {
			t.Fatalf("runner accepted invalid repeats %q", invalid)
		}
	}
}

func TestNativeBuildInfoLabelsExpectedSource(t *testing.T) {
	info := nativeBuildInfo()
	if info["expected_source_commit"] != pinnedGhosttyCommit {
		t.Fatal("native build info lost its expected pin")
	}
	if _, observed := info["source_commit"]; observed {
		t.Fatal("expected native source was reported as observed")
	}
	if _, verified := info["verified_checkout_commit"]; verified {
		t.Fatal("native build info reported an unverified checkout")
	}
}
