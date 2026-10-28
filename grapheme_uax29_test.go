package phosphor

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// This exercises the actual segmentation helper against every Unicode 17
// boundary in our dependency's conformance corpus. It does not test terminal
// dispatch, physical width, control execution or the 32-byte storage limit.
func TestGraphemeUAX29Corpus(t *testing.T) {
	corpus := loadUAX29Corpus(t)
	type boundary struct {
		prefix string
		next   rune
	}
	// The terminal executes C0 controls before segmentation. Like Ghostty,
	// our helper treats any stored grapheme-Control scalar as Other when
	// clustering later resumes. These are the exact affected corpus pairs;
	// keep them visible instead of dropping their cases from the denominator.
	tailored := map[boundary]bool{
		{"\x00", 0x0308}: false, {"\x00", 0x094d}: false, {"\x00", 0x0300}: false,
		{"\x00", 0x200c}: false, {"\x00", 0x200d}: false, {"\x00", 0x0903}: false,
		{"\u06dd", 0}: false,
	}
	total, official, tailoring := 0, 0, 0
	for ci, c := range corpus {
		boundaries := map[int]bool{}
		pos := 0
		for _, part := range c.expected {
			boundaries[pos] = true
			pos += len(part)
		}
		prefix := ""
		for offset, r := range string(c.input) {
			if offset == 0 {
				prefix = string(r)
				continue
			}
			_, got := graphemeContinues(nil, prefix, r)
			want := !boundaries[offset]
			total++
			key := boundary{prefix, r}
			if _, ok := tailored[key]; ok {
				if !got || want {
					t.Fatalf("case%d byte%d: control tailoring changed: %q + %U, join%v Unicodejoin%v", ci, offset, prefix, r, got, want)
				}
				tailored[key] = true
				tailoring++
			} else {
				if got != want {
					t.Fatalf("case%d byte%d: %q + %U join%v want%v", ci, offset, prefix, r, got, want)
				}
				official++
			}
			if got {
				prefix += string(r)
			} else {
				prefix = string(r)
			}
		}
	}
	for pair, seen := range tailored {
		if !seen {
			t.Fatalf("unexercised control tailoring:%+v", pair)
		}
	}
	if len(corpus) != 766 || total != 1224 || official != 1199 || tailoring != 25 {
		t.Fatalf("Unicode corpus coverage changed: cases%d boundaries%d official%d tailored%d", len(corpus), total, official, tailoring)
	}
	t.Logf("Unicode17 helper corpus: %d cases, %d boundaries, %d official matches, %d explicit control-tailoring differences", len(corpus), total, official, tailoring)
}

type uax29Case struct {
	input    []byte
	expected [][]byte
}

// The dependency keeps the official corpus as Go literals. Read those
// literals with Go's parser, retaining the dependency's pinned source and
// license instead of maintaining another copy of the same expected data.
func loadUAX29Corpus(t *testing.T) []uax29Case {
	t.Helper()
	dir, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/clipperhouse/uax29/v2").Output()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(strings.TrimSpace(string(dir)), "graphemes", "unicode_test.go")
	source, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var literal *ast.CompositeLit
	ast.Inspect(source, func(n ast.Node) bool {
		v, ok := n.(*ast.ValueSpec)
		if ok && len(v.Names) == 1 && v.Names[0].Name == "unicodeTests" && len(v.Values) == 1 {
			literal, _ = v.Values[0].(*ast.CompositeLit)
		}
		return true
	})
	if literal == nil {
		t.Fatal("upstream Unicode corpus no longer has a unicodeTests literal")
	}
	bytesOf := func(expr ast.Expr) []byte {
		t.Helper()
		list, ok := expr.(*ast.CompositeLit)
		if !ok {
			t.Fatalf("upstream Unicode byte literal changed:%T", expr)
		}
		out := make([]byte, len(list.Elts))
		for i, e := range list.Elts {
			value, ok := e.(*ast.BasicLit)
			if !ok {
				t.Fatalf("upstream Unicode byte is not a literal:%T", e)
			}
			n, err := strconv.ParseUint(value.Value, 0, 8)
			if err != nil {
				t.Fatal(err)
			}
			out[i] = byte(n)
		}
		return out
	}
	corpus := make([]uax29Case, len(literal.Elts))
	for i, expr := range literal.Elts {
		fields, ok := expr.(*ast.CompositeLit)
		if !ok {
			t.Fatalf("upstream Unicode case%d changed:%T", i, expr)
		}
		for _, field := range fields.Elts {
			kv, ok := field.(*ast.KeyValueExpr)
			if !ok {
				t.Fatalf("upstream Unicode case%d field changed:%T", i, field)
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				t.Fatalf("upstream Unicode case%d key changed:%T", i, kv.Key)
			}
			switch key.Name {
			case "input":
				corpus[i].input = bytesOf(kv.Value)
			case "expected":
				parts, ok := kv.Value.(*ast.CompositeLit)
				if !ok {
					t.Fatalf("upstream Unicode expected%d changed:%T", i, kv.Value)
				}
				for _, part := range parts.Elts {
					corpus[i].expected = append(corpus[i].expected, bytesOf(part))
				}
			case "comment":
			default:
				t.Fatalf("upstream Unicode case%d unknownfield%s", i, key.Name)
			}
		}
		if len(corpus[i].input) == 0 || !bytes.Equal(corpus[i].input, bytes.Join(corpus[i].expected, nil)) {
			t.Fatalf("upstream Unicode case%d lost data", i)
		}
	}
	return corpus
}
